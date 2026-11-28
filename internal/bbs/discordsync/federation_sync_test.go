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

package discordsync_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

func TestRenderers_FederationOptionsAndAuthPrompts(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	v := discordsync.PayloadValidator{}
	loc := discord.LocaleJapanese
	board := bbs.Board{
		ID:          uuid.New(),
		Scope:       bbs.ScopeGlobal,
		ProviderKey: "remote:prod",
		Name:        "本番ニュース板",
		Description: "テスト説明",
		Rules:       bbs.BoardRules{MaxRes: 1000}.Normalize(),
	}
	mountID := uuid.New()

	// 1. Read-only remote board guide renders badge, note, and disabled create button.
	roGuide := discordsync.RenderBoardGuideWithOptions(board, mountID, discordsync.BoardGuideOptions{
		ProviderLabel: "prod",
		ReadOnly:      true,
	}, loc)
	if err := v.ValidateMessageCreate(roGuide); err != nil {
		t.Fatalf("ValidateMessageCreate(roGuide) failed: %v", err)
	}
	roJSON, _ := json.Marshal(roGuide)
	roStr := string(roJSON)
	if !strings.Contains(roStr, "[prod]") || !strings.Contains(roStr, `"disabled":true`) {
		t.Fatalf("expected [prod] badge and disabled create button in read-only guide: %s", roStr)
	}

	// 2. Writable remote board guide enables create button.
	rwGuide := discordsync.RenderBoardGuideWithOptions(board, mountID, discordsync.BoardGuideOptions{
		ProviderLabel: "prod",
		ReadOnly:      false,
	}, loc)
	if err := v.ValidateMessageCreate(rwGuide); err != nil {
		t.Fatalf("ValidateMessageCreate(rwGuide) failed: %v", err)
	}

	// 3. Thread control panel honors CanPost=false and CanReport=false.
	mirrorID := uuid.New()
	thread := bbs.Thread{
		ID:       uuid.New(),
		BoardID:  board.ID,
		Title:    "リモートテストスレッド",
		ResCount: 5,
		Rules:    board.Rules,
	}
	roPanel := discordsync.RenderThreadControlPanelWithOptions(mirrorID, thread, discordsync.ThreadControlPanelOptions{
		CanPost:   false,
		CanReport: false,
	}, loc)
	if err := v.ValidateMessageCreate(roPanel); err != nil {
		t.Fatalf("ValidateMessageCreate(roPanel) failed: %v", err)
	}
	panelJSON, _ := json.Marshal(roPanel)
	panelStr := string(panelJSON)
	if !strings.Contains(panelStr, `/bbs/post/new/`+mirrorID.String()+`","disabled":true`) {
		t.Fatalf("expected post button disabled when CanPost=false: %s", panelStr)
	}
	if !strings.Contains(panelStr, `/bbs/report/thread/`+thread.ID.String()+`","disabled":true`) {
		t.Fatalf("expected report button disabled when CanReport=false: %s", panelStr)
	}
	postMenu := discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
		Board:      board,
		Thread:     thread,
		Post:       bbs.Post{ID: uuid.New(), ThreadID: thread.ID, ResNumber: 1, Name: "ななし", Body: "本文"},
		CanReply:   false,
		HideReport: true,
	}, loc)
	if err := v.ValidateMessageCreate(postMenu); err != nil {
		t.Fatalf("ValidateMessageCreate(postMenu) failed: %v", err)
	}
	pmJSON, _ := json.Marshal(postMenu)
	if !strings.Contains(string(pmJSON), `"disabled":true`) {
		t.Fatalf("expected disabled report/reply buttons on read-only PostActionMenu: %s", string(pmJSON))
	}

	// 4. OAuth prompt and completed update validate cleanly.
	authPrompt := discordsync.RenderFederationAuthPrompt("prod", "https://origin.example.com/authorize?state=abc", "/bbs/thread/new/"+mountID.String(), loc)
	if err := v.ValidateMessageCreate(authPrompt); err != nil {
		t.Fatalf("ValidateMessageCreate(authPrompt) failed: %v", err)
	}
	promptJSON, _ := json.Marshal(authPrompt)
	if !strings.Contains(string(promptJSON), "https://origin.example.com/authorize?state=abc") || !strings.Contains(string(promptJSON), "/bbs/thread/new/"+mountID.String()) {
		t.Fatalf("expected auth link and continue custom ID in prompt: %s", string(promptJSON))
	}

	authDone := discordsync.RenderFederationAuthCompleted("prod", "/bbs/thread/new/"+mountID.String(), loc)
	if err := v.ValidateMessageUpdate(authDone, true); err != nil {
		t.Fatalf("ValidateMessageUpdate(authDone) failed: %v", err)
	}
}

type nonModeratableRemoteProvider struct {
	boards   []bbs.Board
	threads  []bbs.Thread
	archived atomic.Int32
}

func (p *nonModeratableRemoteProvider) Key() string { return "remote:readonly" }
func (p *nonModeratableRemoteProvider) Capabilities() bbs.ProviderCapabilities {
	return bbs.ProviderCapabilities{CanModerate: false}
}
func (p *nonModeratableRemoteProvider) GetCategory(context.Context, uuid.UUID) (*bbs.Category, error) {
	return nil, bbs.ErrNotFound
}
func (p *nonModeratableRemoteProvider) ListCategories(context.Context, bbs.CategoryFilter) ([]bbs.Category, error) {
	return nil, nil
}
func (p *nonModeratableRemoteProvider) CreateCategory(context.Context, bbs.CreateCategoryInput) (*bbs.Category, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) UpdateCategory(context.Context, uuid.UUID, bbs.UpdateCategoryInput) (*bbs.Category, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) DeleteCategory(context.Context, uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) GetBoard(_ context.Context, id uuid.UUID) (*bbs.Board, error) {
	for i := range p.boards {
		if p.boards[i].ID == id {
			b := p.boards[i]
			return &b, nil
		}
	}
	return nil, bbs.ErrNotFound
}
func (p *nonModeratableRemoteProvider) ListBoards(context.Context, bbs.BoardFilter) ([]bbs.Board, error) {
	return slices.Clone(p.boards), nil
}
func (p *nonModeratableRemoteProvider) CreateBoard(context.Context, bbs.CreateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) UpdateBoard(context.Context, uuid.UUID, bbs.UpdateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) DeleteBoard(context.Context, uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) GetThread(_ context.Context, id uuid.UUID) (*bbs.Thread, error) {
	for i := range p.threads {
		if p.threads[i].ID == id {
			th := p.threads[i]
			return &th, nil
		}
	}
	return nil, bbs.ErrNotFound
}
func (p *nonModeratableRemoteProvider) ListThreads(context.Context, bbs.ThreadFilter) ([]bbs.Thread, error) {
	return slices.Clone(p.threads), nil
}
func (p *nonModeratableRemoteProvider) CreateThread(context.Context, bbs.CreateThreadRecord) (*bbs.Thread, *bbs.Post, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) ArchiveThread(context.Context, uuid.UUID) (*bbs.Thread, error) {
	p.archived.Add(1)
	return nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) ListStaleThreads(context.Context, time.Time) ([]bbs.Thread, error) {
	return slices.Clone(p.threads), nil
}
func (p *nonModeratableRemoteProvider) GetPost(context.Context, uuid.UUID) (*bbs.Post, error) {
	return nil, bbs.ErrNotFound
}
func (p *nonModeratableRemoteProvider) GetPostByNumber(context.Context, uuid.UUID, uint32) (*bbs.Post, error) {
	return nil, bbs.ErrNotFound
}
func (p *nonModeratableRemoteProvider) ListPosts(context.Context, uuid.UUID, int, int) ([]bbs.Post, error) {
	return nil, nil
}
func (p *nonModeratableRemoteProvider) CreatePost(context.Context, bbs.CreatePostRecord) (*bbs.Post, *bbs.Thread, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}
func (p *nonModeratableRemoteProvider) AbonePost(context.Context, uuid.UUID, string) (*bbs.Post, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func TestSyncer_NotifyWatchersRemotePostAndDatOchiSkip(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	ctx := context.Background()
	store, svc := newSyncerTestEnv(t)

	remoteBoard := bbs.Board{
		ID:          uuid.New(),
		Scope:       bbs.ScopeGlobal,
		ProviderKey: "remote:readonly",
		Name:        "外部監視テスト板",
		Rules:       bbs.BoardRules{MaxRes: 100}.Normalize(),
	}
	remoteThread := bbs.Thread{
		ID:          uuid.New(),
		BoardID:     remoteBoard.ID,
		ProviderKey: "remote:readonly",
		Title:       "外部スレ",
		Status:      bbs.ThreadStatusActive,
		ResCount:    2,
		Rules:       remoteBoard.Rules,
		BumpedAt:    time.Now().Add(-365 * 24 * time.Hour),
	}
	rp := &nonModeratableRemoteProvider{
		boards:  []bbs.Board{remoteBoard},
		threads: []bbs.Thread{remoteThread},
	}
	svc.Registry.Register(rp)

	guildID := snowflake.ID(91001)
	channelID := snowflake.ID(91002)
	discordThreadID := snowflake.ID(91003)
	watcherID := snowflake.ID(91009)
	dmChannelID := snowflake.ID(991100)

	mount := &bbs.BoardMount{
		ID:          uuid.New(),
		BoardID:     remoteBoard.ID,
		ProviderKey: "remote:readonly",
		GuildID:     guildID,
		ChannelID:   channelID,
		Locale:      "ja",
		IsForum:     false,
		Active:      true,
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	tm := &bbs.ThreadMirror{
		ID:              uuid.New(),
		ThreadID:        remoteThread.ID,
		BoardMountID:    mount.ID,
		GuildID:         guildID,
		DiscordThreadID: discordThreadID,
		SyncedUpToRes:   1,
	}
	pm1 := &bbs.PostMirror{
		ID:               uuid.New(),
		PostID:           uuid.New(),
		ThreadID:         remoteThread.ID,
		ResNumber:        1,
		ThreadMirrorID:   tm.ID,
		GuildID:          guildID,
		DiscordMessageID: snowflake.ID(91004),
	}
	if err := store.CreateThreadAndPostMirror(ctx, tm, pm1); err != nil {
		t.Fatalf("CreateThreadAndPostMirror failed: %v", err)
	}
	if err := store.SaveThreadWatch(ctx, &bbs.ThreadWatch{
		ID:       uuid.New(),
		ThreadID: remoteThread.ID,
		UserID:   watcherID,
		Locale:   "ja",
	}); err != nil {
		t.Fatalf("SaveThreadWatch failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, watcherID); err != nil {
		t.Fatalf("AcceptPolicy for watcher failed: %v", err)
	}

	var mu sync.Mutex
	var dmSent bool
	dmCh := make(chan struct{}, 2)

	client := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		switch {
		case req.Method == http.MethodPost && p == "/users/@me/channels":
			return jsonResponse(http.StatusOK, map[string]any{"id": dmChannelID.String(), "type": 1}), nil
		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/messages", dmChannelID):
			mu.Lock()
			dmSent = true
			mu.Unlock()
			select {
			case dmCh <- struct{}{}:
			default:
			}
			return jsonResponse(http.StatusOK, map[string]any{"id": "880099"}), nil
		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/messages", discordThreadID):
			return jsonResponse(http.StatusOK, map[string]any{"id": "880050"}), nil
		case req.Method == http.MethodGet && strings.HasPrefix(p, "/channels/"):
			return jsonResponse(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{"id": "880001"}), nil
	})

	syncer := discordsync.NewSyncer(svc)
	// Do not schedule debounce timers that outlive test.
	syncer.Debouncer = discordsync.NewControlPanelDebouncerWithTimer(time.Hour, func(_ time.Duration, _ func()) discordsync.DebounceTimer {
		return &fakeTimer{}
	})

	// 1. RunDatOchiCheck must skip non-moderatable remote provider even though it returns stale threads.
	if err := syncer.RunDatOchiCheck(ctx, client); err != nil {
		t.Fatalf("RunDatOchiCheck failed: %v", err)
	}
	if rp.archived.Load() != 0 {
		t.Fatalf("expected non-moderatable provider not to be archived, got %d", rp.archived.Load())
	}

	// 2. FanoutPost for a remote post (no local bbs_posts / GetAuditPostInfo == ErrNotFound) must notify watchers!
	remotePost2 := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  remoteThread.ID,
		ResNumber: 2,
		Name:      "風吹けば名無し",
		AnonID:    "REMOTE123",
		Body:      "リモートのレス",
		CreatedAt: time.Now(),
	}
	syncer.FanoutPost(ctx, client, remoteBoard, remoteThread, remotePost2, nil, discord.LocaleJapanese)

	select {
	case <-dmCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for DM notification to thread watcher for remote post")
	}
	mu.Lock()
	gotDM := dmSent
	mu.Unlock()
	if !gotDM {
		t.Fatal("expected watcher DM to be sent for remote post without local audit info")
	}
}
