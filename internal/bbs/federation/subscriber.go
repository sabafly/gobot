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

package federation

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	minReconnectBackoff = 1 * time.Second
	maxReconnectBackoff = 60 * time.Second
	maxSSETokenBytes    = 4 * 1024 * 1024
)

// Subscriber maintains a persistent SSE event stream against Origin and drives idempotent catch-up
// and live updates into the Listener's discordsync.Syncer.
type Subscriber struct {
	provider *RemoteProvider
	svc      *bbs.Service
	syncer   *discordsync.Syncer

	mu        sync.Mutex
	lastSeq   uint64
	lastEpoch string
	needSync  bool
}

// NewSubscriber constructs a federation SSE subscriber for provider.
func NewSubscriber(provider *RemoteProvider, svc *bbs.Service, syncer *discordsync.Syncer) *Subscriber {
	return &Subscriber{
		provider: provider,
		svc:      svc,
		syncer:   syncer,
		needSync: true,
	}
}

func jitterDuration(base time.Duration) time.Duration {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return base
	}
	v := binary.LittleEndian.Uint32(b[:])
	// Add up to +25% jitter
	jitter := time.Duration(v % uint32(max(base/4, time.Millisecond)))
	return base + jitter
}

// Run loops until ctx is cancelled, streaming Origin SSE events and catching up on reconnects.
func (sub *Subscriber) Run(ctx context.Context, client *bot.Client) {
	backoff := minReconnectBackoff
	for {
		if ctx.Err() != nil {
			return
		}
		connectedCleanly := sub.connectOnce(ctx, client)
		if ctx.Err() != nil {
			return
		}
		if connectedCleanly {
			backoff = minReconnectBackoff
		}
		wait := jitterDuration(backoff)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(backoff*2, maxReconnectBackoff)
	}
}

func (sub *Subscriber) connectOnce(ctx context.Context, client *bot.Client) bool {
	caps, err := sub.provider.Client().GetCapabilities(ctx)
	if err != nil {
		slog.Warn("bbs federation failed to fetch origin capabilities", "remote", sub.provider.RemoteKey(), "err", err)
		return false
	}
	prevCaps := sub.provider.Capabilities()
	sub.provider.UpdateCapabilities(*caps)
	nextCaps := sub.provider.Capabilities()
	capsChanged := prevCaps.CanCreateThread != nextCaps.CanCreateThread || prevCaps.CanCreatePost != nextCaps.CanCreatePost

	sub.mu.Lock()
	afterSeq := sub.lastSeq
	epoch := sub.lastEpoch
	needResync := sub.needSync || epoch == "" || epoch != caps.InstanceEpoch
	if needResync {
		afterSeq = caps.LatestSeq
		epoch = caps.InstanceEpoch
	}
	sub.mu.Unlock()

	if capsChanged && !needResync {
		sub.refreshMountedGuides(ctx, client)
	}

	resp, err := sub.provider.Client().OpenEventsStream(ctx, afterSeq, epoch)
	if err != nil {
		slog.Warn("bbs federation SSE connect failed", "remote", sub.provider.RemoteKey(), "err", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if needResync {
		sub.resyncMountedBoards(ctx, client)
		if ctx.Err() == nil {
			sub.mu.Lock()
			sub.lastEpoch = caps.InstanceEpoch
			if caps.LatestSeq > sub.lastSeq {
				sub.lastSeq = caps.LatestSeq
			}
			sub.needSync = false
			sub.mu.Unlock()
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSETokenBytes)

	var evName, evID, evData strings.Builder
	dispatchFrame := func() bool {
		if evData.Len() == 0 && evName.Len() == 0 && evID.Len() == 0 {
			return true
		}
		name := strings.TrimSpace(evName.String())
		idStr := strings.TrimSpace(evID.String())
		dataStr := strings.TrimSpace(evData.String())
		evName.Reset()
		evID.Reset()
		evData.Reset()

		if dataStr == "" {
			return true
		}

		switch name {
		case "resync":
			var meta struct {
				Epoch     string `json:"epoch"`
				LatestSeq uint64 `json:"latest_seq"`
			}
			_ = json.Unmarshal([]byte(dataStr), &meta)
			sub.resyncMountedBoards(ctx, client)
			sub.mu.Lock()
			if meta.Epoch != "" {
				sub.lastEpoch = meta.Epoch
			}
			if meta.LatestSeq > 0 {
				sub.lastSeq = meta.LatestSeq
			}
			sub.needSync = false
			sub.mu.Unlock()
			return true
		case "ready":
			var meta struct {
				Epoch     string `json:"epoch"`
				LatestSeq uint64 `json:"latest_seq"`
			}
			if uErr := json.Unmarshal([]byte(dataStr), &meta); uErr == nil {
				sub.mu.Lock()
				if meta.Epoch != "" {
					sub.lastEpoch = meta.Epoch
				}
				if meta.LatestSeq > sub.lastSeq && sub.lastSeq == 0 {
					sub.lastSeq = meta.LatestSeq
				}
				sub.mu.Unlock()
			}
			return true
		}

		var we WireEvent
		if uErr := json.Unmarshal([]byte(dataStr), &we); uErr != nil {
			slog.Warn("bbs federation SSE invalid payload", "remote", sub.provider.RemoteKey(), "event", name, "err", uErr)
			return true
		}
		if idStr != "" {
			if parsedID, pErr := strconv.ParseUint(idStr, 10, 64); pErr == nil && we.Seq == 0 {
				we.Seq = parsedID
			}
		}

		sub.handleWireEvent(ctx, client, we)

		sub.mu.Lock()
		if we.Epoch != "" {
			sub.lastEpoch = we.Epoch
		}
		if we.Seq > sub.lastSeq {
			sub.lastSeq = we.Seq
		}
		sub.mu.Unlock()
		return true
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if !dispatchFrame() {
				break
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, val, hasColon := strings.Cut(line, ":")
		if hasColon {
			val = strings.TrimPrefix(val, " ")
		} else {
			field = line
			val = ""
		}
		switch field {
		case "event":
			evName.Reset()
			evName.WriteString(val)
		case "id":
			evID.Reset()
			evID.WriteString(val)
		case "data":
			if evData.Len() > 0 {
				evData.WriteByte('\n')
			}
			evData.WriteString(val)
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		slog.Warn("bbs federation SSE stream ended with error", "remote", sub.provider.RemoteKey(), "err", err)
	}
	return true
}

func (sub *Subscriber) hasActiveMountsForBoard(ctx context.Context, boardID uuid.UUID) bool {
	if boardID == uuid.Nil || sub.svc == nil || sub.svc.Store == nil {
		return false
	}
	mounts, err := sub.svc.Store.ListActiveBoardMountsByBoard(ctx, boardID)
	return err == nil && len(mounts) > 0
}

func (sub *Subscriber) handleWireEvent(ctx context.Context, client *bot.Client, we WireEvent) {
	if we.Board != nil {
		b := sub.provider.rewriteBoard(*we.Board)
		we.Board = &b
	}
	if we.Thread != nil {
		t := sub.provider.rewriteThread(*we.Thread)
		we.Thread = &t
	}

	switch we.Type {
	case bbs.EventBoardUpdated:
		sub.provider.InvalidateMetadataCache()
		if sub.syncer != nil && client != nil {
			sub.refreshBoardMountsGuides(ctx, client, we.BoardID)
		}
	case bbs.EventBoardDeleted:
		sub.provider.InvalidateMetadataCache()
		if mounts, err := sub.svc.Store.ListActiveBoardMountsByBoard(ctx, we.BoardID); err == nil {
			for _, m := range mounts {
				_ = sub.svc.Store.DeactivateBoardMount(ctx, m.ID)
			}
		}
	case bbs.EventThreadCreated:
		if sub.syncer == nil || client == nil || we.Board == nil || we.Thread == nil || we.Post == nil {
			return
		}
		if !sub.hasActiveMountsForBoard(ctx, we.BoardID) {
			return
		}
		sub.syncer.FanoutThread(ctx, client, *we.Board, *we.Thread, *we.Post, i18n.DefaultLocale())
	case bbs.EventPostCreated:
		if sub.syncer == nil || client == nil || we.Board == nil || we.Thread == nil || we.Post == nil {
			return
		}
		if !sub.hasActiveMountsForBoard(ctx, we.BoardID) {
			return
		}
		sub.syncer.FanoutPost(ctx, client, *we.Board, *we.Thread, *we.Post, we.BackwardRefs, i18n.DefaultLocale())
	case bbs.EventPostAboned:
		if sub.syncer == nil || client == nil || we.Thread == nil || we.Post == nil {
			return
		}
		if err := sub.syncer.SyncAbone(ctx, client, *we.Post, *we.Thread); err != nil {
			slog.Warn("bbs federation sync abone failed", "post_id", we.Post.ID, "err", err)
		}
	case bbs.EventThreadArchived:
		if sub.syncer == nil || client == nil || we.Board == nil || we.Thread == nil {
			return
		}
		sub.syncer.SyncArchive(ctx, client, *we.Board, *we.Thread, we.HitMaxRes)
	}
}

func (sub *Subscriber) refreshMountedGuides(ctx context.Context, client *bot.Client) {
	if sub.syncer == nil || client == nil || sub.svc == nil {
		return
	}
	active := true
	mounts, err := sub.svc.Store.ListBoardMounts(ctx, bbs.BoardMountFilter{Active: &active})
	if err != nil {
		return
	}
	for _, m := range mounts {
		if m.ProviderKey != sub.provider.Key() {
			continue
		}
		sub.syncer.RefreshBoardGuide(ctx, client, m.GuildID, m.BoardID, discord.Locale(m.Locale))
	}
}

func (sub *Subscriber) refreshBoardMountsGuides(ctx context.Context, client *bot.Client, boardID uuid.UUID) {
	mounts, err := sub.svc.Store.ListActiveBoardMountsByBoard(ctx, boardID)
	if err != nil {
		return
	}
	for _, m := range mounts {
		sub.syncer.RefreshBoardGuide(ctx, client, m.GuildID, m.BoardID, discord.Locale(m.Locale))
	}
}

func (sub *Subscriber) resyncMountedBoards(ctx context.Context, client *bot.Client) {
	if sub.svc == nil || sub.svc.Store == nil {
		return
	}
	sub.provider.InvalidateMetadataCache()
	active := true
	mounts, err := sub.svc.Store.ListBoardMounts(ctx, bbs.BoardMountFilter{Active: &active})
	if err != nil {
		slog.Warn("bbs federation resync failed to list mounts", "remote", sub.provider.RemoteKey(), "err", err)
		return
	}

	for _, m := range mounts {
		if ctx.Err() != nil {
			return
		}
		if m.ProviderKey != sub.provider.Key() {
			continue
		}
		board, bErr := sub.provider.GetBoard(ctx, m.BoardID)
		if bErr != nil {
			if errors.Is(bErr, bbs.ErrNotFound) {
				_ = sub.svc.Store.DeactivateBoardMount(ctx, m.ID)
			}
			continue
		}
		if sub.syncer != nil && client != nil {
			sub.syncer.RefreshBoardGuide(ctx, client, m.GuildID, m.BoardID, discord.Locale(m.Locale))
		}

		mirrors, tmErr := sub.svc.Store.ListThreadMirrorsByMount(ctx, m.ID)
		if tmErr != nil {
			continue
		}
		mirroredThreadIDs := make(map[uuid.UUID]struct{}, len(mirrors))
		for _, mirror := range mirrors {
			mirroredThreadIDs[mirror.ThreadID] = struct{}{}
			if ctx.Err() != nil {
				return
			}
			thread, tErr := sub.provider.GetThread(ctx, mirror.ThreadID)
			if tErr != nil {
				continue
			}
			posts, pErr := sub.provider.Client().ListPostsAfterRes(ctx, mirror.ThreadID, mirror.SyncedUpToRes)
			if pErr != nil {
				slog.Warn("bbs federation resync posts failed", "thread_id", mirror.ThreadID, "err", pErr)
				continue
			}
			if sub.syncer != nil && client != nil {
				for _, post := range posts {
					sub.syncer.FanoutPost(ctx, client, *board, *thread, post, nil, discord.Locale(m.Locale))
				}
				if thread.Status == bbs.ThreadStatusArchived && !mirror.ArchivedSynced {
					sub.syncer.SyncArchive(ctx, client, *board, *thread, thread.ResCount >= thread.Rules.Normalize().MaxRes)
				}
			}
		}

		activeStatus := bbs.ThreadStatusActive
		activeThreads, ltErr := sub.provider.ListThreads(ctx, bbs.ThreadFilter{
			BoardID: m.BoardID,
			Status:  &activeStatus,
			Limit:   50,
		})
		if ltErr != nil {
			slog.Warn("bbs federation resync active threads failed", "board_id", m.BoardID, "err", ltErr)
			continue
		}
		for _, th := range activeThreads {
			if ctx.Err() != nil {
				return
			}
			if _, ok := mirroredThreadIDs[th.ID]; ok {
				continue
			}
			if th.CreatedAt.Before(m.CreatedAt) {
				continue
			}
			posts, pErr := sub.provider.Client().ListPostsAfterRes(ctx, th.ID, 0)
			if pErr != nil {
				slog.Warn("bbs federation resync new thread posts failed", "thread_id", th.ID, "err", pErr)
				continue
			}
			if len(posts) == 0 {
				continue
			}
			if sub.syncer != nil && client != nil {
				sub.syncer.FanoutThread(ctx, client, *board, th, posts[0], discord.Locale(m.Locale))
				for _, post := range posts[1:] {
					sub.syncer.FanoutPost(ctx, client, *board, th, post, nil, discord.Locale(m.Locale))
				}
			}
		}
	}
}
