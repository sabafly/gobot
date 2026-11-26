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

package bbs_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

func TestIdentityGenerator(t *testing.T) {
	gen := bbs.NewIdentityGenerator([]byte("test-secret-key-for-bbs"))
	boardA := uuid.New()
	boardB := uuid.New()
	user1 := snowflake.ID(1001)
	user2 := snowflake.ID(1002)
	day1 := time.Date(2026, 11, 25, 12, 0, 0, 0, bbs.JSTLocation)
	day2 := time.Date(2026, 11, 26, 12, 0, 0, 0, bbs.JSTLocation)

	id1a := gen.GenerateAnonID(boardA, user1, day1)
	id1aRepeat := gen.GenerateAnonID(boardA, user1, day1.Add(3*time.Hour))
	id1b := gen.GenerateAnonID(boardB, user1, day1)
	id1day2 := gen.GenerateAnonID(boardA, user1, day2)
	id2a := gen.GenerateAnonID(boardA, user2, day1)

	if len(id1a) != 8 {
		t.Fatalf("expected 8-char AnonID, got %q", id1a)
	}
	if id1a != id1aRepeat {
		t.Fatalf("same board and JST date must match: %q != %q", id1a, id1aRepeat)
	}
	if id1a == id1b {
		t.Fatalf("different boards should yield different IDs: %q", id1a)
	}
	if id1a == id1day2 {
		t.Fatalf("different JST dates should yield different IDs: %q", id1a)
	}
	if id1a == id2a {
		t.Fatalf("different users should yield different IDs: %q", id1a)
	}

	// Tripcode + commands + extend
	parsed, err := gen.ParsePostIdentity(
		boardA,
		bbs.ScopeLocal,
		snowflake.ID(999),
		user1,
		"名無しの権兵衛#mysecret",
		"!extend:checked:vvvvv\n本文テスト",
		bbs.BoardRules{HideID: true, SLIP: bbs.SLIPNone},
		true,
		nil,
		day1,
	)
	if err != nil {
		t.Fatalf("ParsePostIdentity unexpected error: %v", err)
	}
	if parsed.DisplayName != "名無しの権兵衛" {
		t.Fatalf("unexpected display name: %q", parsed.DisplayName)
	}
	if !strings.HasPrefix(parsed.Trip, "◆") || len([]rune(parsed.Trip)) != 11 {
		t.Fatalf("unexpected tripcode: %q", parsed.Trip)
	}
	// Tripcode uses HMAC-SHA256 with server secret: different secret must produce a different tripcode.
	otherGen := bbs.NewIdentityGenerator([]byte("different-secret-key-for-bbs"))
	if parsed.Trip != gen.GenerateTrip("mysecret") {
		t.Fatalf("GenerateTrip must be deterministic for the same secret and key")
	}
	if parsed.Trip == otherGen.GenerateTrip("mysecret") {
		t.Fatalf("GenerateTrip must depend on server secret to resist offline dictionary attacks")
	}
	if parsed.AnonID == "" || parsed.SlipText == "" {
		t.Fatalf("expected !extend:checked:vvvvv to enable AnonID and SLIP")
	}
	if !strings.HasPrefix(parsed.ExtDat, "EXTDAT:checked:vvvvv:") {
		t.Fatalf("unexpected EXTDAT: %q", parsed.ExtDat)
	}
	if parsed.CleanBody != "本文テスト" {
		t.Fatalf("unexpected clean body: %q", parsed.CleanBody)
	}

	// Plain "sage" is now treated as an ordinary name; "!sage" is rejected.
	sageName, err := gen.ParsePostIdentity(
		boardA,
		bbs.ScopeGlobal,
		snowflake.ID(999),
		user1,
		"sage",
		"  hello  ",
		bbs.BoardRules{DefaultName: "風吹けば名無し"},
		false,
		nil,
		day1,
	)
	if err != nil || sageName.DisplayName != "sage" {
		t.Fatalf("expected plain sage to remain literal name, got %+v, err=%v", sageName, err)
	}

	// Spoof symbols (◆/★ and lookalikes) are neutralized in names and board default names.
	spoofed, err := gen.ParsePostIdentity(
		boardA,
		bbs.ScopeLocal,
		999,
		user1,
		"偽◆名♦\uFE0F❖⬥★✦⭑⭐#mysecret",
		"body",
		bbs.BoardRules{DefaultName: "板◆デフォ★ルト"},
		false,
		nil,
		day1,
	)
	if err != nil {
		t.Fatalf("ParsePostIdentity with spoofed symbols failed: %v", err)
	}
	if spoofed.DisplayName != "偽◇名◇◇◇☆☆☆☆" {
		t.Fatalf("expected neutralized name, got %q", spoofed.DisplayName)
	}
	if spoofed.Trip != parsed.Trip {
		t.Fatalf("expected trip generation from #mysecret unaffected by name sanitization, got %q vs %q", spoofed.Trip, parsed.Trip)
	}
	if normDefault := (bbs.BoardRules{DefaultName: "板◆デフォ★ルト"}).Normalize().DefaultName; normDefault != "板◇デフォ☆ルト" {
		t.Fatalf("expected sanitized BoardRules.DefaultName, got %q", normDefault)
	}

	// Strict rejection cases: unrecognized ! command (!sage, !id:off) and malformed/reply !extend
	for _, badName := range []string{"!id:off", "!sage"} {
		if _, err := gen.ParsePostIdentity(boardA, bbs.ScopeLocal, 999, user1, badName, "body", bbs.BoardRules{}, false, nil, day1); !errors.Is(err, bbs.ErrInvalidNameCommand) {
			t.Fatalf("expected ErrInvalidNameCommand for %q, got %v", badName, err)
		}
	}
	for _, badBody := range []string{
		"!extend:checked:vvvvv",
		"!extend:bad:none\nhi",
		"!extend:checked:bad\nhi",
		"!extend:checked:none extra\nhi",
	} {
		if _, err := gen.ParsePostIdentity(boardA, bbs.ScopeLocal, 999, user1, "", badBody, bbs.BoardRules{}, true, nil, day1); !errors.Is(err, bbs.ErrInvalidExtendCommand) {
			t.Fatalf("expected ErrInvalidExtendCommand for %q on new thread, got %v", badBody, err)
		}
	}
	if _, err := gen.ParsePostIdentity(boardA, bbs.ScopeLocal, 999, user1, "", "!extend:checked:none\nhi", bbs.BoardRules{}, false, nil, day1); !errors.Is(err, bbs.ErrInvalidExtendCommand) {
		t.Fatalf("expected ErrInvalidExtendCommand for !extend in reply, got %v", err)
	}
}

func TestCapValidationAndIdentity(t *testing.T) {
	gen := bbs.NewIdentityGenerator([]byte("cap-identity-secret"))
	boardID := uuid.New()
	uid := snowflake.ID(555)
	day := time.Date(2026, 11, 26, 12, 0, 0, 0, bbs.JSTLocation)

	// Valid definitions normalize whitespace, sanitize reserved marks in Name, and dedup users.
	valid, err := bbs.ValidateCapDefinitions([]bbs.CapDefinition{
		{
			ID:             " admin ",
			Name:           "管理人★◆",
			Description:    "  運営公式  ",
			Users:          []snowflake.ID{0, 1001, 1001, 1002},
			GrantOperators: true,
		},
	})
	if err != nil {
		t.Fatalf("ValidateCapDefinitions valid failed: %v", err)
	}
	if len(valid) != 1 || valid[0].ID != "admin" || valid[0].Name != "管理人☆◇" || valid[0].Description != "運営公式" || !slices.Equal(valid[0].Users, []snowflake.ID{1001, 1002}) {
		t.Fatalf("unexpected normalized cap definitions: %+v", valid)
	}

	// Invalid cap definitions fail validation.
	badCases := [][]bbs.CapDefinition{
		{{ID: "", Name: "管理人"}},
		{{ID: bbs.CapSelectNoneValue, Name: "管理人"}},
		{{ID: strings.Repeat("a", bbs.MaxCapIDRunes+1), Name: "管理人"}},
		{{ID: "dup", Name: "A"}, {ID: "dup", Name: "B"}},
		{{ID: "empty_name", Name: "   "}},
		{{ID: "long_name", Name: strings.Repeat("名", bbs.MaxCapNameRunes+1)}},
		{{ID: "long_desc", Name: "OK", Description: strings.Repeat("説明", bbs.MaxCapDescriptionRunes)}},
	}
	for i, tc := range badCases {
		if _, err := bbs.ValidateCapDefinitions(tc); !errors.Is(err, bbs.ErrInvalidCapConfig) {
			t.Fatalf("case %d: expected ErrInvalidCapConfig, got %v", i, err)
		}
	}

	// Cap identity: empty name stays empty (not board default), ID & SLIP are always hidden even with !id:on or !extend:checked:vvvvv.
	cap := &bbs.Cap{ID: "admin", Name: "管理人"}
	capOnly, err := gen.ParsePostIdentity(
		boardID, bbs.ScopeGlobal, 0, uid,
		"!id:on", "!extend:checked:vvvvv\n告知本文",
		bbs.BoardRules{DefaultName: "風吹けば名無し", HideID: false, SLIP: bbs.SLIPVVVVV},
		true, cap, day,
	)
	if err != nil {
		t.Fatalf("ParsePostIdentity with cap failed: %v", err)
	}
	if capOnly.DisplayName != "" || capOnly.CapName != "管理人" {
		t.Fatalf("expected empty DisplayName and CapName=管理人, got %+v", capOnly)
	}
	if capOnly.AnonID != "" || capOnly.SlipText != "" {
		t.Fatalf("expected AnonID and SlipText hidden when using cap, got AnonID=%q SlipText=%q", capOnly.AnonID, capOnly.SlipText)
	}
	if capOnly.ThreadRules.HideID || capOnly.ThreadRules.SLIP != bbs.SLIPVVVVV {
		t.Fatalf("expected !extend thread rules preserved for replies even when OP used cap, got %+v", capOnly.ThreadRules)
	}

	// Custom name + trip + cap preserves custom name & trip while hiding ID.
	capNamed, err := gen.ParsePostIdentity(
		boardID, bbs.ScopeLocal, 999, uid,
		"さばふらい#key", "レス本文",
		bbs.BoardRules{HideID: false, SLIP: bbs.SLIPVVVVV},
		false, cap, day,
	)
	if err != nil {
		t.Fatalf("ParsePostIdentity with named cap failed: %v", err)
	}
	if capNamed.DisplayName != "さばふらい" || capNamed.CapName != "管理人" || !strings.HasPrefix(capNamed.Trip, "◆") {
		t.Fatalf("unexpected named cap identity: %+v", capNamed)
	}
	if capNamed.AnonID != "" || capNamed.SlipText != "" {
		t.Fatalf("expected hidden ID/SLIP on named cap post, got AnonID=%q SlipText=%q", capNamed.AnonID, capNamed.SlipText)
	}
}

func TestFormatAndMediaExtraction(t *testing.T) {
	raw := "@everyone # title\n> quote\n>>1 >>15 >>999 >>10000\nhttps://example.com/a!b.png https://example.technology/b.png https://example.com/page https://pbs.twimg.com/media/abc?format=jpg&name=large"
	var resolved []uint32
	formatted := bbs.RenderFormattedBody(raw, 10, func(resNum uint32) (string, bool) {
		resolved = append(resolved, resNum)
		return "https://discord.com/channels/1/2/" + string(rune('0'+resNum%10)), true
	})

	if strings.Contains(formatted, "@everyone") && !strings.Contains(formatted, "\\@everyone") {
		t.Fatalf("expected @everyone to be escaped, got %q", formatted)
	}
	if !slices.Equal(resolved, []uint32{1}) {
		t.Fatalf("expected resolver only called for in-range anchors [1], got %v", resolved)
	}
	if !strings.Contains(formatted, "[>>1](<https://discord.com/channels/1/2/1>)") {
		t.Fatalf("expected resolved >>1 jump link, got %q", formatted)
	}
	if strings.Contains(formatted, "[>>15]") || !strings.Contains(formatted, ">>15") {
		t.Fatalf("expected out-of-range >>15 to remain plain text, got %q", formatted)
	}
	if !strings.Contains(formatted, ">>10000") || strings.Contains(formatted, "[>>1000]") {
		t.Fatalf("expected >>10000 not to partially match >>1000, got %q", formatted)
	}
	if !strings.Contains(formatted, "<https://example.com/a!b.png>") || !strings.Contains(formatted, "<https://example.technology/b.png>") {
		t.Fatalf("expected URLs (including ! path char and long TLD) wrapped in <...>, got %q", formatted)
	}

	if anchors10 := bbs.ExtractAnchors(">>1 >>1 >>5 >>20", 10); !slices.Equal(anchors10, []uint32{1, 5}) {
		t.Fatalf("unexpected anchors under maxRes=10: %v", anchors10)
	}
	anchors := bbs.ExtractAnchors(">>1 >>1 >>5 >>20 >>1000 >>10000", 1000)
	if !slices.Equal(anchors, []uint32{1, 5, 20, 1000}) {
		t.Fatalf("unexpected anchors (must ignore >>10000 without partial match): %v", anchors)
	}

	media := bbs.ExtractMediaURLs(
		"a https://example.com/1.PNG https://example.com/1.PNG https://example.com/a!b.png https://example.technology/longtld.png https://example.com/doc.pdf " +
			"https://pbs.twimg.com/media/X?format=webp " +
			"https://example.com/v.mp4 ftp://example.com/bad.png",
	)
	expected := []string{
		"https://example.com/1.PNG",
		"https://example.com/a!b.png",
		"https://example.technology/longtld.png",
		"https://pbs.twimg.com/media/X?format=webp",
		"https://example.com/v.mp4",
	}
	if !slices.Equal(media, expected) {
		t.Fatalf("ExtractMediaURLs = %v, want %v", media, expected)
	}

	// Verify 10-item cap
	var many strings.Builder
	for i := 1; i <= 15; i++ {
		many.WriteString(" https://cdn.example.com/img")
		many.WriteByte(byte('a' + i))
		many.WriteString(".jpg")
	}
	if got := bbs.ExtractMediaURLs(many.String()); len(got) != 10 {
		t.Fatalf("expected 10 media URLs max, got %d", len(got))
	}
}

func TestPolicyManager(t *testing.T) {
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v2"})
	if !pm.NeedsConsent(nil) {
		t.Fatal("nil consent must require consent")
	}
	if !pm.NeedsConsent(&bbs.UserConsent{PolicyVersion: "v1"}) {
		t.Fatal("outdated consent must require re-consent")
	}
	if pm.NeedsConsent(&bbs.UserConsent{PolicyVersion: "v2"}) {
		t.Fatal("matching consent version must pass")
	}
	if doc := pm.GetDocument(bbs.PolicyDocTerms, "ja"); !strings.Contains(doc, "利用規約") {
		t.Fatalf("expected Japanese terms doc, got %q", doc)
	}
	if doc := pm.GetDocument(bbs.PolicyDocPrivacy, "en-US"); !strings.Contains(doc, "Privacy Policy") {
		t.Fatalf("expected English privacy doc, got %q", doc)
	}
	if err := bbs.ValidatePolicyConfig(bbs.PolicyConfig{Version: strings.Repeat("a", bbs.MaxPolicyVersionLen)}); err != nil {
		t.Fatalf("expected %d-char policy version to pass validation: %v", bbs.MaxPolicyVersionLen, err)
	}
	if err := bbs.ValidatePolicyConfig(bbs.PolicyConfig{Version: strings.Repeat("a", bbs.MaxPolicyVersionLen+1)}); !errors.Is(err, bbs.ErrInvalidPolicyVersion) {
		t.Fatalf("expected ErrInvalidPolicyVersion for %d+1 chars, got %v", bbs.MaxPolicyVersionLen, err)
	}
}

// mockThirdPartyProvider simulates a 3rd-party external BBS mounted under a global category.
type mockThirdPartyProvider struct {
	key        string
	categories map[uuid.UUID]bbs.Category
	boards     map[uuid.UUID]bbs.Board
	threads    map[uuid.UUID]bbs.Thread
	posts      map[uuid.UUID][]bbs.Post
}

func (m *mockThirdPartyProvider) Key() string { return m.key }
func (m *mockThirdPartyProvider) Capabilities() bbs.ProviderCapabilities {
	return bbs.ProviderCapabilities{CanCreateBoard: false, CanCreateThread: false, CanCreatePost: false}
}
func (m *mockThirdPartyProvider) GetCategory(_ context.Context, id uuid.UUID) (*bbs.Category, error) {
	c, ok := m.categories[id]
	if !ok {
		return nil, bbs.ErrNotFound
	}
	return &c, nil
}
func (m *mockThirdPartyProvider) ListCategories(_ context.Context, filter bbs.CategoryFilter) ([]bbs.Category, error) {
	var out []bbs.Category
	for _, c := range m.categories {
		if filter.Scope != nil && c.Scope != *filter.Scope {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}
func (m *mockThirdPartyProvider) CreateCategory(_ context.Context, _ bbs.CreateCategoryInput) (*bbs.Category, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) DeleteCategory(_ context.Context, _ uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) GetBoard(_ context.Context, id uuid.UUID) (*bbs.Board, error) {
	b, ok := m.boards[id]
	if !ok {
		return nil, bbs.ErrNotFound
	}
	return &b, nil
}
func (m *mockThirdPartyProvider) ListBoards(_ context.Context, filter bbs.BoardFilter) ([]bbs.Board, error) {
	var out []bbs.Board
	for _, b := range m.boards {
		if filter.CategoryID != nil && b.CategoryID != *filter.CategoryID {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}
func (m *mockThirdPartyProvider) CreateBoard(_ context.Context, _ bbs.CreateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) UpdateBoard(_ context.Context, _ uuid.UUID, _ bbs.UpdateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) DeleteBoard(_ context.Context, _ uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) GetThread(_ context.Context, id uuid.UUID) (*bbs.Thread, error) {
	th, ok := m.threads[id]
	if !ok {
		return nil, bbs.ErrNotFound
	}
	return &th, nil
}
func (m *mockThirdPartyProvider) ListThreads(_ context.Context, filter bbs.ThreadFilter) ([]bbs.Thread, error) {
	var out []bbs.Thread
	for _, th := range m.threads {
		if th.BoardID == filter.BoardID {
			out = append(out, th)
		}
	}
	return out, nil
}
func (m *mockThirdPartyProvider) CreateThread(_ context.Context, _ bbs.CreateThreadRecord) (*bbs.Thread, *bbs.Post, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) ArchiveThread(_ context.Context, _ uuid.UUID) (*bbs.Thread, error) {
	return nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) ListStaleThreads(_ context.Context, _ time.Time) ([]bbs.Thread, error) {
	return nil, nil
}
func (m *mockThirdPartyProvider) GetPost(_ context.Context, id uuid.UUID) (*bbs.Post, error) {
	for _, list := range m.posts {
		for _, p := range list {
			if p.ID == id {
				return &p, nil
			}
		}
	}
	return nil, bbs.ErrNotFound
}
func (m *mockThirdPartyProvider) GetPostByNumber(_ context.Context, threadID uuid.UUID, resNumber uint32) (*bbs.Post, error) {
	for _, p := range m.posts[threadID] {
		if p.ResNumber == resNumber {
			return &p, nil
		}
	}
	return nil, bbs.ErrNotFound
}
func (m *mockThirdPartyProvider) ListPosts(_ context.Context, threadID uuid.UUID, offset, limit int) ([]bbs.Post, error) {
	list := m.posts[threadID]
	if offset >= len(list) {
		return nil, nil
	}
	end := min(offset+limit, len(list))
	return list[offset:end], nil
}
func (m *mockThirdPartyProvider) CreatePost(_ context.Context, _ bbs.CreatePostRecord) (*bbs.Post, *bbs.Thread, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}
func (m *mockThirdPartyProvider) AbonePost(_ context.Context, _ uuid.UUID, _ string) (*bbs.Post, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func TestThirdPartyProviderExtensibility(t *testing.T) {
	catID := uuid.New()
	boardID := uuid.New()
	threadID := uuid.New()
	ext := &mockThirdPartyProvider{
		key: "external-2ch",
		categories: map[uuid.UUID]bbs.Category{
			catID: {ID: catID, Scope: bbs.ScopeGlobal, ProviderKey: "external-2ch", ExternalID: "ext-cat", Name: "外部カテゴリ"},
		},
		boards: map[uuid.UUID]bbs.Board{
			boardID: {ID: boardID, CategoryID: catID, Scope: bbs.ScopeGlobal, ProviderKey: "external-2ch", ExternalID: "news", Name: "外部ニュース板"},
		},
		threads: map[uuid.UUID]bbs.Thread{
			threadID: {ID: threadID, BoardID: boardID, ProviderKey: "external-2ch", ExternalID: "1700000000", Title: "外部スレッド", Status: bbs.ThreadStatusActive, ResCount: 1},
		},
		posts: map[uuid.UUID][]bbs.Post{
			threadID: {{ID: uuid.New(), ThreadID: threadID, ResNumber: 1, Name: "外部名無し", Body: "テスト投稿"}},
		},
	}

	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(ext)
	svc := bbs.NewService(reg, nil, bbs.NewIdentityGenerator([]byte("k")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)

	ctx := context.Background()
	scope := bbs.ScopeGlobal
	cats, err := svc.ListCategories(ctx, bbs.CategoryFilter{Scope: &scope})
	if err != nil || len(cats) != 1 || cats[0].ProviderKey != "external-2ch" {
		t.Fatalf("unexpected external categories: %+v, err=%v", cats, err)
	}
	boards, err := svc.ListBoards(ctx, bbs.BoardFilter{CategoryID: &catID})
	if err != nil || len(boards) != 1 || boards[0].Name != "外部ニュース板" {
		t.Fatalf("unexpected external boards: %+v, err=%v", boards, err)
	}
	threads, err := svc.ListThreads(ctx, bbs.ThreadFilter{BoardID: boardID})
	if err != nil || len(threads) != 1 || threads[0].Title != "外部スレッド" {
		t.Fatalf("unexpected external threads: %+v, err=%v", threads, err)
	}
	posts, err := svc.ListPosts(ctx, threadID, 0, 10)
	if err != nil || len(posts) != 1 || posts[0].Body != "テスト投稿" {
		t.Fatalf("unexpected external posts: %+v, err=%v", posts, err)
	}
	if _, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{CategoryID: catID, Scope: bbs.ScopeGlobal, Name: "new"}); !errors.Is(err, bbs.ErrUnsupportedOperation) {
		t.Fatalf("expected ErrUnsupportedOperation for read-only provider, got %v", err)
	}
	if _, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID:  catID,
		Scope:       bbs.ScopeGlobal,
		ProviderKey: bbs.DefaultProviderKey,
		Name:        "mismatch",
	}); !errors.Is(err, bbs.ErrBoardProviderMismatch) {
		t.Fatalf("expected ErrBoardProviderMismatch for mismatched provider key, got %v", err)
	}
}
