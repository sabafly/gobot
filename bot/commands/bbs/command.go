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

package bbs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	pendingTTL         = 10 * time.Minute
	maxPendingEntries  = 2048
	msgCmdReportName   = "BBS - Report Post"
	msgCmdModerateName = "BBS - Moderate Post"
	datCheckInterval   = 15 * time.Minute
)

type pendingKind string

const (
	pendingNewThread    pendingKind = "new_thread"
	pendingNewPost      pendingKind = "new_post"
	pendingWatchThread  pendingKind = "watch_thread"
	pendingReportThread pendingKind = "report_thread"
	pendingReportPost   pendingKind = "report_post"
)

type pendingAction struct {
	Kind      pendingKind
	UserID    snowflake.ID
	TargetID  uuid.UUID
	ExpiresAt time.Time
}

// System coordinates Discord BBS command/component handlers, services, and syncer.
type System struct {
	Components *components.Components
	Service    *bbscore.Service
	Syncer     *discordsync.Syncer
	LogChannel *snowflake.ID

	pendingMu sync.Mutex
	pending   map[string]pendingAction
}

// NewSystem constructs a System orchestrating BBS commands and interactions.
func NewSystem(c *components.Components, svc *bbscore.Service, syncer *discordsync.Syncer, logCh *snowflake.ID) *System {
	return &System{
		Components: c,
		Service:    svc,
		Syncer:     syncer,
		LogChannel: logCh,
		pending:    make(map[string]pendingAction),
	}
}

func (s *System) savePending(userID snowflake.ID, kind pendingKind, targetID uuid.UUID) string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	token := hex.EncodeToString(buf[:])

	now := time.Now()
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for k, v := range s.pending {
		if now.After(v.ExpiresAt) || len(s.pending) >= maxPendingEntries {
			delete(s.pending, k)
		}
	}
	s.pending[token] = pendingAction{
		Kind:      kind,
		UserID:    userID,
		TargetID:  targetID,
		ExpiresAt: now.Add(pendingTTL),
	}
	return token
}

func (s *System) popPending(token string, userID snowflake.ID) (pendingAction, bool) {
	if token == "" || token == "none" {
		return pendingAction{}, false
	}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	act, ok := s.pending[token]
	if !ok {
		return pendingAction{}, false
	}
	if time.Now().After(act.ExpiresAt) || act.UserID != userID {
		if time.Now().After(act.ExpiresAt) {
			delete(s.pending, token)
		}
		return pendingAction{}, false
	}
	delete(s.pending, token)
	return act, true
}

type permEventAdapter struct {
	member  *discord.ResolvedMember
	guildID *snowflake.ID
	user    discord.User
	client  *bot.Client
}

func (p permEventAdapter) Member() *discord.ResolvedMember { return p.member }
func (p permEventAdapter) GuildID() *snowflake.ID          { return p.guildID }
func (p permEventAdapter) User() discord.User              { return p.user }
func (p permEventAdapter) Client() *bot.Client             { return p.client }

func (s *System) hasGuildAdmin(ctx context.Context, member *discord.ResolvedMember, guildID *snowflake.ID, user discord.User, client *bot.Client) bool {
	if guildID == nil {
		return false
	}
	return generic.CheckEventPermission(
		ctx,
		permEventAdapter{member: member, guildID: guildID, user: user, client: client},
		s.Components,
		[]generic.Permission{generic.PermissionString("bbs.admin")},
		discord.PermissionManageGuild,
	)
}

func (s *System) requireCmdGuildAdmin(e *handler.CommandEvent) bool {
	if s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		return true
	}
	loc := e.Locale()
	s.replyCommandError(e,
		i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
		i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
	)
	return false
}

func (s *System) replyCommandError(e *handler.CommandEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(e.Locale(), title, body))
}

func (s *System) replyCommandNotice(e *handler.CommandEvent, title, body string, color int) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(e.Locale(), title, body, color).WithEphemeral(true))
}

func (s *System) replyComponentError(e *handler.ComponentEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(e.Locale(), title, body))
}

func (s *System) replyModalError(e *handler.ModalEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(e.Locale(), title, body))
}

func guildLocale(e interface {
	GuildID() *snowflake.ID
	GuildLocale() *discord.Locale
	Locale() discord.Locale
	Client() *bot.Client
}) discord.Locale {
	fb := e.Locale()
	if gl := e.GuildLocale(); gl != nil && *gl != "" && *gl != discord.LocaleUnknown {
		fb = *gl
	}
	var gid snowflake.ID
	if e.GuildID() != nil {
		gid = *e.GuildID()
	}
	return discordsync.ResolveGuildLocale(e.Client(), gid, fb)
}

// Command returns the public /bbs command, message context menus, routes, event hooks, and dat-ochi scheduler.
func Command(sys *System) components.Command {
	return &bbsCommand{sys: sys}
}

type bbsCommand struct {
	sys *System
}

func (c *bbsCommand) Name() string { return "bbs" }

func (c *bbsCommand) IsPrivate() bool { return false }

func intPtr(v int) *int { return &v }

func (c *bbsCommand) Create() []discord.ApplicationCommandCreate {
	guildCtx := []discord.InteractionContextType{discord.InteractionContextTypeGuild}
	slipChoices := []discord.ApplicationCommandOptionChoiceString{
		{
			Name:              "none (Hidden)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs.choices.slip.none"),
			Value:             "none",
		},
		{
			Name:              "vvvvv (Device + OS)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs.choices.slip.vvvvv"),
			Value:             "vvvvv",
		},
	}

	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:                     "bbs",
			Description:              "Configure the anonymous BBS, manage boards, and review policies",
			DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.description"),
			Contexts:                 guildCtx,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:                     "setup",
					Description:              "Enable or disable BBS features in this server",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.description"),
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{
							Name:                     "enabled",
							Description:              "Enable BBS features in this server",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.options.enabled.description"),
							Required:                 true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "category",
					Description:              "Manage server-local categories",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a server-local category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Category name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(60),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Category description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.options.description.description"),
									MaxLength:                intPtr(200),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List server-local categories",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.list.description"),
						},
						{
							Name:                     "delete",
							Description:              "Delete a server-local category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Local category to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.delete.options.category.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "board",
					Description:              "Manage server-local boards",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a local board and Discord channel",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Board name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(80),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Local category for the board",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.category.description"),
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "Default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "forum",
									Description:              "Create as a forum channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.forum.description"),
								},
								discord.ApplicationCommandOptionChannel{
									Name:                     "discord_category",
									Description:              "Parent Discord category channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.discord_category.description"),
									ChannelTypes:             []discord.ChannelType{discord.ChannelTypeGuildCategory},
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP (Wachoi) display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.slip.description"),
									Choices:                  slipChoices,
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "max_res",
									Description:              "Max replies per thread (50-1000)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.max_res.description"),
									MinValue:                 intPtr(50),
									MaxValue:                 intPtr(1000),
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "archive_days",
									Description:              "Days of inactivity before auto-archive (0 disables)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.archive_days.description"),
									MinValue:                 intPtr(0),
									MaxValue:                 intPtr(365),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List boards mounted in this server",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.list.description"),
						},
						{
							Name:                     "edit",
							Description:              "Edit a server-local board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Local board to edit",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "New default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "New board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP (Wachoi) display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.slip.description"),
									Choices:                  slipChoices,
								},
							},
						},
						{
							Name:                     "delete",
							Description:              "Delete a local board and remove its mount",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Local board to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.delete.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "global",
					Description:              "Browse and subscribe to global BBS boards",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "list",
							Description:              "List available global BBS boards",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.list.description"),
						},
						{
							Name:                     "subscribe",
							Description:              "Subscribe this server to a global BBS board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to subscribe",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "forum",
									Description:              "Create as a forum channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.forum.description"),
								},
								discord.ApplicationCommandOptionChannel{
									Name:                     "discord_category",
									Description:              "Parent Discord category channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.discord_category.description"),
									ChannelTypes:             []discord.ChannelType{discord.ChannelTypeGuildCategory},
								},
							},
						},
						{
							Name:                     "unsubscribe",
							Description:              "Unsubscribe this server from a global board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.unsubscribe.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to unsubscribe",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.unsubscribe.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:                     "policy",
					Description:              "View BBS Terms, Guidelines, Privacy Policy, and consent status",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.policy.description"),
				},
			},
		},
		discord.MessageCommandCreate{
			Name:              msgCmdReportName,
			NameLocalizations: i18n.TranslateTextMap("command.bbs.msg.report.name"),
			Contexts:          guildCtx,
		},
		discord.MessageCommandCreate{
			Name:                     msgCmdModerateName,
			NameLocalizations:        i18n.TranslateTextMap("command.bbs.msg.moderate.name"),
			DefaultMemberPermissions: omit.NewPtr(discord.PermissionManageGuild),
			Contexts:                 guildCtx,
		},
	}
}

func (c *bbsCommand) RegisterRoutes(r handler.Router) {
	r.SlashCommand("/bbs/setup", c.handleSetup)
	r.SlashCommand("/bbs/category/create", c.handleCategoryCreate)
	r.SlashCommand("/bbs/category/list", c.handleCategoryList)
	r.SlashCommand("/bbs/category/delete", c.handleCategoryDelete)
	r.SlashCommand("/bbs/board/create", c.handleBoardCreate)
	r.SlashCommand("/bbs/board/list", c.handleBoardList)
	r.SlashCommand("/bbs/board/edit", c.handleBoardEdit)
	r.SlashCommand("/bbs/board/delete", c.handleBoardDelete)
	r.SlashCommand("/bbs/global/list", c.handleGlobalList)
	r.SlashCommand("/bbs/global/subscribe", c.handleGlobalSubscribe)
	r.SlashCommand("/bbs/global/unsubscribe", c.handleGlobalUnsubscribe)
	r.SlashCommand("/bbs/policy", c.handlePolicy)

	r.Autocomplete("/bbs/category/delete", c.autocompleteLocalCategories)
	r.Autocomplete("/bbs/board/create", c.autocompleteLocalCategories)
	r.Autocomplete("/bbs/board/edit", c.autocompleteLocalBoards)
	r.Autocomplete("/bbs/board/delete", c.autocompleteLocalBoards)
	r.Autocomplete("/bbs/global/subscribe", c.autocompleteGlobalBoards)
	r.Autocomplete("/bbs/global/unsubscribe", c.autocompleteSubscribedGlobalBoards)

	r.MessageCommand("/"+msgCmdReportName, c.sys.handleReportMessageCommand)
	r.MessageCommand("/"+msgCmdModerateName, c.sys.handleModerateMessageCommand)

	r.Component("/bbs/consent/revoke", c.sys.handleConsentRevoke)
	r.Component("/bbs/consent/view/{doc}", c.sys.handleConsentViewDoc)
	r.Component("/bbs/consent/open/{version}/{token}", c.sys.handleConsentOpenModal)
	r.Modal("/bbs/consent/modal/{version}/{token}", c.sys.handleConsentModal)
	r.Component("/bbs/consent/accept/{version}/{token}", c.sys.handleConsentOpenModal)
	r.Component("/bbs/consent/accept/{token}", c.sys.handleConsentOpenModal)
	r.Component("/bbs/consent/role/{guild_id}/{role_id}/{version}", c.sys.handleConsentRoleDeprecated)
	r.Component("/bbs/consent/role/{guild_id}/{version}", c.sys.handleConsentRoleDeprecated)
	r.Component("/bbs/consent/role/{guild_id}", c.sys.handleConsentRoleDeprecated)

	r.Component("/bbs/board/threads/{mount_id}", c.sys.handleBoardThreads)
	r.Component("/bbs/board/jump/{mount_id}", c.sys.handleBoardJumpSelect)
	r.Component("/bbs/thread/new/{mount_id}", c.sys.handleNewThreadButton)
	r.Modal("/bbs/thread/modal/{mount_id}", c.sys.handleNewThreadModal)
	r.Component("/bbs/post/new/{mirror_id}", c.sys.handleNewPostButton)
	r.Modal("/bbs/post/modal/{mirror_id}", c.sys.handleNewPostModal)
	r.Component("/bbs/watch/toggle/{thread_id}", c.sys.handleWatchToggle)

	// CRITICAL ORDER: static /jump BEFORE parameterized /{thread_id}/{page}
	r.Component("/bbs/reader/jump/{thread_id}", c.sys.handleReaderJumpButton)
	r.Modal("/bbs/reader/jumpmodal/{thread_id}", c.sys.handleReaderJumpModal)
	r.Component("/bbs/reader/{thread_id}/{page}", c.sys.handleReaderPage)

	r.Component("/bbs/report/thread/{thread_id}", c.sys.handleReportThreadButton)
	r.Modal("/bbs/report/threadmodal/{thread_id}", c.sys.handleReportThreadModal)
	r.Component("/bbs/report/post/{post_id}", c.sys.handleReportPostButton)
	r.Modal("/bbs/report/postmodal/{post_id}", c.sys.handleReportPostModal)
	r.Component("/bbs/mod/abone/{post_id}", c.sys.handleModAboneButton)
	r.Modal("/bbs/mod/abonemodal/{post_id}", c.sys.handleModAboneModal)
	r.Component("/bbs/mod/archive/{thread_id}", c.sys.handleModArchiveButton)
	r.Component("/bbs/mod/ban/{post_id}", c.sys.handleModBanButton)
	r.Modal("/bbs/mod/banmodal/{post_id}", c.sys.handleModBanModal)
}

func (c *bbsCommand) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return nil
}
func (c *bbsCommand) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return nil
}
func (c *bbsCommand) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return nil
}
func (c *bbsCommand) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
	return nil
}
func (c *bbsCommand) HasCommand(_ *events.ApplicationCommandInteractionCreate) bool { return false }

func (c *bbsCommand) OnEvent() func(event bot.Event) error {
	return func(event bot.Event) error {
		ctx := c.sys.Components.Ctx()
		switch e := event.(type) {
		case *events.GuildMessageUpdate:
			c.sys.Syncer.HandleMessageUpdate(ctx, e.Client(), e)
		case *events.GuildChannelDelete:
			c.sys.Syncer.HandleChannelDelete(ctx, e.ChannelID)
		case *events.ThreadDelete:
			c.sys.Syncer.HandleThreadDelete(ctx, e.ThreadID)
		}
		return nil
	}
}

func (c *bbsCommand) Scheduler() []components.Scheduler {
	return []components.Scheduler{
		{
			Duration: datCheckInterval,
			Worker: func(comp *components.Components, client *bot.Client) error {
				return c.sys.Syncer.RunDatOchiCheck(comp.Ctx(), client)
			},
		},
	}
}

func (c *bbsCommand) handleSetup(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()

	enabled := data.Bool("enabled")
	input := bbscore.GuildSetupInput{
		GuildID:     *guildID,
		Enabled:     enabled,
		ActorUserID: e.User().ID,
	}

	cfg, err := c.sys.Service.SetupGuild(e.Ctx, input)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.cmd.setup.failed_title"),
			describeBBSError(loc, err),
		)
		return nil
	}

	stateStr := i18n.TranslateText(loc, "components.bbs.common.disabled")
	if cfg.Enabled {
		stateStr = i18n.TranslateText(loc, "components.bbs.common.enabled")
	}
	summary := i18n.TranslateText(loc, "components.bbs.cmd.setup.summary", map[string]any{
		"enabled": stateStr,
		"version": cfg.AgreedVersion,
	})

	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.setup.success_title"),
		summary,
		discordsync.ColorLocalBoard,
	)
	return nil
}

func (c *bbsCommand) handleCategoryCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	cat, err := c.sys.Service.CreateCategory(e.Ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeLocal,
		GuildID:     guildID,
		Name:        data.String("name"),
		Description: data.String("description"),
	})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.create_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.category.created_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.category.created_body", map[string]any{
			"name": cat.Name,
			"id":   cat.ID.String(),
		}),
		discordsync.ColorLocalBoard,
	)
	return nil
}

func (c *bbsCommand) handleCategoryList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: guildID})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.cmd.category.list_title")
	if len(cats) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.cmd.category.list_empty"), discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, cat := range cats {
		line := "- " + cat.Name + " (" + cat.ID.String() + ")"
		if cat.Description != "" {
			line += " -- " + discordsync.TruncateRunes(cat.Description, 60)
		}
		lines = append(lines, line)
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleCategoryDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.category.invalid_select"),
		)
		return nil
	}
	cat, err := c.sys.Service.GetCategory(e.Ctx, catID)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if cat.Scope != bbscore.ScopeLocal || cat.GuildID == nil || *cat.GuildID != *guildID {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.cmd.category.local_only_delete"),
		)
		return nil
	}
	if err := c.sys.Service.DeleteCategory(e.Ctx, catID); err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.category.deleted_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.category.deleted_body", map[string]any{"name": cat.Name}),
		discordsync.ColorNotice,
	)
	return nil
}

func (c *bbsCommand) handleBoardCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	gLoc := guildLocale(e)

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs board create defer failed", "err", err)
	}

	if _, err := c.sys.Service.EnsureGuildEnabled(e.Ctx, *guildID); err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.create_failed_title"),
			describeBBSError(loc, err),
		))
		return nil
	}

	boardName := bbscore.SanitizeSingleLine(data.String("name"))
	if boardName == "" || utf8.RuneCountInString(boardName) > 80 {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.create_failed_title"),
			describeBBSError(loc, bbscore.ErrInvalidBoardName),
		))
		return nil
	}
	rules, err := parseBoardRulesFromSlash(data, gLoc)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.create_failed_title"),
			describeBBSError(loc, err),
		))
		return nil
	}

	catID, err := c.resolveOrCreateDefaultLocalCategory(e.Ctx, *guildID, data.String("category"), gLoc)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.cat_resolve_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}
	wantForum := true
	if f, ok := data.OptBool("forum"); ok {
		wantForum = f
	}
	var discordCatID *snowflake.ID
	if ch, ok := data.OptChannel("discord_category"); ok {
		id := ch.ID
		discordCatID = &id
	}

	board, err := c.sys.Service.CreateBoard(e.Ctx, bbscore.CreateBoardInput{
		CategoryID:  catID,
		Scope:       bbscore.ScopeLocal,
		GuildID:     guildID,
		Name:        data.String("name"),
		Description: data.String("description"),
		Rules:       rules,
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.create_failed_title"),
			describeBBSError(loc, err),
		))
		return nil
	}

	mount, _, err := c.sys.Syncer.ProvisionBoardChannel(e.Ctx, e.Client(), *guildID, *board, discordCatID, wantForum, gLoc)
	if err != nil {
		_ = c.sys.Service.DeleteBoard(e.Ctx, board.ID)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.board.channel_failed_title"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.channel_failed_body", map[string]any{
				"err": describeBBSError(loc, err),
			}),
		))
		return nil
	}

	msg := i18n.TranslateText(loc, "components.bbs.cmd.board.created_body", map[string]any{
		"name":       board.Name,
		"channel_id": mount.ChannelID,
		"max_res":    board.Rules.MaxRes,
	})
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.cmd.board.created_title"),
		msg,
		discordsync.ColorLocalBoard,
	))
	return nil
}

func (c *bbsCommand) resolveOrCreateDefaultLocalCategory(ctx context.Context, guildID snowflake.ID, rawCat string, locale discord.Locale) (uuid.UUID, error) {
	if strings.TrimSpace(rawCat) != "" {
		id, err := uuid.Parse(strings.TrimSpace(rawCat))
		if err != nil {
			return uuid.Nil, bbscore.ErrInvalidCategoryName
		}
		cat, err := c.sys.Service.GetCategory(ctx, id)
		if err != nil {
			return uuid.Nil, err
		}
		if cat.Scope != bbscore.ScopeLocal || cat.GuildID == nil || *cat.GuildID != guildID {
			return uuid.Nil, bbscore.ErrUnauthorized
		}
		return id, nil
	}
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: &guildID})
	if err != nil {
		return uuid.Nil, err
	}
	if len(cats) > 0 {
		return cats[0].ID, nil
	}
	created, err := c.sys.Service.CreateCategory(ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeLocal,
		GuildID:     &guildID,
		Name:        i18n.TranslateText(locale, "components.bbs.cmd.default_category_name"),
		Description: i18n.TranslateText(locale, "components.bbs.cmd.default_category_desc"),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

func (c *bbsCommand) handleBoardList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	mounts, err := c.sys.Service.Store.ListBoardMountsByGuild(e.Ctx, *guildID)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.cmd.board.list_title")
	if len(mounts) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.cmd.board.list_empty"), discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, m := range mounts {
		b, bErr := c.sys.Service.GetBoard(e.Ctx, m.BoardID)
		if bErr != nil {
			continue
		}
		scopeKey := "components.bbs.common.scope.local"
		if b.Scope == bbscore.ScopeGlobal {
			scopeKey = "components.bbs.common.scope.global"
		}
		modeKey := "components.bbs.common.mode.text"
		if m.IsForum {
			modeKey = "components.bbs.common.mode.forum"
		}
		lines = append(lines, i18n.TranslateText(loc, "components.bbs.cmd.board.list_entry", map[string]any{
			"scope":      i18n.TranslateText(loc, scopeKey),
			"name":       b.Name,
			"channel_id": m.ChannelID,
			"mode":       i18n.TranslateText(loc, modeKey),
			"max_res":    b.Rules.Normalize().MaxRes,
		}))
	}
	if len(lines) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.cmd.board.list_none_visible"), discordsync.ColorNotice)
		return nil
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleBoardEdit(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.invalid_id"),
		)
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.edit_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.local_only_edit"),
		)
		return nil
	}
	input, err := parseUpdateBoardInput(data)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.edit_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	updated, err := c.sys.Service.UpdateBoard(e.Ctx, boardID, input)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.edit_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.refreshLocalMountGuide(e.Ctx, e.Client(), *guildID, *updated, guildLocale(e))

	hideKey := "components.bbs.common.disabled"
	if updated.Rules.HideID {
		hideKey = "components.bbs.common.enabled"
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.board.updated_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.board.updated_body", map[string]any{
			"name":         updated.Name,
			"default_name": updated.Rules.Normalize().DefaultName,
			"hide_id":      i18n.TranslateText(loc, hideKey),
		}),
		discordsync.ColorLocalBoard,
	)
	return nil
}

func (c *bbsCommand) refreshLocalMountGuide(ctx context.Context, client *bot.Client, guildID snowflake.ID, board bbscore.Board, fallbackLocale discord.Locale) {
	c.sys.Syncer.RefreshBoardGuide(ctx, client, guildID, board.ID, fallbackLocale)
}

func (c *bbsCommand) handleBoardDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.invalid_id"),
		)
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.local_only_delete"),
		)
		return nil
	}
	if err := c.sys.Service.DeleteBoard(e.Ctx, boardID); err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.board.deleted_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.board.deleted_body", map[string]any{"name": board.Name}),
		discordsync.ColorNotice,
	)
	return nil
}

func (c *bbsCommand) handleGlobalList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	loc := e.Locale()
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.cmd.global.list_title")
	if len(boards) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.cmd.global.list_empty"), discordsync.ColorGlobalBoard)
		return nil
	}
	var lines []string
	for _, b := range boards {
		line := "- " + b.Name + " (ID: " + b.ID.String() + ")"
		if b.Description != "" {
			line += " -- " + discordsync.TruncateRunes(b.Description, 60)
		}
		lines = append(lines, line)
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *bbsCommand) handleGlobalSubscribe(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	gLoc := guildLocale(e)

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs global subscribe defer failed", "err", err)
	}

	subFailTitle := i18n.TranslateText(loc, "components.bbs.cmd.global.subscribe_failed")
	if _, err := c.sys.Service.EnsureGuildEnabled(e.Ctx, *guildID); err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, subFailTitle, describeBBSError(loc, err)))
		return nil
	}

	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.invalid_id"),
		))
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, subFailTitle, describeBBSError(loc, err)))
		return nil
	}
	if board.Scope != bbscore.ScopeGlobal {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.global.not_global"),
		))
		return nil
	}

	wantForum := true
	if f, ok := data.OptBool("forum"); ok {
		wantForum = f
	}
	var discordCatID *snowflake.ID
	if ch, ok := data.OptChannel("discord_category"); ok {
		id := ch.ID
		discordCatID = &id
	}

	mount, _, err := c.sys.Syncer.ProvisionBoardChannel(e.Ctx, e.Client(), *guildID, *board, discordCatID, wantForum, gLoc)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, subFailTitle, describeBBSError(loc, err)))
		return nil
	}

	msg := i18n.TranslateText(loc, "components.bbs.cmd.global.subscribed_body", map[string]any{
		"name":       board.Name,
		"channel_id": mount.ChannelID,
	})
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.cmd.global.subscribed_title"),
		msg,
		discordsync.ColorGlobalBoard,
	))
	return nil
}

func (c *bbsCommand) handleGlobalUnsubscribe(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.cmd.board.invalid_id"),
		)
		return nil
	}
	unsubFailTitle := i18n.TranslateText(loc, "components.bbs.cmd.global.unsubscribe_failed")
	mount, err := c.sys.Service.Store.GetBoardMountByGuildAndBoard(e.Ctx, *guildID, boardID)
	if err != nil {
		c.sys.replyCommandError(e, unsubFailTitle, describeBBSError(loc, err))
		return nil
	}
	if err := c.sys.Service.Store.DeactivateBoardMount(e.Ctx, mount.ID); err != nil {
		c.sys.replyCommandError(e, unsubFailTitle, describeBBSError(loc, err))
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.cmd.global.unsubscribed_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.global.unsubscribed_body", map[string]any{
			"channel_id": mount.ChannelID,
		}),
		discordsync.ColorNotice,
	)
	return nil
}

func (c *bbsCommand) handlePolicy(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	loc := e.Locale()
	consent, err := c.sys.Service.CheckConsent(e.Ctx, e.User().ID)
	pm := c.sys.Service.Policy

	if err != nil {
		msg := discordsync.RenderConsentGate(pm, loc, "none")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, msg)
		return nil
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.cmd.policy.status_title")),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.cmd.policy.status_body", map[string]any{
				"agreed_version":  consent.PolicyVersion,
				"current_version": pm.CurrentVersion,
				"agreed_at":       consent.AgreedAt.In(bbscore.JSTLocation).Format("2006/01/02 15:04:05"),
			})),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.consent.btn.terms"), discordsync.MaxButtonLabelRunes), "/bbs/consent/view/terms"),
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.consent.btn.guidelines"), discordsync.MaxButtonLabelRunes), "/bbs/consent/view/guidelines"),
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.consent.btn.privacy"), discordsync.MaxButtonLabelRunes), "/bbs/consent/view/privacy"),
			),
			discord.NewActionRow(
				discord.NewDangerButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.cmd.policy.revoke_btn"), discordsync.MaxButtonLabelRunes), "/bbs/consent/revoke"),
			),
		).WithAccentColor(discordsync.ColorNotice),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (c *bbsCommand) autocompleteLocalCategories(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: guildID})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, cat := range cats {
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		if q == "" || strings.Contains(strings.ToLower(cat.Name), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(cat.Name, 90),
				Value: cat.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs local category autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
}

func (c *bbsCommand) autocompleteLocalBoards(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeLocal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope, GuildID: guildID})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) autocompleteGlobalBoards(e *handler.AutocompleteEvent) error {
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) autocompleteSubscribedGlobalBoards(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	mounts, err := c.sys.Service.Store.ListBoardMountsByGuild(e.Ctx, *guildID)
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	var boards []bbscore.Board
	for _, m := range mounts {
		b, bErr := c.sys.Service.GetBoard(e.Ctx, m.BoardID)
		if bErr == nil && b.Scope == bbscore.ScopeGlobal {
			boards = append(boards, *b)
		}
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) respondBoardChoices(e *handler.AutocompleteEvent, boards []bbscore.Board) error {
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, b := range boards {
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) || strings.Contains(strings.ToLower(b.Slug), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(b.Name, 90),
				Value: b.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs board autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
}

func parseBoardRulesFromSlash(data discord.SlashCommandInteractionData, defaultLocale discord.Locale) (bbscore.BoardRules, error) {
	rules := bbscore.BoardRules{}
	if v, ok := data.OptString("default_name"); ok {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > bbscore.MaxNameRunes {
			return bbscore.BoardRules{}, bbscore.ErrInvalidBoardRules
		}
		rules.DefaultName = trimmed
	} else {
		rules.DefaultName = i18n.TranslateText(defaultLocale, "components.bbs.post.default_name")
	}
	if v, ok := data.OptBool("hide_id"); ok {
		rules.HideID = v
	}
	if v, ok := data.OptString("slip"); ok {
		switch v {
		case "none":
			rules.SLIP = bbscore.SLIPNone
		case "vvvvv":
			rules.SLIP = bbscore.SLIPVVVVV
		default:
			return bbscore.BoardRules{}, bbscore.ErrInvalidBoardRules
		}
	}
	if v, ok := data.OptInt("max_res"); ok {
		if v < int(bbscore.MinSlashBoardMaxRes) || v > int(bbscore.MaxBoardMaxRes) {
			return bbscore.BoardRules{}, bbscore.ErrInvalidBoardRules
		}
		rules.MaxRes = uint32(v)
	}
	if v, ok := data.OptInt("archive_days"); ok {
		if v < 0 || v > bbscore.MaxArchiveDays {
			return bbscore.BoardRules{}, bbscore.ErrInvalidBoardRules
		}
		if v == 0 {
			rules.DatOchiIdleHours = bbscore.DatOchiDisabled
			rules.DatOchiLowResHours = bbscore.DatOchiDisabled
		} else {
			rules.DatOchiIdleHours = v * 24
			rules.DatOchiLowResHours = v * 24
		}
	}
	rules = rules.Normalize()
	if err := rules.Validate(); err != nil {
		return bbscore.BoardRules{}, err
	}
	return rules, nil
}

func parseUpdateBoardInput(data discord.SlashCommandInteractionData) (bbscore.UpdateBoardInput, error) {
	var input bbscore.UpdateBoardInput
	if v, ok := data.OptString("default_name"); ok {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > bbscore.MaxNameRunes {
			return bbscore.UpdateBoardInput{}, bbscore.ErrInvalidBoardRules
		}
		input.DefaultName = &trimmed
	}
	if v, ok := data.OptString("description"); ok {
		input.Description = &v
	}
	if v, ok := data.OptBool("hide_id"); ok {
		input.HideID = &v
	}
	if v, ok := data.OptString("slip"); ok {
		switch v {
		case "none":
			lvl := bbscore.SLIPNone
			input.SLIP = &lvl
		case "vvvvv":
			lvl := bbscore.SLIPVVVVV
			input.SLIP = &lvl
		default:
			return bbscore.UpdateBoardInput{}, bbscore.ErrInvalidBoardRules
		}
	}
	if v, ok := data.OptBool("locked"); ok {
		input.Locked = &v
	}
	return input, nil
}

func describeBBSError(locale discord.Locale, err error) string {
	if msg, ok := discordsync.FormatPreflightError(locale, err); ok {
		return msg
	}
	switch {
	case errors.Is(err, bbscore.ErrGuildNotEnabled):
		return i18n.TranslateText(locale, "components.bbs.errors.guild_not_enabled")
	case errors.Is(err, bbscore.ErrConsentRequired):
		return i18n.TranslateText(locale, "components.bbs.errors.consent_required")
	case errors.Is(err, bbscore.ErrStalePolicyVersion):
		return i18n.TranslateText(locale, "components.bbs.consent.stale_version")
	case errors.Is(err, bbscore.ErrBanned):
		return i18n.TranslateText(locale, "components.bbs.errors.banned")
	case errors.Is(err, bbscore.ErrRateLimited):
		return i18n.TranslateText(locale, "components.bbs.errors.rate_limited")
	case errors.Is(err, bbscore.ErrThreadArchived):
		return i18n.TranslateText(locale, "components.bbs.errors.thread_archived")
	case errors.Is(err, bbscore.ErrMaxResReached):
		return i18n.TranslateText(locale, "components.bbs.errors.max_res_reached")
	case errors.Is(err, bbscore.ErrBoardLocked):
		return i18n.TranslateText(locale, "components.bbs.errors.board_locked")
	case errors.Is(err, bbscore.ErrCapNotAuthorized):
		return i18n.TranslateText(locale, "components.bbs.errors.cap_not_authorized")
	case errors.Is(err, bbscore.ErrUnauthorized):
		return i18n.TranslateText(locale, "components.bbs.errors.unauthorized")
	case errors.Is(err, bbscore.ErrAlreadyExists):
		return i18n.TranslateText(locale, "components.bbs.errors.already_exists")
	case errors.Is(err, bbscore.ErrNotFound):
		return i18n.TranslateText(locale, "components.bbs.errors.not_found")
	case errors.Is(err, discordsync.ErrSourceMountInactive):
		return i18n.TranslateText(locale, "components.bbs.post_ui.mount_missing_or_inactive")
	case errors.Is(err, bbscore.ErrInvalidCategoryName):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_category_name")
	case errors.Is(err, bbscore.ErrInvalidBoardName):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_board_name")
	case errors.Is(err, bbscore.ErrInvalidBoardRules):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_board_rules")
	case errors.Is(err, bbscore.ErrInvalidTitleLength):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_title_length", map[string]any{"max": strconv.Itoa(bbscore.MaxTitleRunes)})
	case errors.Is(err, bbscore.ErrInvalidNameLength):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_name_length", map[string]any{"max": strconv.Itoa(bbscore.MaxNameRunes)})
	case errors.Is(err, bbscore.ErrInvalidBodyLength):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_body_length", map[string]any{"max": strconv.Itoa(bbscore.MaxBodyRunes)})
	case errors.Is(err, bbscore.ErrFormattedBodyTooLong):
		return i18n.TranslateText(locale, "components.bbs.errors.formatted_body_too_long", map[string]any{"max": strconv.Itoa(bbscore.MaxEscapedBodyRunes)})
	case errors.Is(err, bbscore.ErrInvalidNameCommand):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_name_command")
	case errors.Is(err, bbscore.ErrInvalidExtendCommand):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_extend_command")
	case errors.Is(err, bbscore.ErrInvalidReaderJump):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_reader_jump")
	case errors.Is(err, bbscore.ErrInvalidPolicyVersion):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_policy_version", map[string]any{"max": strconv.Itoa(bbscore.MaxPolicyVersionLen)})
	case errors.Is(err, bbscore.ErrReportReasonRequired):
		return i18n.TranslateText(locale, "components.bbs.errors.report_reason_required")
	case errors.Is(err, bbscore.ErrGuildIDRequired):
		return i18n.TranslateText(locale, "components.bbs.errors.guild_id_required")
	case errors.Is(err, bbscore.ErrBoardScopeMismatch):
		return i18n.TranslateText(locale, "components.bbs.errors.board_scope_mismatch")
	case errors.Is(err, bbscore.ErrBoardProviderMismatch):
		return i18n.TranslateText(locale, "components.bbs.errors.board_provider_mismatch")
	case errors.Is(err, bbscore.ErrInvalidBanDuration):
		return i18n.TranslateText(locale, "components.bbs.mod.invalid_hours_range", map[string]any{"max": bbscore.MaxBanHours})
	case errors.Is(err, bbscore.ErrInvalidInput):
		return i18n.TranslateText(locale, "components.bbs.errors.invalid_input")
	default:
		slog.Error("bbs unexpected error", "err", err)
		return i18n.TranslateText(locale, "components.bbs.errors.internal")
	}
}
