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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func newInternalTestStore(t *testing.T) *gormstore.Provider {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	return gormstore.New(db)
}

func TestOAuthAuthorize_RateLimited(t *testing.T) {
	store := newInternalTestStore(t)
	reg := bbs.NewProviderRegistry(store.Key())
	reg.Register(store)
	svc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)
	hub := NewEventHub(64)

	srv, err := NewServer(ServerDeps{
		Config: ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  "https://origin.example.com",
			OAuth: OAuthConfig{
				ClientID:     "discord-client",
				ClientSecret: "discord-secret",
			},
			Peers: []PeerConfig{
				{
					ID:           "peer-a",
					Token:        "peer-tok-a",
					AllowWrite:   true,
					RedirectURIs: []string{"https://listener.example.com/api/v1/bbs/federation/client/callback"},
				},
			},
		},
		Service: svc,
		Hub:     hub,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	srv.oauth.authLimiter = newOAuthRateLimiter(2, 0.01)

	reqURL := "/api/v1/bbs/federation/oauth/authorize?client_id=peer-a&redirect_uri=" +
		url.QueryEscape("https://listener.example.com/api/v1/bbs/federation/client/callback") +
		"&response_type=code&scope=bbs.write&state=st&code_challenge=" +
		ComputePKCES256("test-verifier-123456789012345678901234567890123") +
		"&code_challenge_method=S256"

	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		req.RemoteAddr = "198.51.100.10:12345"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Fatalf("attempt %d expected 302, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.RemoteAddr = "198.51.100.10:12345"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}

	// Another client IP must still be allowed.
	reqOther := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqOther.RemoteAddr = "198.51.100.11:54321"
	recOther := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recOther, reqOther)
	if recOther.Code != http.StatusFound {
		t.Fatalf("expected other IP to succeed with 302, got %d", recOther.Code)
	}
}

func TestCallbackServer_BeginUserAuth_DoesNotEvictActivePending(t *testing.T) {
	store := newInternalTestStore(t)
	cipher, err := NewTokenCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("NewTokenCipher failed: %v", err)
	}
	cbSrv, err := NewCallbackServer(CallbackServerDeps{
		Config: ClientConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  "https://listener.example.com",
		},
		Store:  store,
		Cipher: cipher,
	})
	if err != nil {
		t.Fatalf("NewCallbackServer failed: %v", err)
	}

	rp, err := NewRemoteProvider(RemoteProviderDeps{
		Config: RemoteConfig{
			Key:     "prod",
			Label:   "Prod",
			BaseURL: "https://origin.example.com",
			Token:   "tok",
			PeerID:  "peer-a",
		},
		Store:           store,
		Cipher:          cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider failed: %v", err)
	}
	rp.UpdateCapabilities(CapabilitiesResponse{
		ProtocolVersion: ProtocolVersion,
		PeerID:          "peer-a",
		AllowWrite:      true,
		AllowReport:     true,
		PolicyVersion:   "2026-01-01",
	})
	cbSrv.RegisterRemote(rp)

	// Fill pending map with active entries up to maxPendingClientAuths.
	future := time.Now().UTC().Add(5 * time.Minute)
	cbSrv.pendingMu.Lock()
	for i := range maxPendingClientAuths {
		key := fmt.Sprintf("active-state-%d", i)
		cbSrv.pending[key] = &pendingClientAuth{
			State:     key,
			ExpiresAt: future,
		}
	}
	cbSrv.pendingMu.Unlock()

	_, _, err = cbSrv.BeginUserAuth(context.Background(), BeginAuthRequest{
		ProviderKey: rp.Key(),
		UserID:      snowflake.ID(99001),
	})
	if !errors.Is(err, bbs.ErrFederationUnavailable) {
		t.Fatalf("expected ErrFederationUnavailable when pending map is full, got %v", err)
	}

	cbSrv.pendingMu.Lock()
	remaining := len(cbSrv.pending)
	cbSrv.pendingMu.Unlock()
	if remaining != maxPendingClientAuths {
		t.Fatalf("expected all %d active pending entries to remain intact, got %d", maxPendingClientAuths, remaining)
	}

	// Expire one entry and verify BeginUserAuth prunes only expired entries and succeeds.
	cbSrv.pendingMu.Lock()
	cbSrv.pending["active-state-0"].ExpiresAt = time.Now().UTC().Add(-1 * time.Second)
	cbSrv.pendingMu.Unlock()

	authURL, _, err := cbSrv.BeginUserAuth(context.Background(), BeginAuthRequest{
		ProviderKey: rp.Key(),
		UserID:      snowflake.ID(99002),
	})
	if err != nil || authURL == "" {
		t.Fatalf("expected BeginUserAuth to succeed after pruning expired entry, got url=%q err=%v", authURL, err)
	}
}

func TestRemoteProvider_ListRemoteAvailableCaps_ScopeCheckedUnderLock(t *testing.T) {
	store := newInternalTestStore(t)
	cipher, err := NewTokenCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("NewTokenCipher failed: %v", err)
	}

	refreshCalled := false
	fakeOrigin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/bbs/federation/oauth/token" {
			refreshCalled = true
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Code: CodeInvalidGrant, Error: "should not be called"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fakeOrigin.Close()

	rp, err := NewRemoteProvider(RemoteProviderDeps{
		Config: RemoteConfig{
			Key:     "prod",
			Label:   "Prod",
			BaseURL: fakeOrigin.URL,
			Token:   "tok",
			PeerID:  "peer-a",
		},
		Store:           store,
		Cipher:          cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider failed: %v", err)
	}
	rp.UpdateCapabilities(CapabilitiesResponse{
		ProtocolVersion: ProtocolVersion,
		PeerID:          "peer-a",
		AllowWrite:      true,
		AllowReport:     true,
		AllowCaps:       true,
		PolicyVersion:   "2026-01-01",
	})

	ctx := context.Background()
	userID := snowflake.ID(77001)
	accEnc, err := cipher.Encrypt("stale-access")
	if err != nil {
		t.Fatalf("encrypt access failed: %v", err)
	}
	refEnc, err := cipher.Encrypt("refresh-tok")
	if err != nil {
		t.Fatalf("encrypt refresh failed: %v", err)
	}
	if err := store.UpsertFederationUserToken(ctx, &bbs.FederationUserToken{
		RemoteKey:        rp.RemoteKey(),
		UserID:           userID,
		AccessTokenEnc:   accEnc,
		AccessExpiresAt:  time.Now().UTC().Add(-1 * time.Minute),
		RefreshTokenEnc:  refEnc,
		RefreshExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		Scopes:           []string{bbs.FederationScopeWrite},
		PolicyVersion:    "2026-01-01",
	}); err != nil {
		t.Fatalf("UpsertFederationUserToken failed: %v", err)
	}

	caps, err := rp.ListRemoteAvailableCaps(ctx, userID, uuid.New())
	if err != nil || len(caps) != 0 {
		t.Fatalf("expected empty caps without error, got caps=%v err=%v", caps, err)
	}
	if refreshCalled {
		t.Fatal("expected token refresh NOT to be called when delegation lacks bbs.caps scope")
	}
	if _, err := store.GetFederationUserToken(ctx, rp.RemoteKey(), userID); err != nil {
		t.Fatalf("expected stored token to remain intact, got err=%v", err)
	}
}

func TestOAuthRateLimiter_DoesNotRewindLastSeen(t *testing.T) {
	limiter := newOAuthRateLimiter(2, 1.0)
	t0 := time.Date(2026, 11, 28, 12, 0, 10, 0, time.UTC)

	if ok, _ := limiter.Allow("client-1", t0); !ok {
		t.Fatal("first request at t0 should be allowed")
	}
	if ok, _ := limiter.Allow("client-1", t0); !ok {
		t.Fatal("second request at t0 should be allowed")
	}
	// Out-of-order earlier timestamp must NOT rewind lastSeen.
	earlier := t0.Add(-5 * time.Second)
	if ok, _ := limiter.Allow("client-1", earlier); ok {
		t.Fatal("request at earlier timestamp must be rejected when bucket is empty")
	}
	// Calling again at t0 must still be rejected (no artificial refill from rewound lastSeen).
	if ok, _ := limiter.Allow("client-1", t0); ok {
		t.Fatal("request at t0 after earlier timestamp must not receive refilled tokens")
	}
}

func TestOAuthAuthorize_TrustedProxiesRateLimit(t *testing.T) {
	store := newInternalTestStore(t)
	reg := bbs.NewProviderRegistry(store.Key())
	reg.Register(store)
	svc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)
	hub := NewEventHub(64)

	// Invalid trusted_proxies entry must fail closed.
	if _, err := NewServer(ServerDeps{
		Config: ServerConfig{
			Enabled:        true,
			ListenAddr:     "127.0.0.1:0",
			PublicURL:      "https://origin.example.com",
			TrustedProxies: []string{"not-an-ip"},
		},
		Service: svc,
		Hub:     hub,
	}); err == nil {
		t.Fatal("expected NewServer to reject invalid trusted_proxies entry")
	}

	srv, err := NewServer(ServerDeps{
		Config: ServerConfig{
			Enabled:        true,
			ListenAddr:     "127.0.0.1:0",
			PublicURL:      "https://origin.example.com",
			TrustedProxies: []string{"127.0.0.1", "10.0.0.0/8"},
			OAuth: OAuthConfig{
				ClientID:     "discord-client",
				ClientSecret: "discord-secret",
			},
			Peers: []PeerConfig{
				{
					ID:           "peer-a",
					Token:        "peer-tok-a",
					AllowWrite:   true,
					RedirectURIs: []string{"https://listener.example.com/api/v1/bbs/federation/client/callback"},
				},
			},
		},
		Service: svc,
		Hub:     hub,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	srv.oauth.authLimiter = newOAuthRateLimiter(1, 0.01)

	reqURL := "/api/v1/bbs/federation/oauth/authorize?client_id=peer-a&redirect_uri=" +
		url.QueryEscape("https://listener.example.com/api/v1/bbs/federation/client/callback") +
		"&response_type=code&scope=bbs.write&state=st&code_challenge=" +
		ComputePKCES256("test-verifier-123456789012345678901234567890123") +
		"&code_challenge_method=S256"

	// Behind trusted proxy (127.0.0.1 -> 10.0.0.2): client A gets limited independently of client B.
	reqA1 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqA1.RemoteAddr = "127.0.0.1:40000"
	reqA1.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.2")
	recA1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recA1, reqA1)
	if recA1.Code != http.StatusFound {
		t.Fatalf("client A first request expected 302, got %d", recA1.Code)
	}

	reqA2 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqA2.RemoteAddr = "127.0.0.1:40001"
	reqA2.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.2")
	recA2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recA2, reqA2)
	if recA2.Code != http.StatusTooManyRequests {
		t.Fatalf("client A second request expected 429, got %d", recA2.Code)
	}

	reqB := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqB.RemoteAddr = "127.0.0.1:40002"
	reqB.Header.Set("X-Forwarded-For", "198.51.100.2, 10.0.0.2")
	recB := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recB, reqB)
	if recB.Code != http.StatusFound {
		t.Fatalf("client B behind same trusted proxy expected 302, got %d", recB.Code)
	}

	// Untrusted peer spoofing X-Forwarded-For must be keyed by RemoteAddr, not the spoofed header.
	reqUntrusted1 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqUntrusted1.RemoteAddr = "203.0.113.50:50000"
	reqUntrusted1.Header.Set("X-Forwarded-For", "198.51.100.99")
	recU1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recU1, reqUntrusted1)
	if recU1.Code != http.StatusFound {
		t.Fatalf("untrusted first request expected 302, got %d", recU1.Code)
	}

	reqUntrusted2 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	reqUntrusted2.RemoteAddr = "203.0.113.50:50001"
	reqUntrusted2.Header.Set("X-Forwarded-For", "198.51.100.100")
	recU2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recU2, reqUntrusted2)
	if recU2.Code != http.StatusTooManyRequests {
		t.Fatalf("untrusted second request with spoofed XFF expected 429, got %d", recU2.Code)
	}
}
