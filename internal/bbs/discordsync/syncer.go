/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package discordsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/uuidv7"
)

const (
	MaxForwardAnchorEditsPerPost = 5
	MaxFanoutConcurrency         = 4
)

// ErrSourceMountInactive indicates the originating mount was deactivated or missing during publish.
var ErrSourceMountInactive = errors.New("discordsync: source board mount is inactive")

type anchorItem struct {
	client  *bot.Client
	guildID snowflake.ID
	thread  bbs.Thread
	resNums []uint32
	locale  discord.Locale
}

type anchorQueue struct {
	running bool
	jobs    []anchorItem
}

// Syncer bridges BBS domain state with Discord channels, threads, and multi-guild mirrors.
type Syncer struct {
	Service    *bbs.Service
	Validator  PayloadValidator
	Debouncer  *ControlPanelDebouncer
	DMCache    *DMChannelCache
	Guard      *InvalidRequestGuard
	CanSendDM  func(ctx context.Context, userID snowflake.ID) bool
	UserLocale func(ctx context.Context, userID snowflake.ID) discord.Locale

	anchorMu        sync.Mutex
	anchorQ         map[snowflake.ID]*anchorQueue
	threadEditLocks [64]sync.Mutex
}

func (s *Syncer) threadEditMutex(discordThreadID snowflake.ID) *sync.Mutex {
	v := uint64(discordThreadID)
	v ^= v >> 22
	idx := v % uint64(len(s.threadEditLocks))
	return &s.threadEditLocks[idx]
}

// ResolveGuildLocale returns the guild's preferred Discord locale from cache, falling back cleanly.
func ResolveGuildLocale(client *bot.Client, guildID snowflake.ID, fallback discord.Locale) discord.Locale {
	if client != nil && guildID != 0 {
		if g, ok := client.Caches.Guild(guildID); ok && g.PreferredLocale != "" {
			return discord.Locale(g.PreferredLocale)
		}
	}
	if fallback != "" && fallback != discord.LocaleUnknown {
		return fallback
	}
	return discord.LocaleJapanese
}

// ResolveMountLocale returns the effective locale for a board mount.
func ResolveMountLocale(client *bot.Client, mount bbs.BoardMount) discord.Locale {
	var fb discord.Locale
	if mount.Locale != "" {
		fb = discord.Locale(mount.Locale)
	}
	return ResolveGuildLocale(client, mount.GuildID, fb)
}

// ResolveUserLocaleFromDB looks up the user's saved locale in gobot's core users table.
func ResolveUserLocaleFromDB(ctx context.Context, gdb *database.DB, userID snowflake.ID) discord.Locale {
	if gdb == nil || gdb.DB == nil {
		return discord.LocaleJapanese
	}
	u, err := database.GetOrCreateUser(gdb.DB.WithContext(ctx), userID)
	if err != nil || u.Locale == "" || u.Locale == discord.LocaleUnknown {
		return discord.LocaleJapanese
	}
	return u.Locale
}

// NewSyncer initializes a Syncer with default safety guards.
func NewSyncer(svc *bbs.Service) *Syncer {
	return &Syncer{
		Service:   svc,
		Validator: PayloadValidator{},
		Debouncer: NewControlPanelDebouncer(DefaultDebouncerDelay),
		DMCache:   NewDMChannelCache(DefaultDMCooldown),
		Guard:     NewInvalidRequestGuard(DefaultInvalidReqWindow, DefaultInvalidReqThreshold, DefaultMountCircuitFails),
		anchorQ:   make(map[snowflake.ID]*anchorQueue),
	}
}

// ProvisionBoardChannel creates a Forum or Text channel for a board and mounts it for guildID.
func (s *Syncer) ProvisionBoardChannel(
	ctx context.Context,
	client *bot.Client,
	guildID snowflake.ID,
	board bbs.Board,
	categoryDiscordID *snowflake.ID,
	wantForum bool,
	viewerRoleID *snowflake.ID,
	locale discord.Locale,
) (*bbs.BoardMount, ChannelCreatePlan, error) {
	if existing, err := s.Service.Store.GetBoardMountByGuildAndBoard(ctx, guildID, board.ID); err == nil && existing != nil {
		return existing, ChannelCreatePlan{UseForum: existing.IsForum}, bbs.ErrAlreadyExists
	}

	resolvedLocale := ResolveGuildLocale(client, guildID, locale)
	channelName := i18n.TranslateText(resolvedLocale, "components.bbs.sync.channel_name", map[string]any{
		"name": board.Name,
	})
	checker := NewPreflightChecker(NewClientGuildStateReader(client))
	plan, err := checker.CheckChannelCreate(guildID, channelName, board.Description, wantForum, categoryDiscordID)
	if err != nil {
		return nil, plan, err
	}

	overwrites := buildBoardOverwrites(guildID, client.ApplicationID, viewerRoleID)

	var parentID snowflake.ID
	if categoryDiscordID != nil {
		parentID = *categoryDiscordID
	}

	var ch discord.GuildChannel
	if plan.UseForum {
		ch, err = client.Rest.CreateGuildChannel(guildID, discord.GuildForumChannelCreate{
			Name:                 plan.Name,
			Topic:                plan.Topic,
			ParentID:             parentID,
			PermissionOverwrites: overwrites,
		})
	} else {
		ch, err = client.Rest.CreateGuildChannel(guildID, discord.GuildTextChannelCreate{
			Name:                 plan.Name,
			Topic:                plan.Topic,
			ParentID:             parentID,
			PermissionOverwrites: overwrites,
		})
	}
	if err != nil {
		return nil, plan, err
	}

	mount := &bbs.BoardMount{
		ID:        uuidv7.New(),
		BoardID:   board.ID,
		GuildID:   guildID,
		ChannelID: ch.ID(),
		Locale:    string(resolvedLocale),
		IsForum:   plan.UseForum,
		Active:    true,
		CreatedAt: time.Now(),
	}

	guideMsg := RenderBoardGuide(board, mount.ID, resolvedLocale)
	if err := s.Validator.ValidateMessageCreate(guideMsg); err != nil {
		if delErr := client.Rest.DeleteChannel(ch.ID()); delErr != nil {
			slog.Warn("bbs failed to cleanup orphan channel after guide validation error", "channel_id", ch.ID(), "err", delErr)
		}
		return nil, plan, err
	}

	if plan.UseForum {
		var tags []snowflake.ID
		if fc, ok := ch.(discord.GuildForumChannel); ok {
			tags, _ = ResolveForumTags(fc)
		}
		post, err := client.Rest.CreatePostInThreadChannel(ch.ID(), discord.ThreadChannelPostCreate{
			Name:        plan.Name,
			Message:     guideMsg,
			AppliedTags: tags,
		})
		if err != nil {
			if delErr := client.Rest.DeleteChannel(ch.ID()); delErr != nil {
				slog.Warn("bbs failed to cleanup orphan forum channel", "channel_id", ch.ID(), "err", delErr)
			}
			return nil, plan, err
		}
		pinnedID := post.ID()
		mount.PinnedPostID = &pinnedID
		flags := discord.ChannelFlagPinned
		if _, pinErr := client.Rest.UpdateChannel(post.ID(), discord.GuildPostUpdate{Flags: &flags}); pinErr != nil {
			slog.Warn("bbs failed to pin board guide post", "channel_id", ch.ID(), "err", pinErr)
		}
	} else {
		msg, err := client.Rest.CreateMessage(ch.ID(), guideMsg)
		if err != nil {
			if delErr := client.Rest.DeleteChannel(ch.ID()); delErr != nil {
				slog.Warn("bbs failed to cleanup orphan text channel", "channel_id", ch.ID(), "err", delErr)
			}
			return nil, plan, err
		}
		mount.ControlMessageID = &msg.ID
	}

	if err := s.Service.Store.CreateBoardMount(ctx, mount); err != nil {
		if existing, gErr := s.Service.Store.GetBoardMountByChannel(ctx, ch.ID()); gErr == nil && existing != nil {
			return existing, plan, nil
		} else if errors.Is(gErr, bbs.ErrNotFound) {
			if delErr := client.Rest.DeleteChannel(ch.ID()); delErr != nil {
				slog.Warn("bbs failed to cleanup orphan channel after DB failure", "channel_id", ch.ID(), "err", delErr)
			}
		}
		return nil, plan, err
	}
	return mount, plan, nil
}

func buildBoardOverwrites(guildID, botID snowflake.ID, viewerRoleID *snowflake.ID) []discord.PermissionOverwrite {
	everyoneDeny := discord.PermissionSendMessages.Add(
		discord.PermissionSendMessagesInThreads,
		discord.PermissionCreatePublicThreads,
		discord.PermissionCreatePrivateThreads,
	)
	if viewerRoleID != nil && *viewerRoleID != 0 {
		everyoneDeny = everyoneDeny.Add(discord.PermissionViewChannel)
	}
	overwrites := []discord.PermissionOverwrite{
		discord.RolePermissionOverwrite{
			RoleID: guildID,
			Deny:   everyoneDeny,
		},
		discord.MemberPermissionOverwrite{
			UserID: botID,
			Allow: discord.PermissionViewChannel.Add(
				discord.PermissionSendMessages,
				discord.PermissionSendMessagesInThreads,
				discord.PermissionCreatePublicThreads,
				discord.PermissionCreatePrivateThreads,
				discord.PermissionManageThreads,
				discord.PermissionManageMessages,
				discord.PermissionReadMessageHistory,
			),
		},
	}
	if viewerRoleID != nil && *viewerRoleID != 0 {
		overwrites = append(overwrites, discord.RolePermissionOverwrite{
			RoleID: *viewerRoleID,
			Allow:  discord.PermissionViewChannel.Add(discord.PermissionReadMessageHistory),
		})
	}
	return overwrites
}

// PublishThread creates Discord mirrors across the source mount (synchronously) and other mounts (asynchronously).
// A failure or inactive state on the source mount does not prevent fanout to other active mounts.
func (s *Syncer) PublishThread(
	ctx context.Context,
	client *bot.Client,
	sourceMountID uuid.UUID,
	board bbs.Board,
	thread bbs.Thread,
	firstPost bbs.Post,
	locale discord.Locale,
) (snowflake.ID, snowflake.ID, error) {
	mounts, err := s.Service.Store.ListActiveBoardMountsByBoard(ctx, board.ID)
	if err != nil {
		return 0, 0, err
	}

	var srcChannelID, srcMsgID snowflake.ID
	var srcErr error
	var sourceMatched bool
	var otherMounts []bbs.BoardMount
	for _, m := range mounts {
		if m.ID == sourceMountID {
			sourceMatched = true
			chID, mID, err := s.mirrorNewThreadToMount(ctx, client, m, board, thread, firstPost, locale)
			if err != nil {
				srcErr = err
			} else {
				srcChannelID, srcMsgID = chID, mID
			}
		} else {
			otherMounts = append(otherMounts, m)
		}
	}
	if !sourceMatched && srcErr == nil {
		srcErr = ErrSourceMountInactive
	}

	if len(otherMounts) > 0 {
		go s.fanout(otherMounts, func(m bbs.BoardMount) {
			if _, _, err := s.mirrorNewThreadToMount(s.backgroundCtx(), client, m, board, thread, firstPost, locale); err != nil {
				slog.Warn("bbs fanout thread mirror failed", "mount_id", m.ID, "guild_id", m.GuildID, "err", err)
			}
		})
	}
	if srcErr != nil {
		return 0, 0, srcErr
	}
	return srcChannelID, srcMsgID, nil
}

func (s *Syncer) mirrorNewThreadToMount(
	ctx context.Context,
	client *bot.Client,
	mount bbs.BoardMount,
	board bbs.Board,
	thread bbs.Thread,
	firstPost bbs.Post,
	locale discord.Locale,
) (snowflake.ID, snowflake.ID, error) {
	allowed, probeSeq, release := s.Guard.AcquireMountWithProbe(mount.ID, time.Now())
	if !allowed {
		return 0, 0, errors.New("sync skipped due to circuit breaker or invalid-request throttle")
	}
	defer release()
	return s.mirrorNewThreadToMountLocked(ctx, client, mount, board, thread, firstPost, locale, probeSeq)
}

func (s *Syncer) mirrorNewThreadToMountLocked(
	ctx context.Context,
	client *bot.Client,
	mount bbs.BoardMount,
	board bbs.Board,
	thread bbs.Thread,
	firstPost bbs.Post,
	locale discord.Locale,
	probeSeq uint64,
) (snowflake.ID, snowflake.ID, error) {
	mountLocale := ResolveMountLocale(client, mount)
	if mount.Locale == "" && locale != "" && locale != discord.LocaleUnknown {
		mountLocale = locale
	}

	msgCreate := RenderPostMessageCreate(firstPost, thread, nil, mountLocale)
	if err := s.Validator.ValidateMessageCreate(msgCreate); err != nil {
		return 0, 0, err
	}
	title := TruncateRunes(thread.Title, MaxChannelNameRunes)

	var discordThreadID, firstMsgID snowflake.ID
	var createdTextThread bool
	if mount.IsForum {
		var tags []snowflake.ID
		if fc, ok := client.Caches.GuildForumChannel(mount.ChannelID); ok {
			var tErr error
			tags, tErr = ResolveForumTags(fc)
			if tErr != nil {
				return 0, 0, tErr
			}
		}
		tp, err := client.Rest.CreatePostInThreadChannel(mount.ChannelID, discord.ThreadChannelPostCreate{
			Name:                title,
			AutoArchiveDuration: discord.AutoArchiveDuration24h,
			Message:             msgCreate,
			AppliedTags:         tags,
		})
		if err != nil {
			s.handleMountRESTError(ctx, mount.ID, probeSeq, err)
			return 0, 0, err
		}
		discordThreadID = tp.ID()
		firstMsgID = tp.Message.ID
		if firstMsgID == 0 {
			firstMsgID = tp.ID()
		}
	} else {
		gt, err := client.Rest.CreateThread(mount.ChannelID, discord.GuildPublicThreadCreate{
			Name:                title,
			AutoArchiveDuration: discord.AutoArchiveDuration24h,
		})
		if err != nil {
			s.handleMountRESTError(ctx, mount.ID, probeSeq, err)
			return 0, 0, err
		}
		discordThreadID = gt.ID()
		m, err := client.Rest.CreateMessage(discordThreadID, msgCreate)
		if err != nil {
			s.Guard.RecordError(uuid.Nil, err, time.Now())
			if delErr := client.Rest.DeleteChannel(discordThreadID); delErr != nil {
				slog.Warn("bbs failed to cleanup orphan discord thread after OP send failure", "discord_thread_id", discordThreadID, "err", delErr)
			}
			return 0, 0, err
		}
		firstMsgID = m.ID
		createdTextThread = true
	}

	s.Guard.RecordMountSuccess(mount.ID)

	mirror := &bbs.ThreadMirror{
		ID:              uuidv7.New(),
		ThreadID:        thread.ID,
		BoardMountID:    mount.ID,
		GuildID:         mount.GuildID,
		DiscordThreadID: discordThreadID,
		SyncedUpToRes:   firstPost.ResNumber,
		CreatedAt:       time.Now(),
	}
	postMirror := &bbs.PostMirror{
		ID:               uuidv7.New(),
		PostID:           firstPost.ID,
		ThreadMirrorID:   mirror.ID,
		GuildID:          mount.GuildID,
		DiscordMessageID: firstMsgID,
		CreatedAt:        time.Now(),
	}
	if err := s.Service.Store.CreateThreadAndPostMirror(ctx, mirror, postMirror); err != nil {
		if delErr := client.Rest.DeleteChannel(discordThreadID); delErr != nil {
			slog.Warn("bbs failed to cleanup orphan discord thread after mirror save error", "discord_thread_id", discordThreadID, "err", delErr)
		}
		return 0, 0, err
	}

	if createdTextThread {
		s.repositionBoardTextGuide(ctx, client, mount, board, mountLocale)
	}
	s.scheduleControlPanel(client, mirror, thread, mountLocale)
	return discordThreadID, firstMsgID, nil
}

// PublishPost mirrors a newly created reply to the source guild synchronously and other subscribed guilds asynchronously.
// Source mount failure or deactivation does not stop other mounts, watcher DMs, or archive locking.
func (s *Syncer) PublishPost(
	ctx context.Context,
	client *bot.Client,
	sourceMountID uuid.UUID,
	board bbs.Board,
	thread bbs.Thread,
	post bbs.Post,
	backwardRefs []uint32,
	locale discord.Locale,
) (snowflake.ID, snowflake.ID, error) {
	mounts, err := s.Service.Store.ListActiveBoardMountsByBoard(ctx, board.ID)
	if err != nil {
		return 0, 0, err
	}

	var srcGuildID, srcThreadID, srcMsgID snowflake.ID
	var srcErr error
	var sourceMatched bool
	var otherMounts []bbs.BoardMount
	for _, m := range mounts {
		if m.ID == sourceMountID {
			sourceMatched = true
			tID, mID, err := s.mirrorPostToMount(ctx, client, m, board, thread, post, backwardRefs, locale)
			if err != nil {
				srcErr = err
			} else {
				srcGuildID, srcThreadID, srcMsgID = m.GuildID, tID, mID
			}
		} else {
			otherMounts = append(otherMounts, m)
		}
	}
	if !sourceMatched && srcErr == nil {
		srcErr = ErrSourceMountInactive
	}

	go s.notifyWatchers(s.backgroundCtx(), client, board, thread, post, srcGuildID, srcThreadID, srcMsgID)

	if len(otherMounts) > 0 || thread.Status == bbs.ThreadStatusArchived {
		hitMaxRes := post.ResNumber >= thread.Rules.Normalize().MaxRes
		go func() {
			if len(otherMounts) > 0 {
				s.fanout(otherMounts, func(m bbs.BoardMount) {
					if _, _, err := s.mirrorPostToMount(s.backgroundCtx(), client, m, board, thread, post, backwardRefs, locale); err != nil {
						slog.Warn("bbs fanout post mirror failed", "mount_id", m.ID, "guild_id", m.GuildID, "err", err)
					}
				})
			}
			if thread.Status == bbs.ThreadStatusArchived {
				s.SyncArchive(s.backgroundCtx(), client, board, thread, hitMaxRes)
			}
		}()
	}

	if srcErr != nil {
		return 0, 0, srcErr
	}
	return srcThreadID, srcMsgID, nil
}

// EnsureThreadMirror lazily creates a Discord thread mirror on mount for a pre-existing thread.
func (s *Syncer) EnsureThreadMirror(
	ctx context.Context,
	client *bot.Client,
	mount bbs.BoardMount,
	board bbs.Board,
	thread bbs.Thread,
	excludeTrailingRes uint32,
	locale discord.Locale,
) (*bbs.ThreadMirror, error) {
	return s.ensureThreadMirror(ctx, client, mount, board, thread, excludeTrailingRes, locale, true, 0)
}

func (s *Syncer) ensureThreadMirror(
	ctx context.Context,
	client *bot.Client,
	mount bbs.BoardMount,
	board bbs.Board,
	thread bbs.Thread,
	excludeTrailingRes uint32,
	locale discord.Locale,
	acquireGuard bool,
	probeSeq uint64,
) (*bbs.ThreadMirror, error) {
	if existing, err := s.Service.Store.GetThreadMirrorByMountAndThread(ctx, mount.ID, thread.ID); err == nil {
		return existing, nil
	}
	posts, err := s.Service.ListPosts(ctx, thread.ID, 0, 1)
	if err != nil || len(posts) == 0 {
		return nil, fmt.Errorf("failed to load OP for lazy mirror: %w", err)
	}
	op := posts[0]
	if acquireGuard {
		if _, _, err := s.mirrorNewThreadToMount(ctx, client, mount, board, thread, op, locale); err != nil {
			return nil, err
		}
	} else {
		if _, _, err := s.mirrorNewThreadToMountLocked(ctx, client, mount, board, thread, op, locale, probeSeq); err != nil {
			return nil, err
		}
	}
	mirror, err := s.Service.Store.GetThreadMirrorByMountAndThread(ctx, mount.ID, thread.ID)
	if err != nil {
		return nil, err
	}

	// If pre-existing replies exist before excludeTrailingRes, insert the lazy-mirror catchup banner.
	lastPrior := thread.ResCount
	if excludeTrailingRes > 1 && excludeTrailingRes-1 < lastPrior {
		lastPrior = excludeTrailingRes - 1
	} else if excludeTrailingRes == 0 {
		lastPrior = thread.ResCount
	}
	if lastPrior >= 2 {
		mountLocale := ResolveMountLocale(client, mount)
		if mount.Locale == "" && locale != "" && locale != discord.LocaleUnknown {
			mountLocale = locale
		}
		banner := RenderLazyCatchupBanner(thread.ID, 2, lastPrior, mountLocale)
		if vErr := s.Validator.ValidateMessageCreate(banner); vErr == nil {
			sendErr := WithUnarchiveRetry(client, mirror.DiscordThreadID, func() error {
				_, err := client.Rest.CreateMessage(mirror.DiscordThreadID, banner)
				return err
			})
			if sendErr == nil {
				mirror.SyncedUpToRes = lastPrior
				_ = s.Service.Store.AdvanceThreadMirrorSync(ctx, mirror.ID, lastPrior)
			} else {
				s.Guard.RecordError(uuid.Nil, sendErr, time.Now())
			}
		}
	}
	return mirror, nil
}

func (s *Syncer) mirrorPostToMount(
	ctx context.Context,
	client *bot.Client,
	mount bbs.BoardMount,
	board bbs.Board,
	thread bbs.Thread,
	post bbs.Post,
	backwardRefs []uint32,
	locale discord.Locale,
) (snowflake.ID, snowflake.ID, error) {
	allowed, probeSeq, release := s.Guard.AcquireMountWithProbe(mount.ID, time.Now())
	if !allowed {
		return 0, 0, errors.New("sync skipped due to circuit breaker or invalid-request throttle")
	}
	defer release()

	mirror, err := s.ensureThreadMirror(ctx, client, mount, board, thread, post.ResNumber, locale, false, probeSeq)
	if err != nil {
		return 0, 0, err
	}

	mountLocale := ResolveMountLocale(client, mount)
	if mount.Locale == "" && locale != "" && locale != discord.LocaleUnknown {
		mountLocale = locale
	}

	resolveURL := s.guildURLResolver(ctx, thread.ID, mount.GuildID, mirror.DiscordThreadID)
	msgCreate := RenderPostMessageCreate(post, thread, resolveURL, mountLocale)
	if err := s.Validator.ValidateMessageCreate(msgCreate); err != nil {
		return 0, 0, err
	}

	var sent *discord.Message
	latestThread := thread
	var sendMirrorErr error
	func() {
		muLock := s.threadEditMutex(mirror.DiscordThreadID)
		muLock.Lock()
		defer muLock.Unlock()

		sendErr := WithUnarchiveRetry(client, mirror.DiscordThreadID, func() error {
			var cErr error
			sent, cErr = client.Rest.CreateMessage(mirror.DiscordThreadID, msgCreate)
			return cErr
		})
		if sendErr != nil {
			if IsJSONError(sendErr, JSONErrUnknownChannel) || HTTPStatusCode(sendErr) == 404 {
				s.Guard.RecordError(uuid.Nil, sendErr, time.Now())
				slog.Warn("bbs deleting missing child thread mirror after 404", "mirror_id", mirror.ID, "discord_thread_id", mirror.DiscordThreadID, "err", sendErr)
				_ = s.Service.Store.DeleteThreadMirror(ctx, mirror.ID)
			} else {
				s.handleMountRESTError(ctx, mount.ID, probeSeq, sendErr)
			}
			sendMirrorErr = sendErr
			return
		}
		s.Guard.RecordMountSuccess(mount.ID)

		if dbErr := s.Service.Store.CreatePostMirror(ctx, &bbs.PostMirror{
			ID:               uuidv7.New(),
			PostID:           post.ID,
			ThreadMirrorID:   mirror.ID,
			GuildID:          mount.GuildID,
			DiscordMessageID: sent.ID,
			CreatedAt:        time.Now(),
		}); dbErr != nil {
			sendMirrorErr = dbErr
			return
		}

		mirror.SyncedUpToRes = max(mirror.SyncedUpToRes, post.ResNumber)
		_ = s.Service.Store.AdvanceThreadMirrorSync(ctx, mirror.ID, post.ResNumber)

		if freshTh, tErr := s.Service.GetThread(ctx, thread.ID); tErr == nil && freshTh != nil {
			latestThread = *freshTh
		}
		if latestThread.Status == bbs.ThreadStatusArchived {
			if archErr := ArchiveAndLockThread(client, mirror.DiscordThreadID, mount.IsForum); archErr != nil {
				slog.Warn("bbs failed to restore archived thread lock after late reply sync", "discord_thread_id", mirror.DiscordThreadID, "err", archErr)
			}
		}
	}()
	if sendMirrorErr != nil {
		return 0, 0, sendMirrorErr
	}

	// Schedule debounced floating control panel move only if thread remains active.
	if latestThread.Status == bbs.ThreadStatusActive {
		s.scheduleControlPanel(client, mirror, latestThread, mountLocale)
	}

	// Pace forward-anchor updates (without GET) up to MaxForwardAnchorEditsPerPost.
	if len(backwardRefs) > 0 {
		s.enqueueAnchorUpdates(client, mount.GuildID, mirror.DiscordThreadID, latestThread, backwardRefs, mountLocale)
	}

	return mirror.DiscordThreadID, sent.ID, nil
}

func (s *Syncer) scheduleControlPanel(client *bot.Client, mirror *bbs.ThreadMirror, thread bbs.Thread, locale discord.Locale) {
	mirrorCopy := *mirror
	s.Debouncer.Schedule(mirror.ID, func() {
		ctx := s.backgroundCtx()
		if s.Guard.IsThrottled(time.Now()) {
			return
		}
		muLock := s.threadEditMutex(mirrorCopy.DiscordThreadID)
		muLock.Lock()
		defer muLock.Unlock()

		latestMirror, err := s.Service.Store.GetThreadMirror(ctx, mirrorCopy.ID)
		if err != nil {
			return
		}
		if mount, mErr := s.Service.Store.GetBoardMount(ctx, latestMirror.BoardMountID); mErr != nil || !mount.Active {
			return
		}
		latestThread, err := s.Service.GetThread(ctx, thread.ID)
		if err != nil || latestThread.Status != bbs.ThreadStatusActive {
			return
		}
		panelMsg := RenderThreadControlPanel(latestMirror.ID, *latestThread, locale)
		if err := s.Validator.ValidateMessageCreate(panelMsg); err != nil {
			return
		}
		oldControlID := latestMirror.ControlMessageID
		var createdID snowflake.ID
		sendErr := WithUnarchiveRetry(client, latestMirror.DiscordThreadID, func() error {
			sent, cErr := client.Rest.CreateMessage(latestMirror.DiscordThreadID, panelMsg)
			if cErr != nil {
				return cErr
			}
			createdID = sent.ID
			return nil
		})
		if sendErr != nil {
			s.Guard.RecordError(uuid.Nil, sendErr, time.Now())
			return
		}
		if dbErr := s.Service.Store.UpdateThreadMirrorControl(ctx, latestMirror.ID, &createdID); dbErr != nil {
			_ = client.Rest.DeleteMessage(latestMirror.DiscordThreadID, createdID)
			return
		}
		if oldControlID != nil && *oldControlID != createdID {
			if delErr := client.Rest.DeleteMessage(latestMirror.DiscordThreadID, *oldControlID); delErr != nil {
				s.Guard.RecordError(uuid.Nil, delErr, time.Now())
			}
		}
	})
}

func (s *Syncer) enqueueAnchorUpdates(
	client *bot.Client,
	guildID snowflake.ID,
	discordThreadID snowflake.ID,
	thread bbs.Thread,
	refResNums []uint32,
	locale discord.Locale,
) {
	limit := min(len(refResNums), MaxForwardAnchorEditsPerPost)
	targets := slices.Clone(refResNums[:limit])

	s.anchorMu.Lock()
	q, ok := s.anchorQ[discordThreadID]
	if !ok {
		q = &anchorQueue{}
		s.anchorQ[discordThreadID] = q
	}
	if len(q.jobs) >= 32 {
		s.anchorMu.Unlock()
		slog.Warn("bbs anchor update queue full; skipping backpatch", "discord_thread_id", discordThreadID)
		return
	}
	q.jobs = append(q.jobs, anchorItem{
		client:  client,
		guildID: guildID,
		thread:  thread,
		resNums: targets,
		locale:  locale,
	})
	if !q.running {
		q.running = true
		go s.runAnchorWorker(discordThreadID, q)
	}
	s.anchorMu.Unlock()
}

func (s *Syncer) runAnchorWorker(discordThreadID snowflake.ID, q *anchorQueue) {
	for {
		s.anchorMu.Lock()
		if len(q.jobs) == 0 {
			q.running = false
			delete(s.anchorQ, discordThreadID)
			s.anchorMu.Unlock()
			return
		}
		job := q.jobs[0]
		q.jobs[0] = anchorItem{}
		q.jobs = q.jobs[1:]
		s.anchorMu.Unlock()

		s.processAnchorJob(discordThreadID, job)
	}
}

func (s *Syncer) processAnchorJob(discordThreadID snowflake.ID, job anchorItem) {
	ctx := s.backgroundCtx()
	if s.Guard.IsThrottled(time.Now()) {
		return
	}
	prov, err := s.Service.Registry.Get(job.thread.ProviderKey)
	if err != nil {
		return
	}
	resolveURL := s.guildURLResolver(ctx, job.thread.ID, job.guildID, discordThreadID)
	muLock := s.threadEditMutex(discordThreadID)
	for _, resNum := range job.resNums {
		func() {
			muLock.Lock()
			defer muLock.Unlock()
			if s.Guard.IsThrottled(time.Now()) {
				return
			}
			post, err := prov.GetPostByNumber(ctx, job.thread.ID, resNum)
			if err != nil || post.Abone {
				return
			}
			pm, err := s.Service.Store.GetPostMirrorByPostAndGuild(ctx, post.ID, job.guildID)
			if err != nil {
				return
			}
			mu := RenderPostMessageUpdate(*post, job.thread, resolveURL, job.locale)
			if err := s.Validator.ValidateMessageUpdate(mu, true); err != nil {
				return
			}
			wasArchived, editErr := WithUnarchiveRetryState(job.client, discordThreadID, func() error {
				_, uErr := job.client.Rest.UpdateMessage(discordThreadID, pm.DiscordMessageID, mu)
				return uErr
			})
			if editErr != nil {
				s.Guard.RecordError(uuid.Nil, editErr, time.Now())
			}
			latestTh, thErr := prov.GetThread(ctx, job.thread.ID)
			if (thErr == nil && latestTh.Status == bbs.ThreadStatusArchived) || wasArchived {
				isForum := false
				if tm, tmErr := s.Service.Store.GetThreadMirrorByDiscordThread(ctx, discordThreadID); tmErr == nil && tm != nil {
					if m, mErr := s.Service.Store.GetBoardMount(ctx, tm.BoardMountID); mErr == nil && m != nil {
						isForum = m.IsForum
					}
				}
				if thErr == nil && latestTh.Status == bbs.ThreadStatusArchived {
					if archErr := ArchiveAndLockThread(job.client, discordThreadID, isForum); archErr != nil {
						s.Guard.RecordError(uuid.Nil, archErr, time.Now())
					}
				} else if wasArchived {
					if archErr := ArchiveThreadOnly(job.client, discordThreadID, isForum); archErr != nil {
						s.Guard.RecordError(uuid.Nil, archErr, time.Now())
					}
				}
			}
		}()
	}
}

// SyncAbone updates all Discord mirror messages of a soft-deleted post and returns any non-404 errors.
func (s *Syncer) SyncAbone(ctx context.Context, client *bot.Client, post bbs.Post, thread bbs.Thread) error {
	mirrors, err := s.Service.Store.ListPostMirrorsByPost(ctx, post.ID)
	if err != nil {
		slog.Error("bbs failed to list post mirrors for abone", "post_id", post.ID, "err", err)
		return err
	}
	if client == nil || len(mirrors) == 0 {
		return nil
	}
	post.Abone = true
	var errs []error
	for _, pm := range mirrors {
		tm, err := s.Service.Store.GetThreadMirror(ctx, pm.ThreadMirrorID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		var mountLocale discord.Locale
		isForum := false
		if mount, mErr := s.Service.Store.GetBoardMount(ctx, tm.BoardMountID); mErr == nil {
			mountLocale = ResolveMountLocale(client, *mount)
			isForum = mount.IsForum
		} else {
			mountLocale = ResolveGuildLocale(client, pm.GuildID, discord.LocaleJapanese)
		}
		mu := RenderPostMessageUpdate(post, thread, nil, mountLocale)
		if err := s.Validator.ValidateMessageUpdate(mu, true); err != nil {
			errs = append(errs, err)
			continue
		}

		muLock := s.threadEditMutex(tm.DiscordThreadID)
		muLock.Lock()
		wasArchived, editErr := WithUnarchiveRetryState(client, tm.DiscordThreadID, func() error {
			_, uErr := client.Rest.UpdateMessage(tm.DiscordThreadID, pm.DiscordMessageID, mu)
			return uErr
		})
		if editErr != nil {
			s.Guard.RecordError(uuid.Nil, editErr, time.Now())
			slog.Warn("bbs failed to sync abone message", "mirror_id", pm.ID, "err", editErr)
			if !IsJSONError(editErr, JSONErrUnknownChannel) && !IsJSONError(editErr, JSONErrUnknownMessage) && HTTPStatusCode(editErr) != 404 {
				errs = append(errs, editErr)
			}
		}
		if thread.Status == bbs.ThreadStatusArchived {
			if archErr := ArchiveAndLockThread(client, tm.DiscordThreadID, isForum); archErr != nil {
				s.Guard.RecordError(uuid.Nil, archErr, time.Now())
				if !IsJSONError(archErr, JSONErrUnknownChannel) && HTTPStatusCode(archErr) != 404 {
					errs = append(errs, archErr)
				}
			}
		} else if wasArchived {
			if archErr := ArchiveThreadOnly(client, tm.DiscordThreadID, isForum); archErr != nil {
				s.Guard.RecordError(uuid.Nil, archErr, time.Now())
				if !IsJSONError(archErr, JSONErrUnknownChannel) && HTTPStatusCode(archErr) != 404 {
					errs = append(errs, archErr)
				}
			}
		}
		muLock.Unlock()
	}
	return errors.Join(errs...)
}

// SyncArchive posts the 1001/archive warehouse notices and locks+archives every Discord mirror of thread.
func (s *Syncer) SyncArchive(ctx context.Context, client *bot.Client, board bbs.Board, thread bbs.Thread, hitMaxRes bool) {
	mirrors, err := s.Service.Store.ListThreadMirrorsByThread(ctx, thread.ID)
	if err != nil {
		slog.Error("bbs failed to list thread mirrors for archive", "thread_id", thread.ID, "err", err)
		return
	}
	mounts, _ := s.Service.Store.ListActiveBoardMountsByBoard(ctx, board.ID)
	mountMap := make(map[uuid.UUID]bbs.BoardMount, len(mounts))
	for _, m := range mounts {
		mountMap[m.ID] = m
	}

	for _, mirror := range mirrors {
		if !SafeExecuteCtx(ctx) {
			return
		}
		mount, hasMount := mountMap[mirror.BoardMountID]
		mountLocale := discord.LocaleJapanese
		if hasMount {
			mountLocale = ResolveMountLocale(client, mount)
		} else {
			mountLocale = ResolveGuildLocale(client, mirror.GuildID, discord.LocaleJapanese)
		}

		muLock := s.threadEditMutex(mirror.DiscordThreadID)
		muLock.Lock()
		latestMirror := mirror
		if fresh, gErr := s.Service.Store.GetThreadMirror(ctx, mirror.ID); gErr == nil && fresh != nil {
			latestMirror = *fresh
		}
		_ = WithUnarchiveRetry(client, latestMirror.DiscordThreadID, func() error {
			if latestMirror.ControlMessageID != nil {
				delErr := client.Rest.DeleteMessage(latestMirror.DiscordThreadID, *latestMirror.ControlMessageID)
				if delErr == nil || IsJSONError(delErr, JSONErrUnknownMessage) || IsJSONError(delErr, JSONErrUnknownChannel) || HTTPStatusCode(delErr) == 404 {
					_ = s.Service.Store.UpdateThreadMirrorControl(ctx, latestMirror.ID, nil)
				} else {
					s.Guard.RecordError(uuid.Nil, delErr, time.Now())
					slog.Warn("bbs failed to delete control panel during archive sync", "mirror_id", latestMirror.ID, "msg_id", *latestMirror.ControlMessageID, "err", delErr)
				}
			}
			if hitMaxRes {
				overMsg := RenderOver1000Message(thread, mountLocale)
				if err := s.Validator.ValidateMessageCreate(overMsg); err == nil {
					_, _ = client.Rest.CreateMessage(latestMirror.DiscordThreadID, overMsg)
				}
			}
			archMsg := RenderArchivedNotice(mountLocale)
			if err := s.Validator.ValidateMessageCreate(archMsg); err == nil {
				_, _ = client.Rest.CreateMessage(latestMirror.DiscordThreadID, archMsg)
			}
			return nil
		})
		isForum := mount.IsForum
		if !hasMount {
			if mRow, mErr := s.Service.Store.GetBoardMount(ctx, mirror.BoardMountID); mErr == nil && mRow != nil {
				isForum = mRow.IsForum
			}
		}
		if err := ArchiveAndLockThread(client, latestMirror.DiscordThreadID, isForum); err != nil {
			slog.Warn("bbs failed to archive/lock discord thread", "discord_thread_id", latestMirror.DiscordThreadID, "err", err)
		}
		muLock.Unlock()
	}
}

// RunDatOchiCheck archives stale threads across all registered providers and syncs Discord locks.
func (s *Syncer) RunDatOchiCheck(ctx context.Context, client *bot.Client) error {
	now := time.Now()
	if s.Service != nil {
		s.Service.PruneRateLimits(now)
	}
	if s.Guard.IsThrottled(now) {
		return nil
	}
	for _, p := range s.Service.Registry.All() {
		stale, err := p.ListStaleThreads(ctx, now)
		if err != nil {
			slog.Error("bbs failed to list stale threads", "provider", p.Key(), "err", err)
			continue
		}
		for _, th := range stale {
			archived, err := p.ArchiveThread(ctx, th.ID)
			if err != nil {
				slog.Warn("bbs failed to archive stale thread", "thread_id", th.ID, "err", err)
				continue
			}
			board, err := s.Service.GetBoard(ctx, archived.BoardID)
			if err != nil {
				continue
			}
			s.SyncArchive(ctx, client, *board, *archived, archived.ResCount >= archived.Rules.Normalize().MaxRes)
		}
	}
	return nil
}

// HandleMessageUpdate removes MediaGallery previews if Discord ContentScanMetadata flags harmful media.
func (s *Syncer) HandleMessageUpdate(ctx context.Context, client *bot.Client, event *events.GuildMessageUpdate) {
	if event == nil || client == nil {
		return
	}
	chNSFW := false
	if ch, ok := client.Caches.GuildMessageChannel(event.ChannelID); ok {
		chNSFW = ch.NSFW()
	}
	if !HasFlaggedMediaScan(event.Message, chNSFW) {
		return
	}
	pm, err := s.Service.Store.GetPostMirrorByDiscordMessage(ctx, event.MessageID)
	if err != nil {
		return
	}
	post, err := s.Service.GetPost(ctx, pm.PostID)
	if err != nil {
		return
	}
	thread, err := s.Service.GetThread(ctx, post.ThreadID)
	if err != nil {
		return
	}
	aboned, err := s.Service.AbonePostGlobal(ctx, client.ApplicationID, post.ID, "media_scan_flagged")
	if err != nil {
		// Fallback to direct provider abone when ApplicationID is not in Operators slice.
		if p, pErr := s.Service.Registry.Get(thread.ProviderKey); pErr == nil {
			if direct, dErr := p.AbonePost(ctx, post.ID, "media_scan_flagged"); dErr == nil {
				aboned = direct
			}
		}
	}
	if aboned != nil {
		if syncErr := s.SyncAbone(ctx, client, *aboned, *thread); syncErr != nil {
			slog.Warn("bbs failed to sync flagged media abone", "post_id", aboned.ID, "err", syncErr)
		}
	}
}

// HandleChannelDelete deactivates board mounts when a Discord channel is manually deleted (prevents 404 loops).
func (s *Syncer) HandleChannelDelete(ctx context.Context, channelID snowflake.ID) {
	mount, err := s.Service.Store.GetBoardMountByChannel(ctx, channelID)
	if err != nil {
		return
	}
	_ = s.Service.Store.DeactivateBoardMount(ctx, mount.ID)
}

// HandleThreadDelete cleans up thread mirror metadata when a Discord thread is manually deleted.
func (s *Syncer) HandleThreadDelete(ctx context.Context, threadID snowflake.ID) {
	mirror, err := s.Service.Store.GetThreadMirrorByDiscordThread(ctx, threadID)
	if err != nil {
		return
	}
	_ = s.Service.Store.DeleteThreadMirror(ctx, mirror.ID)
}

// RefreshBoardGuide serializes board guide edits against repositionBoardTextGuide under the channel mutex
// and reloads the latest board rules and guide message/post IDs before updating Discord.
func (s *Syncer) RefreshBoardGuide(ctx context.Context, client *bot.Client, guildID snowflake.ID, boardID uuid.UUID, fallbackLocale discord.Locale) {
	if client == nil {
		return
	}
	mount, err := s.Service.Store.GetBoardMountByGuildAndBoard(ctx, guildID, boardID)
	if err != nil || !mount.Active {
		return
	}
	muLock := s.threadEditMutex(mount.ChannelID)
	muLock.Lock()
	defer muLock.Unlock()

	latestMount, err := s.Service.Store.GetBoardMount(ctx, mount.ID)
	if err != nil || !latestMount.Active {
		return
	}
	latestBoard, err := s.Service.GetBoard(ctx, boardID)
	if err != nil {
		return
	}
	mLoc := ResolveMountLocale(client, *latestMount)
	if latestMount.Locale == "" && fallbackLocale != "" && fallbackLocale != discord.LocaleUnknown {
		mLoc = fallbackLocale
	}
	guide := RenderBoardGuide(*latestBoard, latestMount.ID, mLoc)
	mu := discord.MessageUpdate{
		Components:      &guide.Components,
		Flags:           &guide.Flags,
		AllowedMentions: guide.AllowedMentions,
	}
	if err := s.Validator.ValidateMessageUpdate(mu, true); err != nil {
		slog.Warn("bbs refreshLocalMountGuide validation failed", "mount_id", latestMount.ID, "err", err)
		return
	}
	if latestMount.PinnedPostID != nil {
		if err := WithUnarchiveRetry(client, *latestMount.PinnedPostID, func() error {
			_, editErr := client.Rest.UpdateMessage(*latestMount.PinnedPostID, *latestMount.PinnedPostID, mu)
			return editErr
		}); err != nil {
			slog.Warn("bbs failed to refresh forum pinned guide", "mount_id", latestMount.ID, "post_id", *latestMount.PinnedPostID, "err", err)
		}
		return
	}
	if latestMount.ControlMessageID != nil {
		if _, err := client.Rest.UpdateMessage(latestMount.ChannelID, *latestMount.ControlMessageID, mu); err != nil {
			slog.Warn("bbs failed to refresh text control guide", "mount_id", latestMount.ID, "msg_id", *latestMount.ControlMessageID, "err", err)
		}
	}
}

func (s *Syncer) repositionBoardTextGuide(ctx context.Context, client *bot.Client, mount bbs.BoardMount, board bbs.Board, locale discord.Locale) {
	if client == nil || s.Guard.IsThrottled(time.Now()) {
		return
	}
	muLock := s.threadEditMutex(mount.ChannelID)
	muLock.Lock()
	defer muLock.Unlock()

	latestMount, err := s.Service.Store.GetBoardMount(ctx, mount.ID)
	if err != nil || !latestMount.Active {
		return
	}
	latestBoard := board
	if freshBoard, bErr := s.Service.GetBoard(ctx, board.ID); bErr == nil && freshBoard != nil {
		latestBoard = *freshBoard
	}
	guideMsg := RenderBoardGuide(latestBoard, latestMount.ID, locale)
	if err := s.Validator.ValidateMessageCreate(guideMsg); err != nil {
		return
	}
	oldControlID := latestMount.ControlMessageID
	sent, err := client.Rest.CreateMessage(latestMount.ChannelID, guideMsg)
	if err != nil {
		s.Guard.RecordError(uuid.Nil, err, time.Now())
		return
	}
	updated, dbErr := s.Service.Store.UpdateBoardMountControl(ctx, latestMount.ID, &sent.ID)
	if dbErr != nil || !updated {
		_ = client.Rest.DeleteMessage(latestMount.ChannelID, sent.ID)
		return
	}
	if oldControlID != nil && *oldControlID != sent.ID {
		if delErr := client.Rest.DeleteMessage(latestMount.ChannelID, *oldControlID); delErr != nil {
			s.Guard.RecordError(uuid.Nil, delErr, time.Now())
		}
	}
}

func (s *Syncer) notifyWatchers(
	ctx context.Context,
	client *bot.Client,
	board bbs.Board,
	thread bbs.Thread,
	post bbs.Post,
	srcGuildID snowflake.ID,
	srcThreadID snowflake.ID,
	srcMsgID snowflake.ID,
) {
	if client == nil {
		return
	}
	watches, err := s.Service.Store.ListThreadWatches(ctx, thread.ID)
	if err != nil || len(watches) == 0 {
		return
	}
	audit, err := s.Service.Store.GetAuditPostInfo(ctx, post.ID)
	if err != nil {
		return
	}

	var jumpURL string
	if srcGuildID != 0 && srcThreadID != 0 && srcMsgID != 0 {
		jumpURL = discord.MessageURL(srcGuildID, srcThreadID, srcMsgID)
	}

	// Access GORM DB from native store via database helper if available, or check user DM via DB if attached.
	now := time.Now()
	for _, w := range watches {
		if w.UserID == audit.AuthorUserID {
			continue
		}
		if _, cErr := s.Service.CheckConsent(ctx, w.UserID); cErr != nil {
			if errors.Is(cErr, bbs.ErrConsentRequired) {
				_ = s.Service.Store.DeleteThreadWatch(ctx, thread.ID, w.UserID)
			}
			continue
		}
		if s.CanSendDM != nil && !s.CanSendDM(ctx, w.UserID) {
			continue
		}
		if !s.DMCache.AllowNotify(thread.ID, w.UserID, now) {
			continue
		}
		dmChID, err := s.DMCache.GetOrCreateDMChannel(client, w.UserID)
		if err != nil {
			if IsJSONError(err, JSONErrCannotDMUser) {
				_ = s.Service.Store.DeleteThreadWatch(ctx, thread.ID, w.UserID)
			}
			continue
		}
		watchLocale := discord.LocaleJapanese
		if w.Locale != "" {
			watchLocale = discord.Locale(w.Locale)
		} else if s.UserLocale != nil {
			watchLocale = s.UserLocale(ctx, w.UserID)
		}
		dmTitle := i18n.TranslateText(watchLocale, "components.bbs.watch.dm.title")
		bodyText := i18n.TranslateText(watchLocale, "components.bbs.watch.dm.body", map[string]any{
			"board":  bbs.ClearFormat(board.Name),
			"thread": bbs.ClearFormat(thread.Title),
			"res":    post.ResNumber,
		})
		if jumpURL != "" {
			bodyText += "\n" + jumpURL
		}
		unwatchBtn := TruncateRunes(i18n.TranslateText(watchLocale, "components.bbs.watch.dm.unwatch_btn"), MaxButtonLabelRunes)
		dmMsg := discord.NewMessageCreateV2(
			discord.NewContainer(
				discord.NewSection(
					discord.NewTextDisplay(dmTitle),
					discord.NewTextDisplay(TruncateRunes(bodyText, 1000)),
				).WithAccessory(discord.NewDangerButton(unwatchBtn, fmt.Sprintf("/bbs/watch/toggle/%s", thread.ID))),
			).WithAccentColor(ColorNotice),
		).WithAllowedMentions(&discord.AllowedMentions{})
		if vErr := s.Validator.ValidateMessageCreate(dmMsg); vErr != nil {
			continue
		}
		if _, err := client.Rest.CreateMessage(dmChID, dmMsg); err != nil {
			if IsJSONError(err, JSONErrCannotDMUser) {
				s.DMCache.Invalidate(w.UserID)
				_ = s.Service.Store.DeleteThreadWatch(ctx, thread.ID, w.UserID)
			}
		}
	}
}

// CanSendDMToUser checks the user's DMEnabled flag in gobot's core user table.
func CanSendDMToUser(ctx context.Context, gdb *database.DB, userID snowflake.ID) bool {
	if gdb == nil || gdb.DB == nil {
		return true
	}
	u, err := database.GetOrCreateUser(gdb.DB.WithContext(ctx), userID)
	if err != nil {
		slog.Warn("bbs failed to load user DM preference; suppressing DM notification", "user_id", userID, "err", err)
		return false
	}
	return u.DMEnabled
}

func (s *Syncer) guildURLResolver(ctx context.Context, threadID uuid.UUID, guildID snowflake.ID, discordThreadID snowflake.ID) func(resNum uint32) (string, bool) {
	var mu sync.Mutex
	cache := make(map[uint32]string)
	missed := make(map[uint32]bool)
	return func(resNum uint32) (string, bool) {
		mu.Lock()
		if u, ok := cache[resNum]; ok {
			mu.Unlock()
			return u, true
		}
		if missed[resNum] {
			mu.Unlock()
			return "", false
		}
		mu.Unlock()

		pm, err := s.Service.Store.GetPostMirrorByResAndGuild(ctx, threadID, resNum, guildID)
		if err != nil || pm.DiscordMessageID == 0 {
			mu.Lock()
			missed[resNum] = true
			mu.Unlock()
			return "", false
		}
		u := discord.MessageURL(guildID, discordThreadID, pm.DiscordMessageID)
		mu.Lock()
		cache[resNum] = u
		mu.Unlock()
		return u, true
	}
}

func (s *Syncer) handleMountRESTError(ctx context.Context, mountID uuid.UUID, probeSeq uint64, err error) {
	if s.Guard.RecordMountError(mountID, probeSeq, err, time.Now()) {
		slog.Warn("bbs deactivating mount after 404", "mount_id", mountID, "err", err)
		_ = s.Service.Store.DeactivateBoardMount(ctx, mountID)
	}
}

func (s *Syncer) fanout(mounts []bbs.BoardMount, fn func(bbs.BoardMount)) {
	sem := make(chan struct{}, MaxFanoutConcurrency)
	var wg sync.WaitGroup
	for _, m := range mounts {
		wg.Add(1)
		go func(mount bbs.BoardMount) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fn(mount)
		}(m)
	}
	wg.Wait()
}

func (s *Syncer) backgroundCtx() context.Context {
	return context.Background()
}
