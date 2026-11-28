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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

const (
	ProtocolVersion        = 1
	HeaderUserToken        = "X-BBS-User-Token"
	DefaultEventBufferSize = 2048
	DefaultAccessTokenTTL  = 60 * time.Minute
	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
	DefaultRemoteCacheTTL  = 60 * time.Second
	DefaultRemoteTimeout   = 15 * time.Second
	RemoteKeyPrefix        = "remote:"

	CodeReadOnly              = "read_only"
	CodeAuthRequired          = "auth_required"
	CodeTokenExpired          = "token_expired"
	CodeTokenRevoked          = "token_revoked"
	CodePolicyConsentRequired = "policy_consent_required"
	CodeBanned                = "banned"
	CodeRateLimited           = "rate_limited"
	CodeArchived              = "archived"
	CodeMaxRes                = "max_res"
	CodeBoardLocked           = "board_locked"
	CodeCapNotAuthorized      = "cap_not_authorized"
	CodeNotFound              = "not_found"
	CodeInvalidGrant          = "invalid_grant"
	CodeInvalidInput          = "invalid_input"
	CodeUnauthorized          = "unauthorized"
	CodeUnavailable           = "unavailable"
)

// ServerConfig configures the Origin federation HTTP/SSE + OAuth2 authorization server.
type ServerConfig struct {
	Enabled              bool
	ListenAddr           string
	PublicURL            string
	BasePath             string
	TrustedProxies       []string
	ParsedTrustedProxies []netip.Prefix
	BufferSize           int
	AccessTTL            time.Duration
	RefreshTTL           time.Duration
	OAuth                OAuthConfig
	Peers                []PeerConfig
}

// OAuthConfig configures the Origin federation-dedicated Discord OAuth2 application.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// PeerConfig configures one federated listener instance authorized to connect to Origin.
type PeerConfig struct {
	ID              string
	Name            string
	Token           string
	AllowWrite      bool
	AllowReport     *bool
	AllowCaps       bool
	RedirectURIs    []string
	AllowedBoardIDs []string

	ParsedBoardIDs map[uuid.UUID]struct{}
}

// EffectiveAllowReport returns whether this peer is permitted to forward user reports.
func (p PeerConfig) EffectiveAllowReport() bool {
	if p.AllowReport != nil {
		return *p.AllowReport
	}
	return p.AllowWrite
}

// AllowsBoard reports whether the peer is allowed to view/access boardID.
func (p PeerConfig) AllowsBoard(boardID uuid.UUID) bool {
	if len(p.ParsedBoardIDs) == 0 {
		return true
	}
	_, ok := p.ParsedBoardIDs[boardID]
	return ok
}

// ClientConfig configures the Listener OAuth2 callback HTTP server.
type ClientConfig struct {
	Enabled        bool
	ListenAddr     string
	PublicURL      string
	BasePath       string
	TrustedProxies []string
}

// RemoteConfig configures one Origin remote mounted on the Listener instance.
type RemoteConfig struct {
	Key      string
	Label    string
	BaseURL  string
	PeerID   string
	Token    string
	ReadOnly bool
	CacheTTL time.Duration
	Timeout  time.Duration
}

// NormalizeBasePath cleans an optional URL base path ("" or "/prefix" without trailing slash).
func NormalizeBasePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func envVarSuffix(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(unicode.ToUpper(r))
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ResolveServerConfig applies defaults and env fallbacks and validates ServerConfig.
func ResolveServerConfig(cfg ServerConfig) (ServerConfig, error) {
	if !cfg.Enabled {
		return cfg, nil
	}
	cfg.ListenAddr = strings.TrimSpace(cfg.ListenAddr)
	if cfg.ListenAddr == "" {
		return cfg, errors.New("federation server: listen_addr is required when enabled")
	}
	cfg.BasePath = NormalizeBasePath(cfg.BasePath)
	cfg.PublicURL = strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/")
	parsedProxies, err := parseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return cfg, err
	}
	cfg.ParsedTrustedProxies = parsedProxies
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = DefaultEventBufferSize
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = DefaultAccessTokenTTL
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = DefaultRefreshTokenTTL
	}
	if strings.TrimSpace(cfg.OAuth.ClientID) == "" {
		cfg.OAuth.ClientID = strings.TrimSpace(os.Getenv("BBS_FEDERATION_OAUTH_CLIENT_ID"))
	} else {
		cfg.OAuth.ClientID = strings.TrimSpace(cfg.OAuth.ClientID)
	}
	if strings.TrimSpace(cfg.OAuth.ClientSecret) == "" {
		cfg.OAuth.ClientSecret = strings.TrimSpace(os.Getenv("BBS_FEDERATION_OAUTH_CLIENT_SECRET"))
	} else {
		cfg.OAuth.ClientSecret = strings.TrimSpace(cfg.OAuth.ClientSecret)
	}

	seenIDs := make(map[string]struct{}, len(cfg.Peers))
	seenTokHashes := make(map[string]struct{}, len(cfg.Peers))
	needsOAuth := false

	resolvedPeers := make([]PeerConfig, 0, len(cfg.Peers))
	for _, p := range cfg.Peers {
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" {
			return cfg, errors.New("federation server: peer id is required")
		}
		if _, dup := seenIDs[p.ID]; dup {
			return cfg, fmt.Errorf("federation server: duplicate peer id %q", p.ID)
		}
		seenIDs[p.ID] = struct{}{}

		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			p.Name = p.ID
		}
		tok := strings.TrimSpace(p.Token)
		if tok == "" {
			tok = strings.TrimSpace(os.Getenv("BBS_FEDERATION_SERVER_PEER_TOKEN_" + envVarSuffix(p.ID)))
		}
		if tok == "" {
			return cfg, fmt.Errorf("federation server: token is required for peer %q", p.ID)
		}
		th := HashToken(tok)
		if _, dup := seenTokHashes[th]; dup {
			return cfg, fmt.Errorf("federation server: duplicate token across peers (peer %q)", p.ID)
		}
		seenTokHashes[th] = struct{}{}
		p.Token = tok

		boardSet := make(map[uuid.UUID]struct{}, len(p.AllowedBoardIDs))
		for _, rawID := range p.AllowedBoardIDs {
			uid, err := uuid.Parse(strings.TrimSpace(rawID))
			if err != nil {
				return cfg, fmt.Errorf("federation server: invalid allowed_board_ids entry %q for peer %q: %w", rawID, p.ID, err)
			}
			boardSet[uid] = struct{}{}
		}
		p.ParsedBoardIDs = boardSet

		var cleanURIs []string
		for _, u := range p.RedirectURIs {
			trimmed := strings.TrimSpace(u)
			if trimmed == "" {
				continue
			}
			parsed, err := url.Parse(trimmed)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return cfg, fmt.Errorf("federation server: invalid redirect_uri %q for peer %q", u, p.ID)
			}
			cleanURIs = append(cleanURIs, trimmed)
		}
		p.RedirectURIs = cleanURIs

		if p.AllowWrite || p.EffectiveAllowReport() {
			needsOAuth = true
			if len(p.RedirectURIs) == 0 {
				return cfg, fmt.Errorf("federation server: peer %q has allow_write/allow_report enabled so redirect_uris is required", p.ID)
			}
		}
		resolvedPeers = append(resolvedPeers, p)
	}
	cfg.Peers = resolvedPeers

	if needsOAuth {
		parsedPub, err := url.Parse(cfg.PublicURL)
		if err != nil || (parsedPub.Scheme != "http" && parsedPub.Scheme != "https") || parsedPub.Host == "" {
			return cfg, fmt.Errorf("federation server: valid public_url is required when OAuth write/report peers are configured (got %q)", cfg.PublicURL)
		}
		if cfg.OAuth.ClientID == "" || cfg.OAuth.ClientSecret == "" {
			return cfg, errors.New("federation server: federation oauth client_id and client_secret are required when allow_write/allow_report peers exist")
		}
	}
	return cfg, nil
}

// ResolveClientConfig applies defaults and validates the Listener OAuth callback server config.
func ResolveClientConfig(cfg ClientConfig) (ClientConfig, error) {
	if !cfg.Enabled {
		return cfg, nil
	}
	cfg.ListenAddr = strings.TrimSpace(cfg.ListenAddr)
	if cfg.ListenAddr == "" {
		return cfg, errors.New("federation client: listen_addr is required when enabled")
	}
	cfg.BasePath = NormalizeBasePath(cfg.BasePath)
	cfg.PublicURL = strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/")
	parsed, err := url.Parse(cfg.PublicURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return cfg, fmt.Errorf("federation client: valid public_url is required when enabled (got %q)", cfg.PublicURL)
	}
	return cfg, nil
}

// ResolveRemoteConfig applies defaults and env token fallbacks for a remote entry.
func ResolveRemoteConfig(cfg RemoteConfig) (RemoteConfig, error) {
	cfg.Key = strings.TrimSpace(cfg.Key)
	if cfg.Key == "" {
		return cfg, errors.New("federation remote: key is required")
	}
	if strings.ContainsAny(cfg.Key, "/\\:") {
		return cfg, fmt.Errorf("federation remote: key %q contains forbidden characters", cfg.Key)
	}
	cfg.Label = strings.TrimSpace(cfg.Label)
	if cfg.Label == "" {
		cfg.Label = cfg.Key
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return cfg, fmt.Errorf("federation remote %q: invalid base_url %q", cfg.Key, cfg.BaseURL)
	}
	cfg.PeerID = strings.TrimSpace(cfg.PeerID)
	cfg.Token = strings.TrimSpace(cfg.Token)
	if cfg.Token == "" {
		cfg.Token = strings.TrimSpace(os.Getenv("BBS_FEDERATION_REMOTE_TOKEN_" + envVarSuffix(cfg.Key)))
	}
	if cfg.Token == "" {
		return cfg, fmt.Errorf("federation remote %q: token is required", cfg.Key)
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = DefaultRemoteCacheTTL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultRemoteTimeout
	}
	return cfg, nil
}

// HashToken hashes a high-entropy bearer/refresh/code token using SHA-256 hex.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// GenerateRandomToken returns a URL-safe random token with nBytes of CSPRNG entropy.
func GenerateRandomToken(nBytes int) (string, error) {
	if nBytes <= 0 {
		nBytes = 32
	}
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ComputePKCES256 computes the RFC 7636 S256 code_challenge from code_verifier.
func ComputePKCES256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TokenCipher encrypts/decrypts delegated Origin tokens at rest using AES-256-GCM.
type TokenCipher struct {
	aead cipher.AEAD
}

// NewTokenCipher derives an AES-256-GCM cipher from a persistent bot secret.
func NewTokenCipher(secret []byte) (*TokenCipher, error) {
	if len(secret) == 0 {
		return nil, errors.New("federation: empty cipher secret")
	}
	h := sha256.New()
	_, _ = h.Write([]byte("gobot:bbs:federation:token:v1:"))
	_, _ = h.Write(secret)
	key := h.Sum(nil)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TokenCipher{aead: aead}, nil
}

// Encrypt seals plaintext into a versioned base64url AES-256-GCM ciphertext.
func (c *TokenCipher) Encrypt(plaintext string) (string, error) {
	if c == nil || c.aead == nil {
		return "", errors.New("federation: token cipher uninitialized")
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return "v1:" + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt authenticates and recovers plaintext from a sealed token string.
func (c *TokenCipher) Decrypt(ciphertext string) (string, error) {
	if c == nil || c.aead == nil {
		return "", errors.New("federation: token cipher uninitialized")
	}
	raw, ok := strings.CutPrefix(ciphertext, "v1:")
	if !ok {
		return "", errors.New("federation: unsupported encrypted token format")
	}
	buf, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	ns := c.aead.NonceSize()
	if len(buf) < ns {
		return "", errors.New("federation: truncated ciphertext")
	}
	plain, err := c.aead.Open(nil, buf[:ns], buf[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// --- Wire Protocol DTOs (never expose AuthorUserID) ---

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// APIError represents a structured HTTP error returned by Origin federation endpoints.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("federation api (%d %s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("federation api (%d): %s", e.StatusCode, e.Message)
}

type CapabilitiesResponse struct {
	ProtocolVersion int         `json:"protocol_version"`
	InstanceEpoch   string      `json:"instance_epoch"`
	LatestSeq       uint64      `json:"latest_seq"`
	PeerID          string      `json:"peer_id"`
	AllowWrite      bool        `json:"allow_write"`
	AllowReport     bool        `json:"allow_report"`
	AllowCaps       bool        `json:"allow_caps"`
	AllowedBoardIDs []uuid.UUID `json:"allowed_board_ids,omitempty"`
	PolicyVersion   string      `json:"policy_version"`
}

type CategoryListResponse struct {
	Categories []bbs.Category `json:"categories"`
}

type BoardListResponse struct {
	Boards []bbs.Board `json:"boards"`
}

type ThreadListResponse struct {
	Threads []bbs.Thread `json:"threads"`
}

type PostListResponse struct {
	Posts []bbs.Post `json:"posts"`
}

type CapsListResponse struct {
	Caps []bbs.Cap `json:"caps"`
}

// CreateThreadRequest carries a delegated thread creation payload from a trusted peer.
// Trust boundary: GuildID and SlipContext are self-reported by the peer authenticated via its
// shared peer bearer token and are not independently verifiable by Origin.
type CreateThreadRequest struct {
	GuildID     snowflake.ID    `json:"guild_id"`
	Title       string          `json:"title"`
	RawName     string          `json:"raw_name"`
	CapID       string          `json:"cap_id"`
	RawBody     string          `json:"raw_body"`
	SlipContext bbs.SlipContext `json:"slip_context"`
}

type CreateThreadResponse struct {
	Board  bbs.Board  `json:"board"`
	Thread bbs.Thread `json:"thread"`
	Post   bbs.Post   `json:"post"`
}

// CreatePostRequest carries a delegated reply payload from a trusted peer.
// Trust boundary: GuildID and SlipContext are self-reported by the authenticated peer.
type CreatePostRequest struct {
	GuildID     snowflake.ID    `json:"guild_id"`
	RawName     string          `json:"raw_name"`
	CapID       string          `json:"cap_id"`
	RawBody     string          `json:"raw_body"`
	SlipContext bbs.SlipContext `json:"slip_context"`
}

type CreatePostResponse struct {
	Board        bbs.Board  `json:"board"`
	Thread       bbs.Thread `json:"thread"`
	Post         bbs.Post   `json:"post"`
	BackwardRefs []uint32   `json:"backward_refs,omitempty"`
}

// CreateReportRequest carries a delegated post report payload from a trusted peer.
// Trust boundary: GuildID is self-reported by the authenticated peer.
type CreateReportRequest struct {
	GuildID snowflake.ID `json:"guild_id"`
	Reason  string       `json:"reason"`
}

type CreateReportResponse struct {
	ID        uuid.UUID        `json:"id"`
	Scope     bbs.Scope        `json:"scope"`
	GuildID   snowflake.ID     `json:"guild_id"`
	PostID    uuid.UUID        `json:"post_id"`
	ThreadID  uuid.UUID        `json:"thread_id"`
	BoardID   uuid.UUID        `json:"board_id"`
	Reason    string           `json:"reason"`
	Status    bbs.ReportStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type TokenResponse struct {
	TokenType        string       `json:"token_type"`
	AccessToken      string       `json:"access_token"`
	ExpiresIn        int64        `json:"expires_in"`
	RefreshToken     string       `json:"refresh_token"`
	RefreshExpiresIn int64        `json:"refresh_expires_in"`
	UserID           snowflake.ID `json:"user_id"`
	Scopes           []string     `json:"scopes"`
	PolicyVersion    string       `json:"policy_version"`
}

type RevokeRequest struct {
	UserID snowflake.ID `json:"user_id"`
	Token  string       `json:"token,omitempty"`
}

type WireEvent struct {
	Seq          uint64        `json:"seq"`
	Epoch        string        `json:"epoch"`
	Type         bbs.EventType `json:"type"`
	BoardID      uuid.UUID     `json:"board_id"`
	ThreadID     uuid.UUID     `json:"thread_id,omitempty"`
	PostID       uuid.UUID     `json:"post_id,omitempty"`
	Board        *bbs.Board    `json:"board,omitempty"`
	Thread       *bbs.Thread   `json:"thread,omitempty"`
	Post         *bbs.Post     `json:"post,omitempty"`
	BackwardRefs []uint32      `json:"backward_refs,omitempty"`
	HitMaxRes    bool          `json:"hit_max_res,omitempty"`
	OriginPeer   string        `json:"origin_peer,omitempty"`
	OccurredAt   time.Time     `json:"occurred_at"`
}

// MapBBSErrorToHTTP translates BBS domain sentinel errors to federation HTTP status codes and wire error codes.
func MapBBSErrorToHTTP(err error) (status int, code string) {
	switch {
	case errors.Is(err, bbs.ErrFederationReadOnly):
		return 403, CodeReadOnly
	case errors.Is(err, bbs.ErrFederationAuthRequired):
		return 401, CodeAuthRequired
	case errors.Is(err, bbs.ErrConsentRequired), errors.Is(err, bbs.ErrStalePolicyVersion):
		return 403, CodePolicyConsentRequired
	case errors.Is(err, bbs.ErrBanned):
		return 403, CodeBanned
	case errors.Is(err, bbs.ErrRateLimited):
		return 429, CodeRateLimited
	case errors.Is(err, bbs.ErrThreadArchived):
		return 409, CodeArchived
	case errors.Is(err, bbs.ErrMaxResReached):
		return 409, CodeMaxRes
	case errors.Is(err, bbs.ErrBoardLocked):
		return 403, CodeBoardLocked
	case errors.Is(err, bbs.ErrCapNotAuthorized):
		return 403, CodeCapNotAuthorized
	case errors.Is(err, bbs.ErrNotFound):
		return 404, CodeNotFound
	case errors.Is(err, bbs.ErrUnauthorized):
		return 403, CodeUnauthorized
	case errors.Is(err, bbs.ErrInvalidInput):
		return 400, CodeInvalidInput
	case errors.Is(err, bbs.ErrUnsupportedOperation):
		return 403, CodeReadOnly
	default:
		return 500, CodeUnavailable
	}
}

// MapAPIErrorToBBS converts a wire error response into a domain error compatible with errors.Is.
func MapAPIErrorToBBS(apiErr *APIError) error {
	if apiErr == nil {
		return nil
	}
	switch apiErr.Code {
	case CodeReadOnly:
		return fmt.Errorf("%w: %s", bbs.ErrFederationReadOnly, apiErr.Message)
	case CodeAuthRequired, CodeTokenExpired, CodeTokenRevoked, CodeInvalidGrant, CodePolicyConsentRequired:
		return fmt.Errorf("%w (%s): %s", bbs.ErrFederationAuthRequired, apiErr.Code, apiErr.Message)
	case CodeBanned:
		return fmt.Errorf("%w: %s", bbs.ErrBanned, apiErr.Message)
	case CodeRateLimited:
		return fmt.Errorf("%w: %s", bbs.ErrRateLimited, apiErr.Message)
	case CodeArchived:
		return fmt.Errorf("%w: %s", bbs.ErrThreadArchived, apiErr.Message)
	case CodeMaxRes:
		return fmt.Errorf("%w: %s", bbs.ErrMaxResReached, apiErr.Message)
	case CodeBoardLocked:
		return fmt.Errorf("%w: %s", bbs.ErrBoardLocked, apiErr.Message)
	case CodeCapNotAuthorized:
		return fmt.Errorf("%w: %s", bbs.ErrCapNotAuthorized, apiErr.Message)
	case CodeNotFound:
		return fmt.Errorf("%w: %s", bbs.ErrNotFound, apiErr.Message)
	case CodeUnauthorized:
		return fmt.Errorf("%w: %s", bbs.ErrUnauthorized, apiErr.Message)
	case CodeInvalidInput:
		return fmt.Errorf("%w: %s", bbs.ErrInvalidInput, apiErr.Message)
	default:
		if apiErr.StatusCode >= 500 || apiErr.StatusCode == 0 {
			return fmt.Errorf("%w: %s", bbs.ErrFederationUnavailable, apiErr.Message)
		}
		if apiErr.StatusCode == 401 {
			return fmt.Errorf("%w: %s", bbs.ErrUnauthorized, apiErr.Message)
		}
		if apiErr.StatusCode == 404 {
			return fmt.Errorf("%w: %s", bbs.ErrNotFound, apiErr.Message)
		}
		return fmt.Errorf("%w: %s", bbs.ErrFederationUnavailable, apiErr.Message)
	}
}

func parsePositiveUint32(raw string) (uint32, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func parseTrustedProxies(entries []string) ([]netip.Prefix, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]netip.Prefix, 0, len(entries))
	for _, raw := range entries {
		s := strings.TrimSpace(raw)
		if s == "" {
			return nil, errors.New("federation server: trusted_proxies entry must not be empty")
		}
		if pfx, err := netip.ParsePrefix(s); err == nil {
			if pfx.Addr().Is4In6() {
				out = append(out, netip.PrefixFrom(pfx.Addr().Unmap(), max(0, pfx.Bits()-96)).Masked())
			} else {
				out = append(out, pfx.Masked())
			}
			continue
		}
		if addr, err := netip.ParseAddr(s); err == nil {
			u := addr.Unmap()
			out = append(out, netip.PrefixFrom(u, u.BitLen()))
			continue
		}
		return nil, fmt.Errorf("federation server: invalid trusted_proxies entry %q", raw)
	}
	return out, nil
}

func ipInPrefixes(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, pfx := range prefixes {
		if pfx.Contains(addr) {
			return true
		}
	}
	return false
}
