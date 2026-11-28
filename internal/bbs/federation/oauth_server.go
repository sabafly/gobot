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
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

const (
	defaultDiscordAPIBase       = "https://discord.com/api/v10"
	defaultDiscordAuthorizeBase = "https://discord.com/oauth2/authorize"
	oauthCookieName             = "bbs_fed_oauth"
	pendingOAuthTxTTL           = 10 * time.Minute
	pendingOAuthCodeTTL         = 60 * time.Second
	maxPendingOAuthEntries      = 512
)

type pendingOAuthTx struct {
	TxID               string
	PeerID             string
	RedirectURI        string
	RequestedScopes    []string
	ClientState        string
	CodeChallenge      string
	LoginHint          string
	DiscordState       string
	DiscordUserID      snowflake.ID
	DiscordUsername    string
	DiscordDisplayName string
	SessionCookieHash  string
	CSRFToken          string
	ExpiresAt          time.Time
}

type pendingOAuthCode struct {
	PeerID        string
	RedirectURI   string
	UserID        snowflake.ID
	Scopes        []string
	PolicyVersion string
	CodeChallenge string
	ExpiresAt     time.Time
}

type oauthManager struct {
	srv            *Server
	discordAPIBase string
	discordAuthURL string
	httpClient     *http.Client

	mu          sync.Mutex
	txByID      map[string]*pendingOAuthTx
	txByDiscord map[string]string
	codesByHash map[string]*pendingOAuthCode
}

func newOAuthManager(srv *Server, apiBase, authBase string, client *http.Client) *oauthManager {
	if strings.TrimSpace(apiBase) == "" {
		apiBase = defaultDiscordAPIBase
	}
	if strings.TrimSpace(authBase) == "" {
		authBase = defaultDiscordAuthorizeBase
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &oauthManager{
		srv:            srv,
		discordAPIBase: strings.TrimRight(apiBase, "/"),
		discordAuthURL: authBase,
		httpClient:     client,
		txByID:         make(map[string]*pendingOAuthTx),
		txByDiscord:    make(map[string]string),
		codesByHash:    make(map[string]*pendingOAuthCode),
	}
}

func (m *oauthManager) pruneLocked(now time.Time) {
	for id, tx := range m.txByID {
		if now.After(tx.ExpiresAt) {
			delete(m.txByDiscord, tx.DiscordState)
			delete(m.txByID, id)
		}
	}
	for ds, id := range m.txByDiscord {
		if _, ok := m.txByID[id]; !ok {
			delete(m.txByDiscord, ds)
		}
	}
	for ch, c := range m.codesByHash {
		if now.After(c.ExpiresAt) {
			delete(m.codesByHash, ch)
		}
	}
}

func (m *oauthManager) discordCallbackURL() string {
	return m.srv.cfg.PublicURL + m.srv.basePath + "/api/v1/bbs/federation/oauth/callback"
}

func (m *oauthManager) cookiePath() string {
	return m.srv.basePath + "/api/v1/bbs/federation/oauth"
}

func (m *oauthManager) isHTTPS() bool {
	return strings.HasPrefix(strings.ToLower(m.srv.cfg.PublicURL), "https://")
}

func (m *oauthManager) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthCookieName,
		Value:    "",
		Path:     m.cookiePath(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.isHTTPS(),
		SameSite: http.SameSiteLaxMode,
	})
}

func isValidCodeChallenge(cc string) bool {
	if len(cc) < 43 || len(cc) > 128 {
		return false
	}
	for i := 0; i < len(cc); i++ {
		b := cc[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '-' && b != '_' {
			return false
		}
	}
	return true
}

func validateRequestedScopes(peer PeerConfig, rawScope string) ([]string, error) {
	parsed := bbs.ParseFederationScopes(rawScope)
	if len(parsed) == 0 {
		if peer.AllowWrite {
			parsed = append(parsed, bbs.FederationScopeWrite)
		}
		if peer.EffectiveAllowReport() {
			parsed = append(parsed, bbs.FederationScopeReport)
		}
		if peer.AllowCaps {
			parsed = append(parsed, bbs.FederationScopeCaps)
		}
	}
	if len(parsed) == 0 {
		return nil, errors.New("no scopes requested or permitted")
	}
	var allowed []string
	for _, sc := range parsed {
		switch sc {
		case bbs.FederationScopeWrite:
			if !peer.AllowWrite {
				return nil, fmt.Errorf("scope %q not allowed for peer", sc)
			}
			allowed = append(allowed, sc)
		case bbs.FederationScopeReport:
			if !peer.EffectiveAllowReport() {
				return nil, fmt.Errorf("scope %q not allowed for peer", sc)
			}
			allowed = append(allowed, sc)
		case bbs.FederationScopeCaps:
			if !peer.AllowCaps {
				return nil, fmt.Errorf("scope %q not allowed for peer", sc)
			}
			allowed = append(allowed, sc)
		default:
			return nil, fmt.Errorf("unknown scope %q", sc)
		}
	}
	if !slices.Contains(allowed, bbs.FederationScopeWrite) && !slices.Contains(allowed, bbs.FederationScopeReport) {
		return nil, errors.New("at least write or report scope is required")
	}
	return allowed, nil
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, redirectURI, errCode, state string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		renderStatusPage(w, http.StatusBadRequest, false, "Authorization Error", errCode, "")
		return
	}
	q := u.Query()
	q.Set("error", errCode)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (m *oauthManager) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := strings.TrimSpace(q.Get("client_id"))
	redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
	responseType := strings.TrimSpace(q.Get("response_type"))
	state := q.Get("state")
	codeChallenge := strings.TrimSpace(q.Get("code_challenge"))
	challengeMethod := strings.TrimSpace(q.Get("code_challenge_method"))
	loginHint := strings.TrimSpace(q.Get("login_hint"))

	peer, ok := m.srv.peerByID(clientID)
	if !ok {
		renderStatusPage(w, http.StatusBadRequest, false, "連携エラー / Invalid Client", "指定された client_id (peer) が登録されていません。", clientID)
		return
	}
	if !peer.AllowWrite && !peer.EffectiveAllowReport() {
		renderStatusPage(w, http.StatusForbidden, false, "連携不可 / Read-Only Peer", "この連携クライアントには書き込み・通報権限が許可されていません。", peer.ID)
		return
	}
	if redirectURI == "" || !slices.Contains(peer.RedirectURIs, redirectURI) {
		renderStatusPage(w, http.StatusBadRequest, false, "リダイレクト先不正 / Invalid Redirect URI", "redirect_uri が本番側で許可された URI と一致しません。", "")
		return
	}
	if responseType != "code" {
		redirectOAuthError(w, r, redirectURI, "unsupported_response_type", state)
		return
	}
	if challengeMethod != "S256" || !isValidCodeChallenge(codeChallenge) {
		redirectOAuthError(w, r, redirectURI, "invalid_request", state)
		return
	}
	scopes, sErr := validateRequestedScopes(peer, q.Get("scope"))
	if sErr != nil {
		redirectOAuthError(w, r, redirectURI, "invalid_scope", state)
		return
	}
	if loginHint != "" {
		uid, pErr := snowflake.Parse(loginHint)
		if pErr != nil || uid == 0 {
			redirectOAuthError(w, r, redirectURI, "invalid_request", state)
			return
		}
		loginHint = uid.String()
	}

	txID, err := GenerateRandomToken(16)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "failed to generate transaction id", "")
		return
	}
	discordState, err := GenerateRandomToken(24)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "failed to generate oauth state", "")
		return
	}

	now := time.Now().UTC()
	m.mu.Lock()
	m.pruneLocked(now)
	if len(m.txByID) >= maxPendingOAuthEntries {
		m.mu.Unlock()
		redirectOAuthError(w, r, redirectURI, "temporarily_unavailable", state)
		return
	}
	m.txByID[txID] = &pendingOAuthTx{
		TxID:            txID,
		PeerID:          peer.ID,
		RedirectURI:     redirectURI,
		RequestedScopes: scopes,
		ClientState:     state,
		CodeChallenge:   codeChallenge,
		LoginHint:       loginHint,
		DiscordState:    discordState,
		ExpiresAt:       now.Add(pendingOAuthTxTTL),
	}
	m.txByDiscord[discordState] = txID
	m.mu.Unlock()

	authURL, err := url.Parse(m.discordAuthURL)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Configuration Error", "invalid discord authorize URL", "")
		return
	}
	aq := authURL.Query()
	aq.Set("client_id", m.srv.cfg.OAuth.ClientID)
	aq.Set("redirect_uri", m.discordCallbackURL())
	aq.Set("response_type", "code")
	aq.Set("scope", "identify")
	aq.Set("state", discordState)
	aq.Set("prompt", "consent")
	authURL.RawQuery = aq.Encode()
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (m *oauthManager) handleDiscordCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	discordState := q.Get("state")
	code := strings.TrimSpace(q.Get("code"))
	upstreamErr := strings.TrimSpace(q.Get("error"))

	now := time.Now().UTC()
	m.mu.Lock()
	m.pruneLocked(now)
	txID, ok := m.txByDiscord[discordState]
	var txCopy pendingOAuthTx
	if ok {
		delete(m.txByDiscord, discordState)
		if tx, exists := m.txByID[txID]; exists {
			txCopy = *tx
		} else {
			ok = false
		}
	}
	m.mu.Unlock()

	if !ok {
		renderStatusPage(w, http.StatusBadRequest, false, "セッション期限切れ / Invalid State", "認可セッションが見つからないか期限切れです。Discord からもう一度やり直してください。", "")
		return
	}

	deleteTx := func() {
		m.mu.Lock()
		delete(m.txByID, txCopy.TxID)
		m.mu.Unlock()
	}

	if upstreamErr != "" || code == "" {
		deleteTx()
		redirectOAuthError(w, r, txCopy.RedirectURI, "access_denied", txCopy.ClientState)
		return
	}

	accessToken, err := m.exchangeDiscordCode(r.Context(), code)
	if err != nil {
		deleteTx()
		slog.Warn("federation oauth discord token exchange failed", "peer_id", txCopy.PeerID, "err", err)
		renderStatusPage(w, http.StatusBadGateway, false, "Discord 認証失敗 / Discord Auth Error", "Discord トークン交換に失敗しました。", "")
		return
	}
	user, err := m.fetchDiscordUser(r.Context(), accessToken)
	if err != nil {
		deleteTx()
		slog.Warn("federation oauth discord user lookup failed", "peer_id", txCopy.PeerID, "err", err)
		renderStatusPage(w, http.StatusBadGateway, false, "Discord ユーザー取得失敗", "Discord ユーザー情報の取得に失敗しました。", "")
		return
	}
	uid, err := snowflake.Parse(user.ID)
	if err != nil || uid == 0 {
		deleteTx()
		renderStatusPage(w, http.StatusBadGateway, false, "Discord ユーザーID不正", "有効な Discord ユーザーIDを取得できませんでした。", "")
		return
	}
	if txCopy.LoginHint != "" && txCopy.LoginHint != uid.String() {
		deleteTx()
		renderStatusPage(w, http.StatusForbidden, false, "アカウント不一致 / Account Mismatch", "Discord 上で操作したユーザーと異なる Discord アカウントでログインされました。同じアカウントでやり直してください。", "")
		return
	}

	posterHash := ""
	if m.srv.svc.Identity != nil {
		posterHash = m.srv.svc.Identity.GeneratePosterHash(bbs.ScopeGlobal, 0, uid)
	}
	banned, err := m.srv.svc.Store.IsBanned(r.Context(), bbs.ScopeGlobal, 0, posterHash, uid, now)
	if err != nil {
		deleteTx()
		renderStatusPage(w, http.StatusInternalServerError, false, "確認エラー", "アカウント状態の確認に失敗しました。", "")
		return
	}
	if banned {
		deleteTx()
		renderStatusPage(w, http.StatusForbidden, false, "利用制限中 / Globally Banned", "この Discord アカウントは本番掲示板でグローバル規制中のため連携を許可できません。", "")
		return
	}

	displayName := user.Username
	if user.GlobalName != nil && strings.TrimSpace(*user.GlobalName) != "" {
		displayName = strings.TrimSpace(*user.GlobalName)
	}

	cookieSecret, err := GenerateRandomToken(32)
	if err != nil {
		deleteTx()
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "failed to generate session secret", "")
		return
	}
	csrfTok, err := GenerateRandomToken(24)
	if err != nil {
		deleteTx()
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "failed to generate csrf token", "")
		return
	}

	m.mu.Lock()
	tx, stillExists := m.txByID[txCopy.TxID]
	if stillExists && !now.After(tx.ExpiresAt) {
		tx.DiscordUserID = uid
		tx.DiscordUsername = user.Username
		tx.DiscordDisplayName = displayName
		tx.SessionCookieHash = HashToken(cookieSecret)
		tx.CSRFToken = csrfTok
	} else {
		stillExists = false
	}
	m.mu.Unlock()
	if !stillExists {
		renderStatusPage(w, http.StatusBadRequest, false, "セッション期限切れ / Expired", "認可セッションがタイムアウトしました。Discord からやり直してください。", "")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthCookieName,
		Value:    cookieSecret,
		Path:     m.cookiePath(),
		MaxAge:   int(pendingOAuthTxTTL.Seconds()),
		HttpOnly: true,
		Secure:   m.isHTTPS(),
		SameSite: http.SameSiteLaxMode,
	})
	consentURL := m.srv.basePath + "/api/v1/bbs/federation/oauth/consent?tx=" + url.QueryEscape(txCopy.TxID)
	http.Redirect(w, r, consentURL, http.StatusFound)
}

func (m *oauthManager) loadVerifiedConsentTx(r *http.Request, txID string) (pendingOAuthTx, PeerConfig, bool) {
	cookie, err := r.Cookie(oauthCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return pendingOAuthTx{}, PeerConfig{}, false
	}
	cookieHash := HashToken(cookie.Value)
	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(now)
	tx, ok := m.txByID[txID]
	if !ok || tx.SessionCookieHash == "" || tx.DiscordUserID == 0 {
		return pendingOAuthTx{}, PeerConfig{}, false
	}
	if subtle.ConstantTimeCompare([]byte(cookieHash), []byte(tx.SessionCookieHash)) != 1 {
		return pendingOAuthTx{}, PeerConfig{}, false
	}
	peer, pOK := m.srv.peerByID(tx.PeerID)
	if !pOK {
		return pendingOAuthTx{}, PeerConfig{}, false
	}
	return *tx, peer, true
}

func detectConsentLang(r *http.Request) string {
	al := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(al, "en") {
		return "en"
	}
	return "ja"
}

func (m *oauthManager) handleConsentGet(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimSpace(r.URL.Query().Get("tx"))
	tx, peer, ok := m.loadVerifiedConsentTx(r, txID)
	if !ok {
		renderStatusPage(w, http.StatusUnauthorized, false, "セッション無効 / Session Invalid", "認証セッションが無効または期限切れです。Discord からやり直してください。", "")
		return
	}
	m.renderConsentForm(w, r, tx, peer, "")
}

func (m *oauthManager) renderConsentForm(w http.ResponseWriter, r *http.Request, tx pendingOAuthTx, peer PeerConfig, errMsg string) {
	lang := detectConsentLang(r)
	pm := m.srv.svc.Policy
	userCaps := m.srv.svc.ListAvailableCaps(r.Context(), tx.DiscordUserID, uuid.Nil)
	showCapsOption := peer.AllowCaps && slices.Contains(tx.RequestedScopes, bbs.FederationScopeCaps) && len(userCaps) > 0

	capNames := make([]string, 0, len(userCaps))
	for _, c := range userCaps {
		capNames = append(capNames, c.Name)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'")
	w.WriteHeader(http.StatusOK)
	_ = consentPageTmpl.Execute(w, map[string]any{
		"ActionURL":      m.srv.basePath + "/api/v1/bbs/federation/oauth/consent",
		"TxID":           tx.TxID,
		"CSRFToken":      tx.CSRFToken,
		"PeerName":       peer.Name,
		"PeerID":         peer.ID,
		"UserName":       tx.DiscordDisplayName + " (@" + tx.DiscordUsername + ")",
		"PolicyVersion":  pm.CurrentVersion,
		"TermsURL":       pm.TermsURL,
		"GuidelinesURL":  pm.GuidelinesURL,
		"PrivacyURL":     pm.PrivacyPolicyURL,
		"TermsDoc":       pm.GetDocument(bbs.PolicyDocTerms, lang),
		"GuidelinesDoc":  pm.GetDocument(bbs.PolicyDocGuidelines, lang),
		"PrivacyDoc":     pm.GetDocument(bbs.PolicyDocPrivacy, lang),
		"AllowWrite":     slices.Contains(tx.RequestedScopes, bbs.FederationScopeWrite),
		"AllowReport":    slices.Contains(tx.RequestedScopes, bbs.FederationScopeReport),
		"ShowCapsOption": showCapsOption,
		"CapNames":       strings.Join(capNames, ", "),
		"ErrorMessage":   errMsg,
	})
}

func (m *oauthManager) handleConsentPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderStatusPage(w, http.StatusBadRequest, false, "不正なリクエスト", "フォームを解析できませんでした。", "")
		return
	}
	txID := strings.TrimSpace(r.PostFormValue("tx"))
	tx, peer, ok := m.loadVerifiedConsentTx(r, txID)
	if !ok {
		renderStatusPage(w, http.StatusUnauthorized, false, "セッション無効 / Session Invalid", "認証セッションが無効または期限切れです。Discord からやり直してください。", "")
		return
	}
	csrfToken := r.PostFormValue("csrf_token")
	if subtle.ConstantTimeCompare([]byte(csrfToken), []byte(tx.CSRFToken)) != 1 {
		renderStatusPage(w, http.StatusForbidden, false, "CSRF エラー", "不正なフォーム送信です。", "")
		return
	}

	decision := strings.TrimSpace(r.PostFormValue("decision"))
	if decision == "deny" {
		m.mu.Lock()
		delete(m.txByID, tx.TxID)
		m.mu.Unlock()
		m.clearSessionCookie(w)
		redirectOAuthError(w, r, tx.RedirectURI, "access_denied", tx.ClientState)
		return
	}
	if decision != "approve" {
		m.renderConsentForm(w, r, tx, peer, "不正な操作です。")
		return
	}

	if r.PostFormValue("agree_terms") != "on" ||
		r.PostFormValue("agree_guidelines") != "on" ||
		r.PostFormValue("agree_privacy") != "on" ||
		r.PostFormValue("agree_delegate") != "on" {
		m.renderConsentForm(w, r, tx, peer, "続行するには、利用規約・ガイドライン・プライバシーポリシーおよび権限委譲の4項目すべてにチェックを入れてください。")
		return
	}

	posterHash := ""
	if m.srv.svc.Identity != nil {
		posterHash = m.srv.svc.Identity.GeneratePosterHash(bbs.ScopeGlobal, 0, tx.DiscordUserID)
	}
	now := time.Now().UTC()
	banned, err := m.srv.svc.Store.IsBanned(r.Context(), bbs.ScopeGlobal, 0, posterHash, tx.DiscordUserID, now)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "規制状態の確認に失敗しました。", "")
		return
	}
	if banned {
		m.mu.Lock()
		delete(m.txByID, tx.TxID)
		m.mu.Unlock()
		m.clearSessionCookie(w)
		renderStatusPage(w, http.StatusForbidden, false, "利用制限中 / Globally Banned", "この Discord アカウントは本番掲示板でグローバル規制中です。", "")
		return
	}

	verTok := bbs.PolicyVersionToken(m.srv.svc.Policy.CurrentVersion)
	consent, err := m.srv.svc.AcceptPolicyVersion(r.Context(), tx.DiscordUserID, verTok)
	if err != nil {
		slog.Warn("federation oauth failed to save user consent", "user_id", tx.DiscordUserID, "err", err)
		m.renderConsentForm(w, r, tx, peer, "規約同意の保存中にエラーが発生しました。もう一度お試しください。")
		return
	}

	var grantedScopes []string
	for _, sc := range tx.RequestedScopes {
		if sc == bbs.FederationScopeWrite || sc == bbs.FederationScopeReport {
			grantedScopes = append(grantedScopes, sc)
		}
	}
	if peer.AllowCaps && slices.Contains(tx.RequestedScopes, bbs.FederationScopeCaps) && r.PostFormValue("agree_caps") == "on" {
		if caps := m.srv.svc.ListAvailableCaps(r.Context(), tx.DiscordUserID, uuid.Nil); len(caps) > 0 {
			grantedScopes = append(grantedScopes, bbs.FederationScopeCaps)
		}
	}

	rawCode, err := GenerateRandomToken(32)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Internal Error", "認可コードの生成に失敗しました。", "")
		return
	}
	codeHash := HashToken(rawCode)

	m.mu.Lock()
	delete(m.txByID, tx.TxID)
	m.pruneLocked(now)
	m.codesByHash[codeHash] = &pendingOAuthCode{
		PeerID:        tx.PeerID,
		RedirectURI:   tx.RedirectURI,
		UserID:        tx.DiscordUserID,
		Scopes:        grantedScopes,
		PolicyVersion: consent.PolicyVersion,
		CodeChallenge: tx.CodeChallenge,
		ExpiresAt:     now.Add(pendingOAuthCodeTTL),
	}
	m.mu.Unlock()

	m.clearSessionCookie(w)

	cbURL, err := url.Parse(tx.RedirectURI)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "Redirect Error", "無効なリダイレクトURLです。", "")
		return
	}
	cq := cbURL.Query()
	cq.Set("code", rawCode)
	if tx.ClientState != "" {
		cq.Set("state", tx.ClientState)
	}
	cbURL.RawQuery = cq.Encode()
	http.Redirect(w, r, cbURL.String(), http.StatusFound)
}

func parseTokenParams(r *http.Request) (grantType, code, redirectURI, codeVerifier, refreshToken string, err error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") {
		var body struct {
			GrantType    string `json:"grant_type"`
			Code         string `json:"code"`
			RedirectURI  string `json:"redirect_uri"`
			CodeVerifier string `json:"code_verifier"`
			RefreshToken string `json:"refresh_token"`
		}
		if err = readJSONBody(r, &body); err != nil {
			return "", "", "", "", "", err
		}
		return strings.TrimSpace(body.GrantType), strings.TrimSpace(body.Code), strings.TrimSpace(body.RedirectURI), strings.TrimSpace(body.CodeVerifier), strings.TrimSpace(body.RefreshToken), nil
	}
	if err = r.ParseForm(); err != nil {
		return "", "", "", "", "", err
	}
	return strings.TrimSpace(r.PostFormValue("grant_type")),
		strings.TrimSpace(r.PostFormValue("code")),
		strings.TrimSpace(r.PostFormValue("redirect_uri")),
		strings.TrimSpace(r.PostFormValue("code_verifier")),
		strings.TrimSpace(r.PostFormValue("refresh_token")),
		nil
}

func (m *oauthManager) handleToken(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	grantType, code, redirectURI, codeVerifier, refreshToken, err := parseTokenParams(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid token request body")
		return
	}

	switch grantType {
	case "authorization_code":
		m.handleAuthCodeExchange(w, r, peer, code, redirectURI, codeVerifier)
	case "refresh_token":
		m.handleRefreshExchange(w, r, peer, refreshToken)
	default:
		writeAPIError(w, http.StatusBadRequest, CodeInvalidGrant, "unsupported grant_type")
	}
}

func (m *oauthManager) handleAuthCodeExchange(w http.ResponseWriter, r *http.Request, peer PeerConfig, rawCode, redirectURI, codeVerifier string) {
	if !peer.AllowWrite && !peer.EffectiveAllowReport() {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "peer is read-only")
		return
	}
	if rawCode == "" || redirectURI == "" || codeVerifier == "" {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidGrant, "code, redirect_uri, and code_verifier are required")
		return
	}
	now := time.Now().UTC()
	codeHash := HashToken(rawCode)

	m.mu.Lock()
	m.pruneLocked(now)
	pending, ok := m.codesByHash[codeHash]
	if ok {
		// Strict single-use authorization code deletion.
		delete(m.codesByHash, codeHash)
	}
	m.mu.Unlock()

	if !ok || now.After(pending.ExpiresAt) || pending.PeerID != peer.ID || pending.RedirectURI != redirectURI {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidGrant, "invalid or expired authorization code")
		return
	}
	computed := ComputePKCES256(codeVerifier)
	if subtle.ConstantTimeCompare([]byte(computed), []byte(pending.CodeChallenge)) != 1 {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidGrant, "PKCE code_verifier mismatch")
		return
	}

	if _, cErr := m.srv.svc.CheckConsent(r.Context(), pending.UserID); cErr != nil || pending.PolicyVersion != m.srv.svc.Policy.CurrentVersion {
		writeAPIError(w, http.StatusForbidden, CodePolicyConsentRequired, "policy version changed or consent withdrawn")
		return
	}
	posterHash := ""
	if m.srv.svc.Identity != nil {
		posterHash = m.srv.svc.Identity.GeneratePosterHash(bbs.ScopeGlobal, 0, pending.UserID)
	}
	banned, err := m.srv.svc.Store.IsBanned(r.Context(), bbs.ScopeGlobal, 0, posterHash, pending.UserID, now)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "ban check failed")
		return
	}
	if banned {
		writeAPIError(w, http.StatusForbidden, CodeBanned, "user is globally banned on origin")
		return
	}

	accessTok, err := GenerateRandomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "token generation failed")
		return
	}
	refreshTok, err := GenerateRandomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "token generation failed")
		return
	}

	accessExp := now.Add(m.srv.cfg.AccessTTL)
	refreshExp := now.Add(m.srv.cfg.RefreshTTL)
	grant := &bbs.FederationGrant{
		PeerID:          peer.ID,
		UserID:          pending.UserID,
		Scopes:          pending.Scopes,
		PolicyVersion:   pending.PolicyVersion,
		AccessTokenHash: HashToken(accessTok),
		AccessExpiresAt: accessExp,
	}
	rt := &bbs.FederationRefreshToken{
		TokenHash: HashToken(refreshTok),
		ExpiresAt: refreshExp,
	}
	if err := m.srv.svc.Store.CreateFederationGrantWithRefresh(r.Context(), grant, rt); err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "failed to persist delegation grant")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		TokenType:        "Bearer",
		AccessToken:      accessTok,
		ExpiresIn:        int64(m.srv.cfg.AccessTTL.Seconds()),
		RefreshToken:     refreshTok,
		RefreshExpiresIn: int64(m.srv.cfg.RefreshTTL.Seconds()),
		UserID:           grant.UserID,
		Scopes:           grant.Scopes,
		PolicyVersion:    grant.PolicyVersion,
	})
}

func (m *oauthManager) handleRefreshExchange(w http.ResponseWriter, r *http.Request, peer PeerConfig, oldRefreshTok string) {
	if !peer.AllowWrite && !peer.EffectiveAllowReport() {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "peer is read-only")
		return
	}
	if oldRefreshTok == "" {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidGrant, "refresh_token is required")
		return
	}
	newAccess, err := GenerateRandomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "token generation failed")
		return
	}
	newRefresh, err := GenerateRandomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "token generation failed")
		return
	}

	now := time.Now().UTC()
	accessExp := now.Add(m.srv.cfg.AccessTTL)
	refreshExp := now.Add(m.srv.cfg.RefreshTTL)

	grant, reused, err := m.srv.svc.Store.RotateFederationRefreshToken(
		r.Context(),
		peer.ID,
		HashToken(oldRefreshTok),
		HashToken(newAccess),
		accessExp,
		&bbs.FederationRefreshToken{
			TokenHash: HashToken(newRefresh),
			ExpiresAt: refreshExp,
		},
		now,
	)
	if err != nil {
		if reused {
			slog.Warn("federation security: refresh token reuse detected; grant family revoked", "peer_id", peer.ID)
			writeAPIError(w, http.StatusUnauthorized, CodeInvalidGrant, "refresh token reuse detected; delegation revoked")
			return
		}
		writeAPIError(w, http.StatusUnauthorized, CodeInvalidGrant, "invalid or expired refresh token")
		return
	}

	if _, cErr := m.srv.svc.CheckConsent(r.Context(), grant.UserID); cErr != nil || grant.PolicyVersion != m.srv.svc.Policy.CurrentVersion {
		_ = m.srv.svc.Store.RevokeFederationGrant(r.Context(), grant.ID, grant.UserID, now)
		writeAPIError(w, http.StatusForbidden, CodePolicyConsentRequired, "policy version updated or consent withdrawn; re-authorization required")
		return
	}
	posterHash := ""
	if m.srv.svc.Identity != nil {
		posterHash = m.srv.svc.Identity.GeneratePosterHash(bbs.ScopeGlobal, 0, grant.UserID)
	}
	banned, bErr := m.srv.svc.Store.IsBanned(r.Context(), bbs.ScopeGlobal, 0, posterHash, grant.UserID, now)
	if bErr != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "ban check failed")
		return
	}
	if banned {
		_ = m.srv.svc.Store.RevokeFederationGrant(r.Context(), grant.ID, grant.UserID, now)
		writeAPIError(w, http.StatusForbidden, CodeBanned, "user is globally banned on origin")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		TokenType:        "Bearer",
		AccessToken:      newAccess,
		ExpiresIn:        int64(m.srv.cfg.AccessTTL.Seconds()),
		RefreshToken:     newRefresh,
		RefreshExpiresIn: int64(m.srv.cfg.RefreshTTL.Seconds()),
		UserID:           grant.UserID,
		Scopes:           grant.Scopes,
		PolicyVersion:    grant.PolicyVersion,
	})
}

func (m *oauthManager) handleRevoke(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	var req RevokeRequest
	if err := readJSONBody(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid request body")
		return
	}
	now := time.Now().UTC()
	if strings.TrimSpace(req.Token) != "" {
		if g, err := m.srv.svc.Store.GetFederationGrantByAccessHash(r.Context(), HashToken(strings.TrimSpace(req.Token))); err == nil && g.PeerID == peer.ID {
			if rErr := m.srv.svc.Store.RevokeFederationGrant(r.Context(), g.ID, g.UserID, now); rErr != nil {
				writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "failed to revoke grant")
				return
			}
		}
	} else if req.UserID != 0 {
		if err := m.srv.svc.Store.RevokeFederationGrantsByPeerAndUser(r.Context(), peer.ID, req.UserID, now); err != nil {
			writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "failed to revoke grants")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type discordOAuthTokenResp struct {
	AccessToken string `json:"access_token"`
}

type discordOAuthUserResp struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
}

func (m *oauthManager) exchangeDiscordCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", m.srv.cfg.OAuth.ClientID)
	form.Set("client_secret", m.srv.cfg.OAuth.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", m.discordCallbackURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.discordAPIBase+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed discordOAuthTokenResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", errors.New("empty access_token in discord oauth response")
	}
	return parsed.AccessToken, nil
}

func (m *oauthManager) fetchDiscordUser(ctx context.Context, accessToken string) (discordOAuthUserResp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.discordAPIBase+"/users/@me", nil)
	if err != nil {
		return discordOAuthUserResp{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return discordOAuthUserResp{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return discordOAuthUserResp{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return discordOAuthUserResp{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var user discordOAuthUserResp
	if err := json.Unmarshal(body, &user); err != nil {
		return discordOAuthUserResp{}, err
	}
	return user, nil
}

var statusPageTmpl = template.Must(template.New("fed_status").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; padding: 16px; }
    .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 28px 32px; max-width: 520px; width: 100%; box-shadow: 0 12px 30px rgba(0,0,0,0.45); }
    h1 { margin: 0 0 12px; font-size: 1.25rem; color: {{if .Success}}#34d399{{else}}#f87171{{end}}; }
    p { margin: 0 0 10px; line-height: 1.6; color: #cbd5e1; }
    code { background: #0f172a; padding: 2px 6px; border-radius: 4px; color: #93c5fd; word-break: break-all; }
  </style>
</head>
<body>
  <div class="card">
    <h1>{{.Heading}}</h1>
    <p>{{.Message}}</p>
    {{if .Detail}}<p><code>{{.Detail}}</code></p>{{end}}
  </div>
</body>
</html>`))

var consentPageTmpl = template.Must(template.New("fed_consent").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>BBS 外部連携・規約同意</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #f8fafc; margin: 0; padding: 24px 16px; display: flex; justify-content: center; }
    .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 28px; max-width: 680px; width: 100%; box-shadow: 0 12px 30px rgba(0,0,0,0.45); }
    h1 { margin: 0 0 10px; font-size: 1.35rem; color: #38bdf8; }
    .meta { background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 12px 16px; margin: 14px 0 18px; font-size: 0.95rem; line-height: 1.6; }
    .err { background: #7f1d1d; border: 1px solid #ef4444; color: #fecaca; padding: 10px 14px; border-radius: 8px; margin-bottom: 16px; }
    .scopes { margin: 12px 0; padding-left: 20px; color: #cbd5e1; }
    details { background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 10px 14px; margin-bottom: 10px; }
    summary { cursor: pointer; font-weight: 600; color: #e2e8f0; }
    pre { white-space: pre-wrap; font-family: inherit; font-size: 0.88rem; color: #cbd5e1; line-height: 1.55; margin-top: 10px; }
    .links { margin: 8px 0 14px; font-size: 0.9rem; }
    .links a { color: #38bdf8; margin-right: 12px; }
    .check { display: flex; align-items: flex-start; gap: 10px; margin: 10px 0; cursor: pointer; line-height: 1.5; }
    .check input { margin-top: 4px; width: 17px; height: 17px; accent-color: #38bdf8; }
    .caps-box { background: #172554; border: 1px solid #2563eb; border-radius: 8px; padding: 10px 14px; margin: 14px 0; }
    .actions { display: flex; gap: 12px; margin-top: 22px; flex-wrap: wrap; }
    button { border: 0; border-radius: 8px; padding: 11px 20px; font-size: 0.95rem; font-weight: 600; cursor: pointer; }
    .btn-primary { background: #2563eb; color: #fff; }
    .btn-primary:hover { background: #1d4ed8; }
    .btn-deny { background: #334155; color: #e2e8f0; }
    .btn-deny:hover { background: #475569; }
  </style>
</head>
<body>
  <div class="card">
    <h1>BBS 外部インスタンス連携の確認</h1>
    <p>以下の外部インスタンスに対して、あなたとしてのグローバル板への操作権限を委譲します。</p>
    <div class="meta">
      <div><strong>認証済み Discord アカウント:</strong> {{.UserName}}</div>
      <div><strong>委譲先インスタンス:</strong> {{.PeerName}} (<code>{{.PeerID}}</code>)</div>
      <div><strong>本番ポリシーバージョン:</strong> <code>{{.PolicyVersion}}</code></div>
      <div><strong>要求される権限:</strong>
        <ul class="scopes">
          {{if .AllowWrite}}<li>グローバル板でのスレッド作成およびレス投稿</li>{{end}}
          {{if .AllowReport}}<li>グローバル板の投稿に対する通報の送信</li>{{end}}
        </ul>
      </div>
    </div>

    {{if .ErrorMessage}}<div class="err">{{.ErrorMessage}}</div>{{end}}

    {{if or .TermsURL .GuidelinesURL .PrivacyURL}}
    <div class="links">
      外部参照リンク:
      {{if .TermsURL}}<a href="{{.TermsURL}}" target="_blank" rel="noopener noreferrer">利用規約</a>{{end}}
      {{if .GuidelinesURL}}<a href="{{.GuidelinesURL}}" target="_blank" rel="noopener noreferrer">ガイドライン</a>{{end}}
      {{if .PrivacyURL}}<a href="{{.PrivacyURL}}" target="_blank" rel="noopener noreferrer">プライバシーポリシー</a>{{end}}
    </div>
    {{end}}

    <details><summary>利用規約（クリックして開く）</summary><pre>{{.TermsDoc}}</pre></details>
    <details><summary>コミュニティガイドライン（クリックして開く）</summary><pre>{{.GuidelinesDoc}}</pre></details>
    <details><summary>プライバシーポリシー（クリックして開く）</summary><pre>{{.PrivacyDoc}}</pre></details>

    <form method="post" action="{{.ActionURL}}">
      <input type="hidden" name="tx" value="{{.TxID}}" />
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />

      <label class="check"><input type="checkbox" name="agree_terms" value="on" /> <span>本番インスタンスの利用規約に同意します（必須）</span></label>
      <label class="check"><input type="checkbox" name="agree_guidelines" value="on" /> <span>本番インスタンスのコミュニティガイドラインに同意します（必須）</span></label>
      <label class="check"><input type="checkbox" name="agree_privacy" value="on" /> <span>本番インスタンスのプライバシーポリシーに同意します（必須）</span></label>
      <label class="check"><input type="checkbox" name="agree_delegate" value="on" /> <span>外部インスタンス <strong>{{.PeerName}}</strong> が私に代わってグローバル板へ投稿・通報することを許可します（必須・/bbs policy からいつでも解除可能）</span></label>

      {{if .ShowCapsOption}}
      <div class="caps-box">
        <label class="check"><input type="checkbox" name="agree_caps" value="on" /> <span><strong>[任意]</strong> 本番で所持しているキャップ（★: {{.CapNames}}）の利用もこのインスタンスに許可する</span></label>
      </div>
      {{end}}

      <div class="actions">
        <button type="submit" name="decision" value="approve" class="btn-primary">同意して許可する</button>
        <button type="submit" name="decision" value="deny" class="btn-deny">拒否する</button>
      </div>
    </form>
  </div>
</body>
</html>`))

func renderStatusPage(w http.ResponseWriter, statusCode int, success bool, heading, message, detail string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.WriteHeader(statusCode)
	_ = statusPageTmpl.Execute(w, map[string]any{
		"Title":   heading,
		"Success": success,
		"Heading": heading,
		"Message": message,
		"Detail":  detail,
	})
}
