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

package panels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

type opsSubTab int

const (
	subTabOps opsSubTab = iota
	subTabAudit
	subTabSessions
)

const (
	auditFilterPrompt    = "Action: "
	auditPageLimit       = 25
	opsConfirmActionTag  = "ops.exec"
	opsConfirmRevokeTag  = "sessions.revoke"
)

type opActionDef struct {
	ID          string
	Title       string
	Description string
	Danger      bool
}

var defaultOpsActions = []opActionDef{
	{
		ID:          "reload-locales",
		Title:       "Reload i18n Locales",
		Description: "Re-reads locale YAML files from disk immediately without restarting the bot.",
	},
	{
		ID:          "sync-commands",
		Title:       "Sync Discord Slash Commands",
		Description: "Pushes current application command definitions globally and to private guilds.",
	},
	{
		ID:          "bbs-dat-check",
		Title:       "Run BBS Dat-Ochi Check",
		Description: "Archives stale inactive BBS threads and locks/archives their Discord mirrors.",
		Danger:      true,
	},
}

type opsExecutedMsg struct {
	ActionID string
	Result   string
	Err      error
}

type auditLoadedMsg struct {
	Resp admin.AuditListResponse
	Err  error
}

type sessionsLoadedMsg struct {
	Resp admin.SessionsResponse
	Err  error
}

type sessionRevokedMsg struct {
	SessionID string
	Current   bool
	Err       error
}

// OpsAuditPanel bundles Operations execution, Management Audit Log browsing, and Session revocation.
type OpsAuditPanel struct {
	theme        ui.Theme
	subTab       opsSubTab
	canExecOps   bool
	canReadAudit bool
	opsIdx       int
	runningOp    bool
	lastOpRes    string

	auditInput     textinput.Model
	filteringAudit bool
	appliedAction  string
	auditOffset    int
	auditEntries   []admin.AuditLogEntry
	auditTotal     int64
	auditIdx       int
	loadingAudit   bool
	auditErr       string
	auditVP        viewport.Model

	sessions        []admin.SessionInfo
	sessionIdx      int
	loadingSessions bool
	sessionsErr     string

	confirmModal ui.ConfirmModal
	width        int
	height       int
}

// NewOpsAuditPanel constructs the Ops, Audit Logs, and Sessions panel.
func NewOpsAuditPanel(theme ui.Theme) *OpsAuditPanel {
	vp := viewport.New(viewport.WithWidth(44), viewport.WithHeight(16))
	vp.SoftWrap = false
	return &OpsAuditPanel{
		theme:      theme,
		auditInput: ui.NewRealCursorInput("action filter (e.g. guild.update)", 32),
		auditVP:    vp,
	}
}

func (p *OpsAuditPanel) ID() string                           { return "ops" }
func (p *OpsAuditPanel) Title() string                        { return "Ops & Audit" }
func (p *OpsAuditPanel) RequiredPermission() admin.Permission { return "" }
func (p *OpsAuditPanel) Close()                               {}

func (p *OpsAuditPanel) CapturingInput() bool {
	return p.filteringAudit || p.confirmModal.Active
}

func (p *OpsAuditPanel) Cursor() *tea.Cursor {
	if p.subTab != subTabAudit || !p.filteringAudit || p.confirmModal.Active || !p.auditInput.Focused() {
		return nil
	}
	c := p.auditInput.Cursor()
	if c == nil {
		return nil
	}
	cur := *c
	cur.X += lipgloss.Width(auditFilterPrompt)
	cur.Y += 1
	return &cur
}

func (p *OpsAuditPanel) ShortHelp(ctx ui.PanelContext) []key.Binding {
	base := []key.Binding{
		key.NewBinding(key.WithKeys("[", "]"), key.WithHelp("[/]", "subtab")),
	}
	switch p.subTab {
	case subTabOps:
		if ctx.HasPerm(admin.PermOpsExecute) {
			base = append(base,
				key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "execute")),
			)
		}
	case subTabAudit:
		if ctx.HasPerm(admin.PermAuditRead) {
			base = append(base,
				key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
				key.NewBinding(key.WithKeys("n", "p"), key.WithHelp("n/p", "page")),
				key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
			)
		}
	case subTabSessions:
		base = append(base,
			key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select")),
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "revoke")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		)
	}
	return base
}

func (p *OpsAuditPanel) Init(ctx ui.PanelContext) tea.Cmd {
	p.canExecOps = ctx.HasPerm(admin.PermOpsExecute)
	p.canReadAudit = ctx.HasPerm(admin.PermAuditRead)
	switch {
	case p.canExecOps:
		p.subTab = subTabOps
	case p.canReadAudit:
		p.subTab = subTabAudit
	default:
		p.subTab = subTabSessions
	}

	var cmds []tea.Cmd
	if p.canReadAudit {
		p.loadingAudit = true
		cmds = append(cmds, p.fetchAuditCmd(ctx.Client))
	}
	p.loadingSessions = true
	cmds = append(cmds, p.fetchSessionsCmd(ctx.Client))
	return tea.Batch(cmds...)
}

func (p *OpsAuditPanel) Update(msg tea.Msg, ctx ui.PanelContext) (ui.Panel, tea.Cmd) {
	p.canExecOps = ctx.HasPerm(admin.PermOpsExecute)
	p.canReadAudit = ctx.HasPerm(admin.PermAuditRead)
	if p.confirmModal.Active {
		confirmed, canceled, tag, payload := p.confirmModal.Update(msg)
		if canceled {
			return p, nil
		}
		if confirmed {
			switch tag {
			case opsConfirmActionTag:
				if act, ok := payload.(opActionDef); ok {
					p.runningOp = true
					return p, p.execOpCmd(ctx.Client, act.ID)
				}
			case opsConfirmRevokeTag:
				if sess, ok := payload.(admin.SessionInfo); ok {
					return p, p.revokeSessionCmd(ctx.Client, sess)
				}
			}
		}
		return p, nil
	}

	switch m := msg.(type) {
	case opsExecutedMsg:
		p.runningOp = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.lastOpRes = "Error: " + m.Err.Error()
			return p, ui.ShowToastCmd(ui.ToastError, "Operation failed: "+m.Err.Error())
		}
		p.lastOpRes = m.Result
		return p, ui.ShowToastCmd(ui.ToastSuccess, m.Result)

	case auditLoadedMsg:
		p.loadingAudit = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.auditErr = m.Err.Error()
			p.rebuildAuditDetail()
			return p, nil
		}
		p.auditErr = ""
		p.auditEntries = m.Resp.Entries
		p.auditTotal = m.Resp.Total
		if p.auditIdx >= len(p.auditEntries) {
			p.auditIdx = max(len(p.auditEntries)-1, 0)
		}
		p.rebuildAuditDetail()
		return p, nil

	case sessionsLoadedMsg:
		p.loadingSessions = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.sessionsErr = m.Err.Error()
			return p, nil
		}
		p.sessionsErr = ""
		p.sessions = m.Resp.Sessions
		if p.sessionIdx >= len(p.sessions) {
			p.sessionIdx = max(len(p.sessions)-1, 0)
		}
		return p, nil

	case sessionRevokedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			return p, ui.ShowToastCmd(ui.ToastError, "Revoke failed: "+m.Err.Error())
		}
		if m.Current {
			return p, tea.Batch(
				ui.ShowToastCmd(ui.ToastInfo, "Current session revoked; logged out"),
				func() tea.Msg { return ui.UnauthorizedMsg{} },
			)
		}
		p.loadingSessions = true
		return p, tea.Batch(
			p.fetchSessionsCmd(ctx.Client),
			ui.ShowToastCmd(ui.ToastSuccess, "Session revoked"),
		)

	case tea.KeyPressMsg:
		if p.subTab == subTabAudit && p.filteringAudit {
			switch m.String() {
			case "esc":
				p.filteringAudit = false
				p.auditInput.Blur()
				return p, nil
			case "enter":
				p.filteringAudit = false
				p.auditInput.Blur()
				p.appliedAction = strings.TrimSpace(p.auditInput.Value())
				p.auditOffset = 0
				p.auditIdx = 0
				p.loadingAudit = true
				return p, p.fetchAuditCmd(ctx.Client)
			}
			var cmd tea.Cmd
			p.auditInput, cmd = p.auditInput.Update(msg)
			return p, cmd
		}

		switch m.String() {
		case "[", "h", "left":
			p.subTab = (p.subTab + 2) % 3
			return p, p.onSubTabActivated(ctx)
		case "]", "l", "right":
			p.subTab = (p.subTab + 1) % 3
			return p, p.onSubTabActivated(ctx)
		}

		switch p.subTab {
		case subTabOps:
			if !ctx.HasPerm(admin.PermOpsExecute) {
				return p, nil
			}
			switch m.String() {
			case "j", "down":
				if p.opsIdx < len(defaultOpsActions)-1 {
					p.opsIdx++
				}
			case "k", "up":
				if p.opsIdx > 0 {
					p.opsIdx--
				}
			case "enter":
				act := defaultOpsActions[p.opsIdx]
				p.confirmModal.Open(opsConfirmActionTag, act.Title, act.Description, "Execute", act.Danger, act)
			}
			return p, nil

		case subTabAudit:
			if !ctx.HasPerm(admin.PermAuditRead) {
				return p, nil
			}
			switch m.String() {
			case "/":
				p.filteringAudit = true
				return p, p.auditInput.Focus()
			case "c":
				if p.appliedAction != "" || p.auditInput.Value() != "" {
					p.appliedAction = ""
					p.auditInput.SetValue("")
					p.auditOffset = 0
					p.auditIdx = 0
					p.loadingAudit = true
					return p, p.fetchAuditCmd(ctx.Client)
				}
			case "r":
				p.loadingAudit = true
				return p, p.fetchAuditCmd(ctx.Client)
			case "n":
				if int64(p.auditOffset+auditPageLimit) < p.auditTotal {
					p.auditOffset += auditPageLimit
					p.auditIdx = 0
					p.loadingAudit = true
					return p, p.fetchAuditCmd(ctx.Client)
				}
			case "p":
				if p.auditOffset > 0 {
					p.auditOffset = max(p.auditOffset-auditPageLimit, 0)
					p.auditIdx = 0
					p.loadingAudit = true
					return p, p.fetchAuditCmd(ctx.Client)
				}
			case "j", "down":
				if p.auditIdx < len(p.auditEntries)-1 {
					p.auditIdx++
					p.rebuildAuditDetail()
				}
			case "k", "up":
				if p.auditIdx > 0 {
					p.auditIdx--
					p.rebuildAuditDetail()
				}
			}
			var cmd tea.Cmd
			p.auditVP, cmd = p.auditVP.Update(msg)
			return p, cmd

		case subTabSessions:
			switch m.String() {
			case "r":
				p.loadingSessions = true
				return p, p.fetchSessionsCmd(ctx.Client)
			case "j", "down":
				if p.sessionIdx < len(p.sessions)-1 {
					p.sessionIdx++
				}
			case "k", "up":
				if p.sessionIdx > 0 {
					p.sessionIdx--
				}
			case "x", "delete":
				if len(p.sessions) > 0 && p.sessionIdx >= 0 && p.sessionIdx < len(p.sessions) {
					sess := p.sessions[p.sessionIdx]
					warn := ""
					if sess.Current {
						warn = "\nThis is YOUR current session — you will be logged out."
					}
					p.confirmModal.Open(
						opsConfirmRevokeTag,
						"Revoke Operator Session",
						fmt.Sprintf("User: @%s (%s)\nSession ID: %s%s", sess.Username, sess.Role, sess.ID, warn),
						"Revoke",
						true,
						sess,
					)
				}
			}
			return p, nil
		}
	}

	if p.subTab == subTabAudit {
		var cmd tea.Cmd
		p.auditVP, cmd = p.auditVP.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p *OpsAuditPanel) View(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	if p.width != width || p.height != height {
		p.width = width
		p.height = height
		p.auditInput.SetWidth(max(width-lipgloss.Width(auditFilterPrompt)-24, 16))
		auditViewH := max(height-2, 4)
		bodyH := max(auditViewH-2, 4)
		detailW := width - 2
		listInnerH := max((bodyH/2)-3, 3)
		detailInnerH := max(bodyH-listInnerH-6, 4)
		if width >= 88 {
			detailW = max(width-max(width*48/100, 34)-1, 30)
			detailInnerH = max(bodyH-3, 4)
		}
		p.auditVP.SetWidth(max(detailW-4, 20))
		p.auditVP.SetHeight(max(detailInnerH-1, 1))
		p.rebuildAuditDetail()
	}

	if p.confirmModal.Active {
		return p.confirmModal.View(width, height, p.theme)
	}

	tabs := []struct {
		id    opsSubTab
		label string
	}{
		{subTabOps, "Ops"},
		{subTabAudit, "Audit Logs"},
		{subTabSessions, "Sessions"},
	}
	var tabParts []string
	for _, tb := range tabs {
		st := p.theme.SubTab
		if p.subTab == tb.id {
			st = p.theme.SubTabActive
		}
		tabParts = append(tabParts, st.Render(tb.label))
	}
	topBar := ui.TruncateWidth(
		lipgloss.JoinHorizontal(lipgloss.Center, tabParts...) + "   " + p.theme.MutedText.Render("[/] subtabs"),
		width,
	)

	var body string
	switch p.subTab {
	case subTabOps:
		body = p.renderOpsView(width, max(height-2, 4))
	case subTabAudit:
		body = p.renderAuditView(width, max(height-2, 4))
	case subTabSessions:
		body = p.renderSessionsView(width, max(height-2, 4))
	}
	return lipgloss.JoinVertical(lipgloss.Left, topBar, body)
}

func (p *OpsAuditPanel) renderOpsView(width, height int) string {
	cardW := max(width-2, 30)
	if !p.canExecOps {
		return p.theme.Card.Width(cardW).Height(max(height, 6)).Render(
			p.theme.CardTitle.Render("Runtime Operations") + "\n\n" +
				p.theme.BadgeWarn.Render("Permission required: ops.execute"),
		)
	}
	var rows []string
	rows = append(rows, p.theme.CardTitle.Render("Runtime Operations"))
	if p.runningOp {
		rows = append(rows, p.theme.BadgeInfo.Render("Executing operation..."))
	} else {
		rows = append(rows, p.theme.MutedText.Render("Press Enter to confirm & run."))
	}
	rows = append(rows, "")
	for i, act := range defaultOpsActions {
		prefix := "  "
		st := p.theme.TableRow
		if i == p.opsIdx {
			prefix = "> "
			st = p.theme.TableSelected
		}
		rows = append(rows, st.Render(ui.PadRightWidth(prefix+act.Title, cardW-4)))
		rows = append(rows, "  "+p.theme.MutedText.Render(ui.TruncateWidth(act.Description, cardW-6)))
		rows = append(rows, "")
	}
	if p.lastOpRes != "" {
		rows = append(rows, p.theme.CardTitle.Render("Last Result"), p.theme.Value.Render(p.lastOpRes))
	}
	return p.theme.Card.Width(cardW).Height(max(height, 6)).Render(strings.Join(rows, "\n"))
}

func (p *OpsAuditPanel) renderAuditView(width, height int) string {
	if !p.canReadAudit {
		cardW := max(width-2, 30)
		return p.theme.Card.Width(cardW).Height(max(height, 6)).Render(
			p.theme.CardTitle.Render("Audit Logs") + "\n\n" +
				p.theme.BadgeWarn.Render("Permission required: audit.read"),
		)
	}
	prompt := p.theme.InputPrompt.Render(auditFilterPrompt)
	filterLine := prompt + p.auditInput.View()
	if lipgloss.Width(filterLine) > width {
		filterLine = prompt + ui.TruncateWidth(p.auditInput.View(), max(width-lipgloss.Width(prompt), 1))
	}

	split := width >= 88
	bodyH := max(height-2, 4)
	listW := width - 2
	detailW := width - 2
	listInnerH := max((bodyH/2)-3, 3)
	detailInnerH := max(bodyH-listInnerH-6, 4)
	if split {
		listW = max(width*48/100, 34)
		detailW = max(width-listW-1, 30)
		listInnerH = max(bodyH-3, 4)
		detailInnerH = max(bodyH-3, 4)
	}

	innerW := max(listW-4, 20)
	listLines := []string{
		p.theme.CardTitle.Render(fmt.Sprintf("Audit Logs (total %d, offset %d)", p.auditTotal, p.auditOffset)),
	}
	if p.auditErr != "" {
		listLines = append(listLines, p.theme.BadgeErr.Render("Error: "+p.auditErr))
	} else if len(p.auditEntries) == 0 {
		msg := "No audit entries."
		if p.loadingAudit {
			msg = "Loading audit logs..."
		}
		listLines = append(listLines, p.theme.MutedText.Render(msg))
	} else {
		visible := max(listInnerH-2, 1)
		start := 0
		if p.auditIdx >= visible {
			start = p.auditIdx - visible + 1
		}
		end := min(len(p.auditEntries), start+visible)
		actW := max(innerW-28, 12)
		listLines = append(listLines, p.theme.TableHeader.Render(
			ui.PadRightWidth("Time", 9)+" "+ui.PadRightWidth("Action", actW)+" "+ui.PadRightWidth("Actor", 11)+" "+ui.PadRightWidth("St", 5),
		))
		for i := start; i < end; i++ {
			e := p.auditEntries[i]
			row := ui.PadRightWidth(e.CreatedAt.Local().Format("15:04:05"), 9) + " " +
				ui.PadRightWidth(e.Action, actW) + " " +
				ui.PadRightWidth(e.ActorName, 11) + " " +
				ui.PadRightWidth(e.Status, 5)
			st := p.theme.TableRow
			if i == p.auditIdx {
				st = p.theme.TableSelected
			}
			listLines = append(listLines, st.Render(row))
		}
	}

	listCard := p.theme.Card.Width(listW).Height(listInnerH + 2).Render(strings.Join(listLines, "\n"))
	detailCard := p.theme.Card.Width(detailW).Height(detailInnerH + 2).Render(
		p.theme.CardTitle.Render("Audit Detail") + "\n" + p.auditVP.View(),
	)

	var panes string
	if split {
		panes = lipgloss.JoinHorizontal(lipgloss.Top, listCard, " ", detailCard)
	} else {
		panes = lipgloss.JoinVertical(lipgloss.Left, listCard, detailCard)
	}
	return lipgloss.JoinVertical(lipgloss.Left, filterLine, panes)
}

func (p *OpsAuditPanel) renderSessionsView(width, height int) string {
	cardW := max(width-2, 30)
	innerW := max(cardW-4, 24)
	lines := []string{
		p.theme.CardTitle.Render(fmt.Sprintf("Operator Sessions (%d)", len(p.sessions))),
		p.theme.MutedText.Render("[x] Revoke selected session  [r] Refresh"),
	}
	if p.sessionsErr != "" {
		lines = append(lines, p.theme.BadgeErr.Render("Error: "+p.sessionsErr))
	} else if len(p.sessions) == 0 {
		msg := "No active sessions."
		if p.loadingSessions {
			msg = "Loading sessions..."
		}
		lines = append(lines, p.theme.MutedText.Render(msg))
	} else {
		userW := max(innerW-44, 10)
		lines = append(lines, p.theme.TableHeader.Render(
			ui.PadRightWidth("Cur", 3)+" "+
				ui.PadRightWidth("User", userW)+" "+
				ui.PadRightWidth("Role", 13)+" "+
				ui.PadRightWidth("Last Used", 12)+" "+
				ui.PadRightWidth("Expires", 12),
		))
		maxRows := max(height-6, 3)
		start := 0
		if p.sessionIdx >= maxRows {
			start = p.sessionIdx - maxRows + 1
		}
		end := min(len(p.sessions), start+maxRows)
		for i := start; i < end; i++ {
			s := p.sessions[i]
			cur := ""
			if s.Current {
				cur = "*"
			}
			line := ui.PadRightWidth(cur, 3) + " " +
				ui.PadRightWidth("@"+s.Username, userW) + " " +
				ui.PadRightWidth(s.Role, 13) + " " +
				ui.PadRightWidth(s.LastUsedAt.Local().Format("01-02 15:04"), 12) + " " +
				ui.PadRightWidth(s.ExpiresAt.Local().Format("01-02 15:04"), 12)
			st := p.theme.TableRow
			if i == p.sessionIdx {
				st = p.theme.TableSelected
			}
			lines = append(lines, st.Render(line))
		}
	}
	return p.theme.Card.Width(cardW).Height(max(height, 6)).Render(strings.Join(lines, "\n"))
}

func (p *OpsAuditPanel) rebuildAuditDetail() {
	wrapW := max(p.auditVP.Width(), 1)
	if len(p.auditEntries) == 0 || p.auditIdx < 0 || p.auditIdx >= len(p.auditEntries) {
		p.auditVP.SetContent(ui.WrapViewportText(p.theme.MutedText.Render("Select an audit entry to inspect details."), wrapW))
		return
	}
	e := p.auditEntries[p.auditIdx]
	prettyDetail := e.DetailJSON
	if strings.TrimSpace(prettyDetail) != "" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(prettyDetail), "", "  "); err == nil {
			prettyDetail = buf.String()
		}
	} else {
		prettyDetail = "{}"
	}
	kv := func(k, v string) string {
		return p.theme.Label.Render(ui.PadRightWidth(k+":", 12)) + " " + p.theme.Value.Render(v)
	}
	p.auditVP.SetContent(ui.WrapViewportText(strings.Join([]string{
		kv("ID", e.ID),
		kv("Time", e.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		kv("Actor", fmt.Sprintf("%s (%s / %s)", e.ActorName, e.ActorRole, e.ActorUserID)),
		kv("Action", e.Action),
		kv("Resource", fmt.Sprintf("%s %s", e.ResourceType, e.ResourceID)),
		kv("Status", e.Status),
		kv("Remote", e.RemoteAddr),
		kv("Summary", e.Summary),
		"",
		p.theme.Label.Render("Detail JSON:"),
		prettyDetail,
	}, "\n"), wrapW))
}

func (p *OpsAuditPanel) onSubTabActivated(ctx ui.PanelContext) tea.Cmd {
	switch p.subTab {
	case subTabAudit:
		if ctx.HasPerm(admin.PermAuditRead) && len(p.auditEntries) == 0 && !p.loadingAudit {
			p.loadingAudit = true
			return p.fetchAuditCmd(ctx.Client)
		}
	case subTabSessions:
		if len(p.sessions) == 0 && !p.loadingSessions {
			p.loadingSessions = true
			return p.fetchSessionsCmd(ctx.Client)
		}
	}
	return nil
}

func (p *OpsAuditPanel) execOpCmd(client *adminclient.Client, opID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return opsExecutedMsg{ActionID: opID, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		switch opID {
		case "reload-locales":
			resp, err := client.ReloadLocales(reqCtx)
			if err != nil {
				return opsExecutedMsg{ActionID: opID, Err: err}
			}
			return opsExecutedMsg{ActionID: opID, Result: fmt.Sprintf("Reloaded locales: %s", strings.Join(resp.Locales, ", "))}
		case "sync-commands":
			resp, err := client.SyncCommands(reqCtx)
			if err != nil {
				return opsExecutedMsg{ActionID: opID, Err: err}
			}
			return opsExecutedMsg{ActionID: opID, Result: fmt.Sprintf("Synced %d global commands and %d private guilds", resp.GlobalCommands, resp.PrivateGuildCount)}
		case "bbs-dat-check":
			_, err := client.TriggerBBSDatCheck(reqCtx)
			if err != nil {
				return opsExecutedMsg{ActionID: opID, Err: err}
			}
			return opsExecutedMsg{ActionID: opID, Result: "BBS dat-ochi archive check completed"}
		default:
			return opsExecutedMsg{ActionID: opID, Err: fmt.Errorf("unknown op %q", opID)}
		}
	}
}

func (p *OpsAuditPanel) fetchAuditCmd(client *adminclient.Client) tea.Cmd {
	filter := admin.AuditFilter{
		Action: p.appliedAction,
		Limit:  auditPageLimit,
		Offset: p.auditOffset,
	}
	return func() tea.Msg {
		if client == nil {
			return auditLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListAuditLogs(reqCtx, filter)
		return auditLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *OpsAuditPanel) fetchSessionsCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return sessionsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListSessions(reqCtx)
		return sessionsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *OpsAuditPanel) revokeSessionCmd(client *adminclient.Client, sess admin.SessionInfo) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return sessionRevokedMsg{SessionID: sess.ID, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.RevokeSession(reqCtx, sess.ID)
		return sessionRevokedMsg{SessionID: sess.ID, Current: sess.Current, Err: err}
	}
}
