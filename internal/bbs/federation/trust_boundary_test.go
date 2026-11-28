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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/internal/bbs"
)

type testRoundTripper func(*http.Request) (*http.Response, error)

func (f testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResolveServerConfig_SlipLabelValidation(t *testing.T) {
	baseCfg := func(label string) ServerConfig {
		return ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  "https://origin.example.com",
			Peers: []PeerConfig{
				{
					ID:        "peer-a",
					Token:     "tok-a",
					SlipLabel: label,
				},
			},
		}
	}

	resolved, err := ResolveServerConfig(baseCfg("  ｶﾞｲﾌﾞ  "))
	if err != nil {
		t.Fatalf("expected valid slip_label to pass, got %v", err)
	}
	if resolved.Peers[0].SlipLabel != "ｶﾞｲﾌﾞ" {
		t.Fatalf("expected trimmed slip_label 'ｶﾞｲﾌﾞ', got %q", resolved.Peers[0].SlipLabel)
	}

	tooLong := strings.Repeat("ｱ", bbs.MaxSlipEventLabelRunes+1)
	if _, err := ResolveServerConfig(baseCfg(tooLong)); err == nil {
		t.Fatal("expected ResolveServerConfig to reject over-long slip_label")
	}

	if _, err := ResolveServerConfig(baseCfg("\x00\r\n")); err == nil {
		t.Fatal("expected ResolveServerConfig to reject control-char-only slip_label")
	}
}

func TestFederationTrustBoundary_OriginSlipAndReportPeer(t *testing.T) {
	if ProtocolVersion != 2 {
		t.Fatalf("expected ProtocolVersion=2, got %d", ProtocolVersion)
	}

	store := newInternalTestStore(t)
	reg := bbs.NewProviderRegistry(store.Key())
	reg.Register(store)
	idGen := bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!"))
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v1"})
	svc := bbs.NewService(reg, store, idGen, pm, nil)
	hub := NewEventHub(64)

	ctx := context.Background()
	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeGlobal,
		Slug:  "fed-cat",
		Name:  "Fed Cat",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeGlobal,
		Slug:       "fed-board",
		Name:       "Fed Board",
		Rules: bbs.BoardRules{
			SLIP: bbs.SLIPVVVVV,
		},
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	userID := snowflake.ID(555000111222333444)
	if _, err := svc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}

	var restCalls atomic.Int32
	globalName := "OriginAlice"
	dc := &bot.Client{
		ApplicationID: snowflake.ID(999111),
		Caches:        cache.New(),
		Rest: rest.New(rest.NewClient("Bot test-token", rest.WithHTTPClient(&http.Client{
			Transport: testRoundTripper(func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "/users/"+userID.String()) {
					restCalls.Add(1)
					raw, _ := json.Marshal(discord.User{
						ID:          userID,
						Username:    "origin_alice",
						GlobalName:  &globalName,
						PublicFlags: discord.UserFlagHouseBravery,
					})
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(bytes.NewReader(raw)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}),
		}))),
	}

	srv, err := NewServer(ServerDeps{
		Config: ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  "https://origin.example.com",
			OAuth: OAuthConfig{
				ClientID:     "cid",
				ClientSecret: "csec",
			},
			Peers: []PeerConfig{
				{
					ID:           "peer-beta",
					Token:        "peer-tok-beta",
					AllowWrite:   true,
					RedirectURIs: []string{"https://beta.example.com/cb"},
					SlipLabel:    "ﾍﾞｰﾀ",
				},
			},
		},
		Service: svc,
		Client:  dc,
		Hub:     hub,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	rawAccess := "user-access-token-beta"
	now := time.Now().UTC()
	grant := &bbs.FederationGrant{
		PeerID:          "peer-beta",
		UserID:          userID,
		Scopes:          []string{bbs.FederationScopeWrite, bbs.FederationScopeReport},
		PolicyVersion:   pm.CurrentVersion,
		AccessTokenHash: HashToken(rawAccess),
		AccessExpiresAt: now.Add(1 * time.Hour),
	}
	rt := &bbs.FederationRefreshToken{
		TokenHash: HashToken("refresh-beta"),
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := store.CreateFederationGrantWithRefresh(ctx, grant, rt); err != nil {
		t.Fatalf("CreateFederationGrantWithRefresh failed: %v", err)
	}

	// 1. CreateThread via federation -> SLIP must use peer's SlipLabel and Origin-fetched user code.
	ctBody, _ := json.Marshal(CreateThreadRequest{
		Title:   "フェデレーションスレ",
		RawBody: "テスト本文",
	})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/bbs/federation/boards/"+board.ID.String()+"/threads", bytes.NewReader(ctBody))
	req1.Header.Set("Authorization", "Bearer peer-tok-beta")
	req1.Header.Set(HeaderUserToken, rawAccess)
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("CreateThread expected 201, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var ctResp CreateThreadResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &ctResp); err != nil {
		t.Fatalf("unmarshal CreateThreadResponse failed: %v", err)
	}
	expectedCode := idGen.GenerateSlipCode(bbs.ScopeGlobal, userID, bbs.SlipContext{
		Username:   "origin_alice",
		GlobalName: "OriginAlice",
	})
	if !strings.Contains(ctResp.Post.SlipText, "ﾍﾞｰﾀ") || !strings.HasSuffix(ctResp.Post.SlipText, " "+expectedCode) {
		t.Fatalf("expected SlipText with peer label 'ﾍﾞｰﾀ' and origin-derived code %q, got %q", expectedCode, ctResp.Post.SlipText)
	}
	if restCalls.Load() != 1 {
		t.Fatalf("expected 1 Discord REST GetUser call, got %d", restCalls.Load())
	}

	// 2. Second write (CreatePost) must hit userSlipCache without another Discord REST call.
	cpBody, _ := json.Marshal(CreatePostRequest{
		RawBody: "2レス目",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/bbs/federation/threads/"+ctResp.Thread.ID.String()+"/posts", bytes.NewReader(cpBody))
	req2.Header.Set("Authorization", "Bearer peer-tok-beta")
	req2.Header.Set(HeaderUserToken, rawAccess)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("CreatePost expected 201, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if restCalls.Load() != 1 {
		t.Fatalf("expected userSlipCache hit (still 1 REST call), got %d", restCalls.Load())
	}

	// 3. CreateReport must persist authenticated peer.ID into OriginPeer.
	repBody, _ := json.Marshal(CreateReportRequest{
		GuildID: snowflake.ID(777888),
		Reason:  "スパム投稿",
	})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/bbs/federation/posts/"+ctResp.Post.ID.String()+"/reports", bytes.NewReader(repBody))
	req3.Header.Set("Authorization", "Bearer peer-tok-beta")
	req3.Header.Set(HeaderUserToken, rawAccess)
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusCreated {
		t.Fatalf("CreateReport expected 201, got %d: %s", rec3.Code, rec3.Body.String())
	}
	var repResp CreateReportResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &repResp); err != nil {
		t.Fatalf("unmarshal CreateReportResponse failed: %v", err)
	}
	if repResp.OriginPeer != "peer-beta" {
		t.Fatalf("expected response OriginPeer='peer-beta', got %q", repResp.OriginPeer)
	}
	storedRep, err := store.GetReport(ctx, repResp.ID)
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if storedRep.OriginPeer != "peer-beta" {
		t.Fatalf("expected persisted Report.OriginPeer='peer-beta', got %q", storedRep.OriginPeer)
	}
}
