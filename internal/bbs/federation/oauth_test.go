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

package federation_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/federation"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func openTestServiceDB(t *testing.T, policyVer string, caps []bbs.CapDefinition) (*gorm.DB, *gormstore.Provider, *bbs.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	if err := gormstore.DropCrossProviderForeignKeys(db); err != nil {
		t.Fatalf("DropCrossProviderForeignKeys failed: %v", err)
	}

	store := gormstore.New(db)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	svc := bbs.NewService(
		reg,
		store,
		bbs.NewIdentityGenerator([]byte("federation-test-secret-key-32b!!")),
		bbs.NewPolicyManager(bbs.PolicyConfig{
			Version:          policyVer,
			TermsURL:         "https://example.com/terms",
			GuidelinesURL:    "https://example.com/guidelines",
			PrivacyPolicyURL: "https://example.com/privacy",
		}),
		[]snowflake.ID{snowflake.ID(999999)},
	)
	if len(caps) > 0 {
		if err := svc.LoadOrSeedAdminCaps(context.Background(), caps); err != nil {
			t.Fatalf("LoadOrSeedAdminCaps failed: %v", err)
		}
	}
	return db, store, svc
}

type mockDiscordOAuth struct {
	mu           sync.Mutex
	clientID     string
	clientSecret string
	nextUser     snowflake.ID
	nextUsername string
	codes        map[string]snowflake.ID
	tokens       map[string]snowflake.ID
	srv          *httptest.Server
}

func newMockDiscordOAuth(t *testing.T, clientID, clientSecret string) *mockDiscordOAuth {
	t.Helper()
	m := &mockDiscordOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		nextUser:     snowflake.ID(10001),
		nextUsername: "testuser",
		codes:        make(map[string]snowflake.ID),
		tokens:       make(map[string]snowflake.ID),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		state := q.Get("state")
		redir := q.Get("redirect_uri")
		m.mu.Lock()
		uid := m.nextUser
		code := "disc-code-" + uuid.NewString()
		m.codes[code] = uid
		m.mu.Unlock()

		u, _ := url.Parse(redir)
		rq := u.Query()
		rq.Set("code", code)
		rq.Set("state", state)
		u.RawQuery = rq.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	})
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.PostFormValue("client_id") != m.clientID || r.PostFormValue("client_secret") != m.clientSecret {
			http.Error(w, "invalid client credentials", http.StatusUnauthorized)
			return
		}
		code := r.PostFormValue("code")
		m.mu.Lock()
		uid, ok := m.codes[code]
		delete(m.codes, code)
		tok := "disc-tok-" + uuid.NewString()
		if ok {
			m.tokens[tok] = uid
		}
		m.mu.Unlock()
		if !ok {
			http.Error(w, "invalid code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": tok})
	})
	mux.HandleFunc("/users/@me", func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		m.mu.Lock()
		uid, ok := m.tokens[auth]
		uname := m.nextUsername
		m.mu.Unlock()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          uid.String(),
			"username":    uname,
			"global_name": "Display-" + uname,
		})
	})
	m.srv = httptest.NewServer(mux)
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mockDiscordOAuth) SetUser(uid snowflake.ID, username string) {
	m.mu.Lock()
	m.nextUser = uid
	m.nextUsername = username
	m.mu.Unlock()
}

var (
	reHiddenTx   = regexp.MustCompile(`name="tx" value="([^"]+)"`)
	reHiddenCSRF = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
)

func newNoRedirectJarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar failed: %v", err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func resolveLocation(baseURL, loc string) string {
	if strings.HasPrefix(loc, "/") {
		u, _ := url.Parse(baseURL)
		return u.Scheme + "://" + u.Host + loc
	}
	return loc
}

func stepToConsentForm(t *testing.T, httpCli *http.Client, authorizeURL string) (consentURL, consentHTML, tx, csrf string) {
	t.Helper()
	resp1, err := httpCli.Get(authorizeURL)
	if err != nil {
		t.Fatalf("GET authorize failed: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from /oauth/authorize, got %d", resp1.StatusCode)
	}

	resp2, err := httpCli.Get(resp1.Header.Get("Location"))
	if err != nil {
		t.Fatalf("GET discord authorize failed: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from discord authorize, got %d", resp2.StatusCode)
	}

	resp3, err := httpCli.Get(resp2.Header.Get("Location"))
	if err != nil {
		t.Fatalf("GET origin callback failed: %v", err)
	}
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from origin callback, got %d", resp3.StatusCode)
	}
	consentURL = resolveLocation(authorizeURL, resp3.Header.Get("Location"))

	resp4, err := httpCli.Get(consentURL)
	if err != nil {
		t.Fatalf("GET consent failed: %v", err)
	}
	rawHTML, _ := io.ReadAll(resp4.Body)
	_ = resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on consent form, got %d: %s", resp4.StatusCode, string(rawHTML))
	}
	consentHTML = string(rawHTML)
	txM := reHiddenTx.FindStringSubmatch(consentHTML)
	csrfM := reHiddenCSRF.FindStringSubmatch(consentHTML)
	if len(txM) < 2 || len(csrfM) < 2 {
		t.Fatalf("missing tx or csrf_token in consent HTML")
	}
	return consentURL, consentHTML, txM[1], csrfM[1]
}

func approveConsentAndFinishCallback(
	t *testing.T,
	httpCli *http.Client,
	authorizeURL string,
	agreeCaps bool,
) (consentHTML string, cbRespCode int) {
	t.Helper()
	consentURL, htmlStr, tx, csrf := stepToConsentForm(t, httpCli, authorizeURL)
	form := url.Values{
		"tx":               {tx},
		"csrf_token":       {csrf},
		"decision":         {"approve"},
		"agree_terms":      {"on"},
		"agree_guidelines": {"on"},
		"agree_privacy":    {"on"},
		"agree_delegate":   {"on"},
	}
	if agreeCaps {
		form.Set("agree_caps", "on")
	}
	resp5, err := httpCli.PostForm(consentURL, form)
	if err != nil {
		t.Fatalf("POST consent failed: %v", err)
	}
	_ = resp5.Body.Close()
	if resp5.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 after consent approval, got %d", resp5.StatusCode)
	}
	cbURL := resp5.Header.Get("Location")
	resp6, err := httpCli.Get(cbURL)
	if err != nil {
		t.Fatalf("GET listener callback failed: %v", err)
	}
	_ = resp6.Body.Close()
	return htmlStr, resp6.StatusCode
}

type oauthTestHarness struct {
	originDB    *gorm.DB
	originStore *gormstore.Provider
	originSvc   *bbs.Service
	originHub   *federation.EventHub
	originSrv   *federation.Server
	originHTTP  *httptest.Server

	listenerStore *gormstore.Provider
	listenerSvc   *bbs.Service
	cbSrv         *federation.CallbackServer
	cbHTTP        *httptest.Server
	cipher        *federation.TokenCipher
	disc          *mockDiscordOAuth
	completedCh   chan federation.BeginAuthRequest
}

func newOAuthTestHarness(t *testing.T, peers []federation.PeerConfig) *oauthTestHarness {
	t.Helper()
	userWithCap := snowflake.ID(10001)
	capDef := bbs.CapDefinition{
		ID:    "opcap",
		Name:  "★テストキャップ",
		Users: []snowflake.ID{userWithCap},
	}

	originDB, originStore, originSvc := openTestServiceDB(t, "v1", []bbs.CapDefinition{capDef})
	_, listenerStore, listenerSvc := openTestServiceDB(t, "v1", nil)
	disc := newMockDiscordOAuth(t, "disc-client-id", "disc-client-secret")

	var originHandler, cbHandler http.Handler
	originHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(originHTTP.Close)
	cbHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cbHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(cbHTTP.Close)

	cipher, err := federation.NewTokenCipher([]byte("cipher-secret-32-bytes-minimum!!"))
	if err != nil {
		t.Fatalf("NewTokenCipher failed: %v", err)
	}
	completedCh := make(chan federation.BeginAuthRequest, 16)
	cbSrv, err := federation.NewCallbackServer(federation.CallbackServerDeps{
		Config: federation.ClientConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  cbHTTP.URL,
		},
		Store:  listenerStore,
		Cipher: cipher,
		OnAuthComplete: func(_ context.Context, req federation.BeginAuthRequest, _ string) {
			completedCh <- req
		},
	})
	if err != nil {
		t.Fatalf("NewCallbackServer failed: %v", err)
	}
	cbHandler = cbSrv.Handler()

	// Populate CallbackURL in peer redirect URIs if empty.
	resolvedPeers := make([]federation.PeerConfig, len(peers))
	for i, p := range peers {
		if len(p.RedirectURIs) == 0 && (p.AllowWrite || p.EffectiveAllowReport()) {
			p.RedirectURIs = []string{cbSrv.CallbackURL()}
		}
		resolvedPeers[i] = p
	}

	hub := federation.NewEventHub(64)
	originSvc.SetEventSink(hub)
	originSrv, err := federation.NewServer(federation.ServerDeps{
		Config: federation.ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  originHTTP.URL,
			OAuth:      federation.OAuthConfig{ClientID: "disc-client-id", ClientSecret: "disc-client-secret"},
			Peers:      resolvedPeers,
		},
		Service:            originSvc,
		Hub:                hub,
		OAuthAPIBase:       disc.srv.URL,
		OAuthAuthorizeBase: disc.srv.URL + "/oauth2/authorize",
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	originHandler = originSrv.Handler()

	return &oauthTestHarness{
		originDB:      originDB,
		originStore:   originStore,
		originSvc:     originSvc,
		originHub:     hub,
		originSrv:     originSrv,
		originHTTP:    originHTTP,
		listenerStore: listenerStore,
		listenerSvc:   listenerSvc,
		cbSrv:         cbSrv,
		cbHTTP:        cbHTTP,
		cipher:        cipher,
		disc:          disc,
		completedCh:   completedCh,
	}
}

func (h *oauthTestHarness) addRemote(t *testing.T, key, peerID, token string) *federation.RemoteProvider {
	t.Helper()
	rp, err := federation.NewRemoteProvider(federation.RemoteProviderDeps{
		Config: federation.RemoteConfig{
			Key:     key,
			Label:   key,
			BaseURL: h.originHTTP.URL,
			PeerID:  peerID,
			Token:   token,
		},
		Store:           h.listenerStore,
		Cipher:          h.cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider %q failed: %v", key, err)
	}
	h.cbSrv.RegisterRemote(rp)
	h.listenerSvc.Registry.Register(rp)
	return rp
}

func TestOAuth_FullE2E_PKCE_And_CapOptIn(t *testing.T) {
	ctx := context.Background()
	h := newOAuthTestHarness(t, []federation.PeerConfig{
		{ID: "peer-caps", Name: "Dev Caps", Token: "tok-caps", AllowWrite: true, AllowCaps: true},
		{ID: "peer-nocaps", Name: "Dev NoCaps", Token: "tok-nocaps", AllowWrite: true, AllowCaps: false},
	})
	rpCaps := h.addRemote(t, "prodcaps", "peer-caps", "tok-caps")
	rpNoCaps := h.addRemote(t, "prodnocaps", "peer-nocaps", "tok-nocaps")

	cat, _ := h.originSvc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeGlobal, Slug: "gcat", Name: "Global Cat"})
	board, err := h.originSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeGlobal,
		Slug:       "gboard",
		Name:       "Global Board",
		Rules:      bbs.BoardRules{MaxRes: 100}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	userWithCap := snowflake.ID(10001)
	h.disc.SetUser(userWithCap, "capuser")

	// 1. Peer with AllowCaps=true + user checks agree_caps=on -> full caps delegation.
	authURL, _, err := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{
		ProviderKey:      rpCaps.Key(),
		UserID:           userWithCap,
		ApplicationID:    snowflake.ID(555),
		InteractionToken: "tok-inter-1",
		ContinueCustomID: "/bbs/thread/new/cont",
		Locale:           discord.LocaleJapanese,
	})
	if err != nil {
		t.Fatalf("BeginUserAuth failed: %v", err)
	}
	httpCli := newNoRedirectJarClient(t)
	htmlStr, cbStatus := approveConsentAndFinishCallback(t, httpCli, authURL, true)
	if cbStatus != http.StatusOK {
		t.Fatalf("expected 200 on listener callback, got %d", cbStatus)
	}
	if !strings.Contains(htmlStr, `name="agree_caps"`) {
		t.Fatalf("expected optional agree_caps checkbox for cap holder")
	}

	select {
	case completed := <-h.completedCh:
		if completed.UserID != userWithCap || completed.ContinueCustomID != "/bbs/thread/new/cont" {
			t.Fatalf("unexpected OnAuthComplete payload: %+v", completed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected OnAuthComplete callback to fire")
	}

	hasDel, err := rpCaps.HasUserDelegation(ctx, userWithCap)
	if err != nil || !hasDel {
		t.Fatalf("expected HasUserDelegation=true, got %v, err=%v", hasDel, err)
	}
	caps, err := rpCaps.ListRemoteAvailableCaps(ctx, userWithCap, board.ID)
	if err != nil || len(caps) != 1 || caps[0].ID != "opcap" {
		t.Fatalf("expected remote opcap available, got %+v, err=%v", caps, err)
	}
	_, _, post, err := rpCaps.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: snowflake.ID(7001),
		UserID:  userWithCap,
		Title:   "キャップスレ",
		CapID:   "opcap",
		RawBody: "キャップ本文",
	})
	if err != nil || post.Cap == "" {
		t.Fatalf("expected cap thread creation to succeed, got post=%+v err=%v", post, err)
	}

	// 2. Same user + peer-caps, but user UNCHECKS agree_caps -> caps NOT granted and CapID rejected.
	authURL2, _, err := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{
		ProviderKey: rpCaps.Key(),
		UserID:      userWithCap,
	})
	if err != nil {
		t.Fatalf("BeginUserAuth 2 failed: %v", err)
	}
	_, cbStatus2 := approveConsentAndFinishCallback(t, httpCli, authURL2, false)
	if cbStatus2 != http.StatusOK {
		t.Fatalf("expected 200 on callback 2, got %d", cbStatus2)
	}
	capsUnchecked, err := rpCaps.ListRemoteAvailableCaps(ctx, userWithCap, board.ID)
	if err != nil || len(capsUnchecked) != 0 {
		t.Fatalf("expected no caps when user unchecked agree_caps, got %+v, err=%v", capsUnchecked, err)
	}
	if _, _, _, err := rpCaps.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: snowflake.ID(7001),
		UserID:  userWithCap,
		Title:   "キャップ無許可",
		CapID:   "opcap",
		RawBody: "本文",
	}); !errors.Is(err, bbs.ErrCapNotAuthorized) {
		t.Fatalf("expected ErrCapNotAuthorized when user unchecked caps, got %v", err)
	}

	// 3. Peer with AllowCaps=false -> consent screen hides agree_caps and caps are rejected even if tampered.
	authURL3, _, err := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{
		ProviderKey: rpNoCaps.Key(),
		UserID:      userWithCap,
	})
	if err != nil {
		t.Fatalf("BeginUserAuth 3 failed: %v", err)
	}
	htmlNoCaps, cbStatus3 := approveConsentAndFinishCallback(t, httpCli, authURL3, true)
	if cbStatus3 != http.StatusOK {
		t.Fatalf("expected 200 on callback 3, got %d", cbStatus3)
	}
	if strings.Contains(htmlNoCaps, `name="agree_caps"`) {
		t.Fatalf("expected agree_caps hidden when peer.AllowCaps=false")
	}
	if _, _, _, err := rpNoCaps.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: snowflake.ID(7001),
		UserID:  userWithCap,
		Title:   "ピアキャップ無許可",
		CapID:   "opcap",
		RawBody: "本文",
	}); !errors.Is(err, bbs.ErrCapNotAuthorized) {
		t.Fatalf("expected ErrCapNotAuthorized when peer.AllowCaps=false, got %v", err)
	}
}

func TestOAuth_SecurityAndValidationFailures(t *testing.T) {
	ctx := context.Background()
	h := newOAuthTestHarness(t, []federation.PeerConfig{
		{ID: "peer-dev", Name: "Dev Peer", Token: "tok-dev", AllowWrite: true, AllowCaps: false},
	})
	rp := h.addRemote(t, "prod", "peer-dev", "tok-dev")

	catG, _ := h.originSvc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeGlobal, Slug: "gcat", Name: "Global"})
	boardA, _ := h.originSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: catG.ID, Scope: bbs.ScopeGlobal, Slug: "ga", Name: "Global A",
		Rules: bbs.BoardRules{MaxRes: 100}.Normalize(),
	})
	boardB, _ := h.originSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: catG.ID, Scope: bbs.ScopeGlobal, Slug: "gb", Name: "Global B",
		Rules: bbs.BoardRules{MaxRes: 100}.Normalize(),
	})
	guildLocal := snowflake.ID(8080)
	if _, err := h.originSvc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildLocal, Enabled: true, ActorUserID: 1}); err != nil {
		t.Fatalf("SetupGuild guildLocal failed: %v", err)
	}
	catL, err := h.originSvc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeLocal, GuildID: &guildLocal, Slug: "lcat", Name: "Local"})
	if err != nil {
		t.Fatalf("CreateCategory local failed: %v", err)
	}
	boardLocal, err := h.originSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: catL.ID, Scope: bbs.ScopeLocal, GuildID: &guildLocal, Slug: "la", Name: "Local A",
		Rules: bbs.BoardRules{MaxRes: 100}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard local failed: %v", err)
	}

	userA := snowflake.ID(20001)
	httpCli := newNoRedirectJarClient(t)

	// 1. Unknown client_id & invalid redirect_uri & bad PKCE on /oauth/authorize.
	rUnknown, _ := httpCli.Get(h.originHTTP.URL + "/api/v1/bbs/federation/oauth/authorize?client_id=nope&redirect_uri=" + url.QueryEscape(h.cbSrv.CallbackURL()))
	_ = rUnknown.Body.Close()
	if rUnknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on unknown client_id, got %d", rUnknown.StatusCode)
	}
	rBadRedir, _ := httpCli.Get(h.originHTTP.URL + "/api/v1/bbs/federation/oauth/authorize?client_id=peer-dev&redirect_uri=" + url.QueryEscape("https://evil.example.com/cb"))
	_ = rBadRedir.Body.Close()
	if rBadRedir.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on unlisted redirect_uri, got %d", rBadRedir.StatusCode)
	}
	rPlainPKCE, _ := httpCli.Get(h.originHTTP.URL + "/api/v1/bbs/federation/oauth/authorize?client_id=peer-dev&redirect_uri=" + url.QueryEscape(h.cbSrv.CallbackURL()) + "&response_type=code&code_challenge_method=plain&code_challenge=abc&state=s1")
	_ = rPlainPKCE.Body.Close()
	if rPlainPKCE.StatusCode != http.StatusFound || !strings.Contains(rPlainPKCE.Header.Get("Location"), "error=invalid_request") {
		t.Fatalf("expected redirect error=invalid_request for non-S256 PKCE, got status=%d loc=%s", rPlainPKCE.StatusCode, rPlainPKCE.Header.Get("Location"))
	}

	// 2. Code verifier mismatch, code replay, and state replay.
	verifier := "test-verifier-long-enough-for-pkce-s256-0123456789"
	challenge := federation.ComputePKCES256(verifier)
	manualAuthURL := h.originHTTP.URL + "/api/v1/bbs/federation/oauth/authorize?client_id=peer-dev&redirect_uri=" +
		url.QueryEscape(h.cbSrv.CallbackURL()) + "&response_type=code&code_challenge_method=S256&code_challenge=" +
		url.QueryEscape(challenge) + "&state=manual-state-1"
	h.disc.SetUser(userA, "usera")
	consentURL, _, tx, csrf := stepToConsentForm(t, httpCli, manualAuthURL)

	// Missing checkboxes -> stays 200 on consent page with error.
	rMissingCheck, _ := httpCli.PostForm(consentURL, url.Values{
		"tx": {tx}, "csrf_token": {csrf}, "decision": {"approve"},
		"agree_terms": {"on"}, "agree_guidelines": {"on"}, "agree_privacy": {"on"},
		// missing agree_delegate
	})
	_ = rMissingCheck.Body.Close()
	if rMissingCheck.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 form re-render on missing required checkbox, got %d", rMissingCheck.StatusCode)
	}

	// Bad CSRF -> 403 Forbidden.
	rBadCSRF, _ := httpCli.PostForm(consentURL, url.Values{
		"tx": {tx}, "csrf_token": {"wrong-csrf"}, "decision": {"approve"},
		"agree_terms": {"on"}, "agree_guidelines": {"on"}, "agree_privacy": {"on"}, "agree_delegate": {"on"},
	})
	_ = rBadCSRF.Body.Close()
	if rBadCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on bad csrf_token, got %d", rBadCSRF.StatusCode)
	}

	// Approve cleanly to obtain authorization code.
	rApp, _ := httpCli.PostForm(consentURL, url.Values{
		"tx": {tx}, "csrf_token": {csrf}, "decision": {"approve"},
		"agree_terms": {"on"}, "agree_guidelines": {"on"}, "agree_privacy": {"on"}, "agree_delegate": {"on"},
	})
	_ = rApp.Body.Close()
	cbParsed, _ := url.Parse(rApp.Header.Get("Location"))
	rawCode := cbParsed.Query().Get("code")
	if rawCode == "" {
		t.Fatal("expected code in redirect location")
	}

	// Wrong code_verifier fails AND consumes single-use code (replay also fails).
	if _, err := rp.Client().ExchangeAuthCode(ctx, rawCode, h.cbSrv.CallbackURL(), "wrong-verifier"); err == nil {
		t.Fatal("expected code_verifier mismatch to fail")
	}
	if _, err := rp.Client().ExchangeAuthCode(ctx, rawCode, h.cbSrv.CallbackURL(), verifier); err == nil {
		t.Fatal("expected replaying consumed authorization code to fail")
	}

	// 3. Decision=deny redirects access_denied through Listener callback & state replay fails.
	authDenyURL, _, err := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userA})
	if err != nil {
		t.Fatalf("BeginUserAuth deny failed: %v", err)
	}
	cURL2, _, tx2, csrf2 := stepToConsentForm(t, httpCli, authDenyURL)
	rDeny, _ := httpCli.PostForm(cURL2, url.Values{"tx": {tx2}, "csrf_token": {csrf2}, "decision": {"deny"}})
	_ = rDeny.Body.Close()
	denyLoc := rDeny.Header.Get("Location")
	if rDeny.StatusCode != http.StatusFound || !strings.Contains(denyLoc, "error=access_denied") {
		t.Fatalf("expected access_denied redirect, got %d %s", rDeny.StatusCode, denyLoc)
	}
	rDenyCB, _ := httpCli.Get(denyLoc)
	_ = rDenyCB.Body.Close()
	if rDenyCB.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 denied page from listener callback, got %d", rDenyCB.StatusCode)
	}
	rStateReplay, _ := httpCli.Get(denyLoc)
	_ = rStateReplay.Body.Close()
	if rStateReplay.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on state replay, got %d", rStateReplay.StatusCode)
	}

	// 4. login_hint mismatch -> 403 on Origin Discord callback.
	authHintURL, _, _ := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userA})
	h.disc.SetUser(snowflake.ID(29999), "wronguser")
	rA1, _ := httpCli.Get(authHintURL)
	_ = rA1.Body.Close()
	rA2, _ := httpCli.Get(rA1.Header.Get("Location"))
	_ = rA2.Body.Close()
	rHintCB, _ := httpCli.Get(rA2.Header.Get("Location"))
	_ = rHintCB.Body.Close()
	if rHintCB.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on login_hint mismatch, got %d", rHintCB.StatusCode)
	}

	// 5. Authorize userA cleanly, test refresh reuse family revocation.
	h.disc.SetUser(userA, "usera")
	authOK, _, _ := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userA})
	if _, st := approveConsentAndFinishCallback(t, httpCli, authOK, false); st != http.StatusOK {
		t.Fatalf("expected 200 authorize, got %d", st)
	}
	tokSaved, err := h.listenerStore.GetFederationUserToken(ctx, rp.RemoteKey(), userA)
	if err != nil {
		t.Fatalf("GetFederationUserToken failed: %v", err)
	}
	oldRefresh, err := h.cipher.Decrypt(tokSaved.RefreshTokenEnc)
	if err != nil {
		t.Fatalf("Decrypt refresh failed: %v", err)
	}
	refResp, err := rp.Client().RefreshToken(ctx, oldRefresh)
	if err != nil {
		t.Fatalf("first RefreshToken failed: %v", err)
	}
	// Replay oldRefresh -> triggers family revocation.
	if _, rErr := rp.Client().RefreshToken(ctx, oldRefresh); rErr == nil {
		t.Fatal("expected refresh reuse to fail")
	}
	// Newly rotated refresh token from same family is ALSO revoked!
	if _, rErr2 := rp.Client().RefreshToken(ctx, refResp.RefreshToken); rErr2 == nil {
		t.Fatal("expected new refresh token in compromised family to also be revoked")
	}
	// Stored access token is also now rejected on write -> ErrFederationAuthRequired.
	if _, _, _, wErr := rp.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: boardA.ID, GuildID: 1, UserID: userA, Title: "Revoked", RawBody: "Body",
	}); !errors.Is(wErr, bbs.ErrFederationAuthRequired) {
		t.Fatalf("expected ErrFederationAuthRequired after reuse revocation, got %v", wErr)
	}

	// 6. Re-authorize -> PolicyVersion bump on Origin forces re-consent.
	authPolicyURL, _, _ := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userA})
	if _, st := approveConsentAndFinishCallback(t, httpCli, authPolicyURL, false); st != http.StatusOK {
		t.Fatalf("re-auth failed: %d", st)
	}
	// Swap Origin service/server to PolicyVersion "v2" sharing the same originDB.
	originRegV2 := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	originRegV2.Register(h.originStore)
	originSvcV2 := bbs.NewService(
		originRegV2,
		h.originStore,
		bbs.NewIdentityGenerator([]byte("federation-test-secret-key-32b!!")),
		bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v2"}),
		[]snowflake.ID{999999},
	)
	originSrvV2, err := federation.NewServer(federation.ServerDeps{
		Config: federation.ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  h.originHTTP.URL,
			OAuth:      federation.OAuthConfig{ClientID: "disc-client-id", ClientSecret: "disc-client-secret"},
			Peers:      []federation.PeerConfig{{ID: "peer-dev", Name: "Dev Peer", Token: "tok-dev", AllowWrite: true, RedirectURIs: []string{h.cbSrv.CallbackURL()}}},
		},
		Service:            originSvcV2,
		Hub:                federation.NewEventHub(64),
		OAuthAPIBase:       h.disc.srv.URL,
		OAuthAuthorizeBase: h.disc.srv.URL + "/oauth2/authorize",
	})
	if err != nil {
		t.Fatalf("NewServer v2 failed: %v", err)
	}
	// Create a new remote pointing to originSrvV2 while sharing Listener store & Origin DB.
	srvV2 := httptest.NewServer(originSrvV2.Handler())
	defer srvV2.Close()
	rpV2, _ := federation.NewRemoteProvider(federation.RemoteProviderDeps{
		Config:          federation.RemoteConfig{Key: "prod", Label: "prod", BaseURL: srvV2.URL, PeerID: "peer-dev", Token: "tok-dev"},
		Store:           h.listenerStore,
		Cipher:          h.cipher,
		CallbackEnabled: true,
	})
	hasDelAfterBump, err := rpV2.HasUserDelegation(ctx, userA)
	if err != nil || hasDelAfterBump {
		t.Fatalf("expected HasUserDelegation=false after PolicyVersion bump, got %v err=%v", hasDelAfterBump, err)
	}
	if _, err := h.listenerStore.GetFederationUserToken(ctx, "prod", userA); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected stale listener token deleted on PolicyVersion mismatch, got %v", err)
	}

	// 7. Re-authorize on v2 -> Origin Service.RevokePolicy cascades revocation to all FederationGrants.
	var cbV2Handler http.Handler
	cbV2HTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cbV2Handler.ServeHTTP(w, r)
	}))
	defer cbV2HTTP.Close()
	cbSrvV2, err := federation.NewCallbackServer(federation.CallbackServerDeps{
		Config: federation.ClientConfig{Enabled: true, ListenAddr: "127.0.0.1:0", PublicURL: cbV2HTTP.URL},
		Store:  h.listenerStore,
		Cipher: h.cipher,
	})
	if err != nil {
		t.Fatalf("NewCallbackServer v2 failed: %v", err)
	}
	cbV2Handler = cbSrvV2.Handler()

	var originV2Handler http.Handler
	originV2HTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originV2Handler.ServeHTTP(w, r)
	}))
	defer originV2HTTP.Close()

	originSrvV2Reauth, err := federation.NewServer(federation.ServerDeps{
		Config: federation.ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  originV2HTTP.URL,
			OAuth:      federation.OAuthConfig{ClientID: "disc-client-id", ClientSecret: "disc-client-secret"},
			Peers: []federation.PeerConfig{
				{
					ID:              "peer-scoped",
					Name:            "Scoped Peer",
					Token:           "tok-scoped",
					AllowWrite:      true,
					RedirectURIs:    []string{cbSrvV2.CallbackURL()},
					AllowedBoardIDs: []string{boardA.ID.String()},
				},
			},
		},
		Service:            originSvcV2,
		Hub:                h.originHub,
		OAuthAPIBase:       h.disc.srv.URL,
		OAuthAuthorizeBase: h.disc.srv.URL + "/oauth2/authorize",
	})
	if err != nil {
		t.Fatalf("NewServer v2 reauth failed: %v", err)
	}
	originV2Handler = originSrvV2Reauth.Handler()

	rpV2Reauth, err := federation.NewRemoteProvider(federation.RemoteProviderDeps{
		Config:          federation.RemoteConfig{Key: "prodv2", Label: "prodv2", BaseURL: originV2HTTP.URL, PeerID: "peer-scoped", Token: "tok-scoped"},
		Store:           h.listenerStore,
		Cipher:          h.cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider v2 failed: %v", err)
	}
	cbSrvV2.RegisterRemote(rpV2Reauth)

	authV2URL, _, err := cbSrvV2.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rpV2Reauth.Key(), UserID: userA})
	if err != nil {
		t.Fatalf("BeginUserAuth v2 failed: %v", err)
	}
	if _, st := approveConsentAndFinishCallback(t, httpCli, authV2URL, false); st != http.StatusOK {
		t.Fatalf("re-auth v2 failed: %d", st)
	}

	// Origin RevokePolicy cascades grant revocation immediately.
	if err := originSvcV2.RevokePolicy(ctx, userA); err != nil {
		t.Fatalf("origin RevokePolicy failed: %v", err)
	}
	if _, _, _, err := rpV2Reauth.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: boardA.ID, GuildID: 1, UserID: userA, Title: "AfterRevoke", RawBody: "Body",
	}); !errors.Is(err, bbs.ErrFederationAuthRequired) {
		t.Fatalf("expected ErrFederationAuthRequired after Origin RevokePolicy, got %v", err)
	}

	// 8. Global Ban enforcement: banned user rejected at /oauth/callback AND on remote write.
	userBanned := snowflake.ID(30003)
	h.disc.SetUser(userBanned, "banneduser")
	_, _ = h.originSvc.AcceptPolicy(ctx, userBanned)
	authBanURL, _, err := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userBanned})
	if err != nil {
		t.Fatalf("BeginUserAuth for ban test failed: %v", err)
	}
	if _, st := approveConsentAndFinishCallback(t, httpCli, authBanURL, false); st != http.StatusOK {
		t.Fatalf("initial auth before ban failed: %d", st)
	}
	uidBanPtr := userBanned
	if err := h.originStore.CreateBan(ctx, &bbs.Ban{
		ID:           uuid.New(),
		Scope:        bbs.ScopeGlobal,
		TargetUserID: &uidBanPtr,
		Reason:       "test ban",
		CreatedBy:    999999,
	}); err != nil {
		t.Fatalf("CreateBan failed: %v", err)
	}
	if _, _, _, err := rp.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: boardA.ID, GuildID: 1, UserID: userBanned, Title: "BannedWrite", RawBody: "Body",
	}); !errors.Is(err, bbs.ErrBanned) {
		t.Fatalf("expected ErrBanned on remote write after global ban, got %v", err)
	}
	// New OAuth attempt while globally banned is rejected at Discord callback.
	authWhileBannedURL, _, _ := h.cbSrv.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rp.Key(), UserID: userBanned})
	rb1, _ := httpCli.Get(authWhileBannedURL)
	_ = rb1.Body.Close()
	rb2, _ := httpCli.Get(rb1.Header.Get("Location"))
	_ = rb2.Body.Close()
	rb3, _ := httpCli.Get(rb2.Header.Get("Location"))
	_ = rb3.Body.Close()
	if rb3.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden at /oauth/callback for banned user, got %d", rb3.StatusCode)
	}

	// 9. DTO Privacy (no author_user_id) & AllowedBoardIDs & ScopeLocal=404 & SSE board filtering.
	// Re-authorize userA on scoped peer to create a real thread+post on boardA & boardB on Origin.
	h.disc.SetUser(userA, "usera")
	authScoped, _, _ := cbSrvV2.BeginUserAuth(ctx, federation.BeginAuthRequest{ProviderKey: rpV2Reauth.Key(), UserID: userA})
	if _, st := approveConsentAndFinishCallback(t, httpCli, authScoped, false); st != http.StatusOK {
		t.Fatalf("scoped auth failed: %d", st)
	}
	_, thA, pA, err := rpV2Reauth.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: boardA.ID, GuildID: 1, UserID: userA, Title: "Scoped Thread A", RawBody: "Body A",
	})
	if err != nil {
		t.Fatalf("CreateRemoteThread on allowed boardA failed: %v", err)
	}
	if _, _, _, err := rpV2Reauth.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: boardB.ID, GuildID: 1, UserID: userA, Title: "Disallowed B", RawBody: "Body B",
	}); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound writing to boardB outside allowed_board_ids, got %v", err)
	}

	// Check raw HTTP DTO endpoints directly for scoped peer.
	checkPeerGet := func(path string) (int, string) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, originV2HTTP.URL+path, nil)
		req.Header.Set("Authorization", "Bearer tok-scoped")
		resp, gErr := http.DefaultClient.Do(req)
		if gErr != nil {
			t.Fatalf("GET %s failed: %v", path, gErr)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode, string(b)
	}
	if st, body := checkPeerGet("/api/v1/bbs/federation/boards"); st != http.StatusOK || !strings.Contains(body, boardA.ID.String()) || strings.Contains(body, boardB.ID.String()) || strings.Contains(body, boardLocal.ID.String()) {
		t.Fatalf("unexpected /boards response: status=%d body=%s", st, body)
	}
	if st, _ := checkPeerGet("/api/v1/bbs/federation/boards/" + boardLocal.ID.String()); st != http.StatusNotFound {
		t.Fatalf("expected 404 for ScopeLocal board, got %d", st)
	}
	if st, _ := checkPeerGet("/api/v1/bbs/federation/boards/" + boardB.ID.String()); st != http.StatusNotFound {
		t.Fatalf("expected 404 for boardB outside allowed_board_ids, got %d", st)
	}
	if st, body := checkPeerGet("/api/v1/bbs/federation/threads/" + thA.ID.String() + "/posts"); st != http.StatusOK || !strings.Contains(body, pA.ID.String()) || strings.Contains(body, "author_user_id") || strings.Contains(body, userA.String()) {
		t.Fatalf("unexpected /posts response (must omit author_user_id and raw userID): status=%d body=%s", st, body)
	}

	// Open SSE stream for tok-scoped starting from hub.LatestSeq(), emit native post on boardB & boardA.
	sseCtx, sseCancel := context.WithCancel(ctx)
	defer sseCancel()
	sseResp, err := rpV2Reauth.Client().OpenEventsStream(sseCtx, h.originHub.LatestSeq(), h.originHub.Epoch())
	if err != nil {
		t.Fatalf("OpenEventsStream failed: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()

	_, _ = h.disc, thA
	writerB := snowflake.ID(40004)
	if _, err := h.originSvc.AcceptPolicy(ctx, writerB); err != nil {
		t.Fatalf("AcceptPolicy writerB failed: %v", err)
	}
	if _, _, _, bErr := h.originSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: boardB.ID, GuildID: guildLocal, UserID: writerB, Title: "Origin Private Global B", RawBody: "B",
	}); bErr != nil {
		t.Fatalf("origin CreateThread boardB failed: %v", bErr)
	}
	_, _, _, _, err = h.originSvc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: thA.ID, GuildID: guildLocal, UserID: writerB, RawBody: "Allowed BoardA Reply",
	})
	if err != nil {
		t.Fatalf("origin CreatePost boardA failed: %v", err)
	}

	scanner := bufio.NewScanner(sseResp.Body)
	deadline := time.After(3 * time.Second)
	var sseLines []string
	lineCh := make(chan string, 32)
	go func() {
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		close(lineCh)
	}()
	sawBoardA := false
	for !sawBoardA {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for allowed boardA SSE event; got lines=%v", sseLines)
		case line, ok := <-lineCh:
			if !ok {
				t.Fatalf("SSE closed early; lines=%v", sseLines)
			}
			sseLines = append(sseLines, line)
			if strings.Contains(line, "author_user_id") || strings.Contains(line, writerB.String()) {
				t.Fatalf("SSE leaked author_user_id or raw userID: %s", line)
			}
			if strings.Contains(line, boardB.ID.String()) {
				t.Fatalf("SSE leaked disallowed boardB event: %s", line)
			}
			if strings.Contains(line, "Allowed BoardA Reply") {
				sawBoardA = true
			}
		}
	}
}
