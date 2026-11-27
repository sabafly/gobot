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

package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

const loginPollInterval = 2 * time.Second

type loginState int

const (
	loginStateIdle loginState = iota
	loginStateStarting
	loginStatePolling
	loginStateError
)

type loginStartedMsg struct {
	Gen  uint64
	Resp admin.AuthStartResponse
	Err  error
}

type loginBrowserMsg struct {
	Gen uint64
	Err error
}

type loginPollTickMsg struct {
	Gen uint64
}

type loginPollResultMsg struct {
	Gen  uint64
	Resp admin.AuthPollResponse
	Err  error
}

// LoginSuccessMsg notifies the root App when Discord OAuth2 completes and a session token is issued.
type LoginSuccessMsg struct {
	Principal admin.PrincipalInfo
	Token     string
	ExpiresAt *time.Time
}

// LoginModel drives the interactive Discord OAuth2 handshake inside the TUI.
type LoginModel struct {
	theme         ui.Theme
	client        *adminclient.Client
	store         *adminclient.FileSessionStore
	baseURL       string
	noBrowser     bool
	pollInterval  time.Duration
	spin          spinner.Model
	gen           uint64
	state         loginState
	authorizeURL  string
	loginID       string
	pollSecret    string
	expiresAt     time.Time
	notice        string
	errMsg        string
	browserNote   string
}

// NewLoginModel constructs an OAuth2 login screen targeting baseURL.
func NewLoginModel(theme ui.Theme, client *adminclient.Client, store *adminclient.FileSessionStore, baseURL string, noBrowser bool) LoginModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.CyanColor)),
	)
	return LoginModel{
		theme:        theme,
		client:       client,
		store:        store,
		baseURL:      baseURL,
		noBrowser:    noBrowser,
		pollInterval: loginPollInterval,
		spin:         sp,
		state:        loginStateIdle,
	}
}

// Start initiates a new Discord OAuth2 handshake and opens the browser if enabled.
func (m LoginModel) Start() (LoginModel, tea.Cmd) {
	m.gen++
	m.state = loginStateStarting
	m.errMsg = ""
	m.authorizeURL = ""
	m.loginID = ""
	m.pollSecret = ""
	m.browserNote = ""
	gen := m.gen
	client := m.client
	return m, tea.Batch(
		m.spin.Tick,
		func() tea.Msg {
			if client == nil {
				return loginStartedMsg{Gen: gen, Err: fmt.Errorf("admin client not configured")}
			}
			reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := client.StartDiscordLogin(reqCtx, tuiClientInfo())
			return loginStartedMsg{Gen: gen, Resp: resp, Err: err}
		},
	)
}

func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.state == loginStateStarting || m.state == loginStatePolling {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case loginStartedMsg:
		if msg.Gen != m.gen {
			return m, nil
		}
		if msg.Err != nil {
			m.state = loginStateError
			m.errMsg = msg.Err.Error()
			return m, nil
		}
		m.state = loginStatePolling
		m.authorizeURL = msg.Resp.AuthorizeURL
		m.loginID = msg.Resp.LoginID
		m.pollSecret = msg.Resp.PollSecret
		m.expiresAt = msg.Resp.ExpiresAt

		cmds := []tea.Cmd{
			m.spin.Tick,
			m.schedulePoll(msg.Gen),
		}
		if m.noBrowser {
			m.browserNote = "Browser auto-open disabled (--no-browser). Open the URL below manually."
		} else {
			authURL := m.authorizeURL
			gen := msg.Gen
			cmds = append(cmds, func() tea.Msg {
				return loginBrowserMsg{Gen: gen, Err: openBrowserFn(authURL)}
			})
		}
		return m, tea.Batch(cmds...)

	case loginBrowserMsg:
		if msg.Gen != m.gen {
			return m, nil
		}
		if msg.Err != nil {
			m.browserNote = "Could not launch browser automatically (" + msg.Err.Error() + "). Open the URL manually."
		} else {
			m.browserNote = "Browser opened. Complete Discord authorization in your browser."
		}
		return m, nil

	case loginPollTickMsg:
		if msg.Gen != m.gen || m.state != loginStatePolling {
			return m, nil
		}
		if !m.expiresAt.IsZero() && time.Now().After(m.expiresAt) {
			m.state = loginStateError
			m.errMsg = "Login request expired. Press Enter to try again."
			return m, nil
		}
		gen := msg.Gen
		client := m.client
		loginID := m.loginID
		secret := m.pollSecret
		return m, func() tea.Msg {
			reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := client.PollDiscordLogin(reqCtx, loginID, secret)
			return loginPollResultMsg{Gen: gen, Resp: resp, Err: err}
		}

	case loginPollResultMsg:
		if msg.Gen != m.gen || m.state != loginStatePolling {
			return m, nil
		}
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
			if strings.Contains(strings.ToLower(msg.Err.Error()), "404") || strings.Contains(strings.ToLower(msg.Err.Error()), "expired") {
				m.state = loginStateError
				return m, nil
			}
			return m, m.schedulePoll(msg.Gen)
		}

		switch msg.Resp.Status {
		case "pending":
			m.errMsg = ""
			return m, m.schedulePoll(msg.Gen)
		case "denied", "expired":
			m.state = loginStateError
			if msg.Resp.Error != "" {
				m.errMsg = msg.Resp.Error
			} else {
				m.errMsg = "Discord login " + msg.Resp.Status
			}
			return m, nil
		case "approved":
			if m.client != nil {
				m.client.Token = msg.Resp.SessionToken
			}
			var principal admin.PrincipalInfo
			if msg.Resp.Principal != nil {
				principal = *msg.Resp.Principal
			}
			if m.store != nil {
				exp := time.Now().UTC().Add(7 * 24 * time.Hour)
				if msg.Resp.ExpiresAt != nil {
					exp = *msg.Resp.ExpiresAt
				}
				_ = m.store.Save(m.baseURL, adminclient.StoredSession{
					BaseURL:     m.baseURL,
					Token:       msg.Resp.SessionToken,
					UserID:      principal.UserID,
					Username:    principal.Username,
					DisplayName: principal.DisplayName,
					Role:        principal.Role,
					ExpiresAt:   exp,
					SavedAt:     time.Now().UTC(),
				})
			}
			m.state = loginStateIdle
			m.notice = ""
			m.errMsg = ""
			return m, func() tea.Msg {
				return LoginSuccessMsg{
					Principal: principal,
					Token:     msg.Resp.SessionToken,
					ExpiresAt: msg.Resp.ExpiresAt,
				}
			}
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter", "r":
			return m.Start()
		case "o":
			if m.authorizeURL != "" {
				authURL := m.authorizeURL
				gen := m.gen
				return m, func() tea.Msg {
					return loginBrowserMsg{Gen: gen, Err: openBrowserFn(authURL)}
				}
			}
		}
	}
	return m, nil
}

func (m LoginModel) View(width, height int) string {
	boxW := max(min(width-4, 78), 44)
	lines := []string{
		m.theme.CardTitle.Render("gobot Admin Dashboard — Discord OAuth2 Login"),
		m.theme.MutedText.Render("Server: " + m.baseURL),
	}
	if m.notice != "" {
		lines = append(lines, "", m.theme.BadgeWarn.Render(m.notice))
	}

	switch m.state {
	case loginStateIdle:
		lines = append(lines,
			"",
			m.theme.Value.Render("Authenticate with your Discord account to obtain an operator session."),
			m.theme.MutedText.Render("Press [Enter] to start Discord OAuth2 login."),
		)
	case loginStateStarting:
		lines = append(lines,
			"",
			m.spin.View()+" Contacting admin server...",
		)
	case loginStatePolling:
		lines = append(lines,
			"",
			m.spin.View()+" Waiting for Discord authorization in browser...",
		)
		if m.browserNote != "" {
			lines = append(lines, m.theme.MutedText.Render(m.browserNote))
		}
		if m.authorizeURL != "" {
			lines = append(lines,
				"",
				m.theme.Label.Render("Authorize URL (click or copy):"),
				m.theme.BadgeInfo.Render(m.authorizeURL),
			)
		}
		if !m.expiresAt.IsZero() {
			rem := max(int(time.Until(m.expiresAt).Seconds()), 0)
			lines = append(lines, m.theme.MutedText.Render(fmt.Sprintf("Expires in %ds", rem)))
		}
		if m.errMsg != "" {
			lines = append(lines, m.theme.BadgeWarn.Render("Poll warning: "+m.errMsg))
		}
	case loginStateError:
		lines = append(lines,
			"",
			m.theme.BadgeErr.Render("Login failed: "+m.errMsg),
			m.theme.MutedText.Render("Press [Enter] or [r] to retry login."),
		)
	}

	lines = append(lines, "", m.theme.MutedText.Render("[Enter/r] Start/Retry   [o] Open Browser   [q] Quit"))
	card := m.theme.ModalBox.Width(boxW).Render(strings.Join(lines, "\n"))
	if width > 0 && height > 0 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
	}
	return card
}

func (m LoginModel) schedulePoll(gen uint64) tea.Cmd {
	d := m.pollInterval
	if d <= 0 {
		d = loginPollInterval
	}
	return tea.Tick(d, func(time.Time) tea.Msg {
		return loginPollTickMsg{Gen: gen}
	})
}

func tuiClientInfo() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "gobot-tui"
	}
	return "gobot-tui@" + host
}
