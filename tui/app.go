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
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/panels"
	bbspanel "github.com/sabafly/gobot/tui/panels/bbs"
	"github.com/sabafly/gobot/tui/ui"
)

const (
	tuiSidebarWidth    = 22
	toastClearDuration = 4 * time.Second
)

type appMode int

const (
	modeChecking appMode = iota
	modeLogin
	modeDashboard
)

type sessionCheckedMsg struct {
	Principal admin.PrincipalInfo
	Err       error
}

type logoutDoneMsg struct {
	Err error
}

type toastClearMsg struct {
	Gen uint64
}

// AppOptions configures the root Bubble Tea v2 TUI model.
type AppOptions struct {
	BaseURL       string
	Client        *adminclient.Client
	Store         *adminclient.FileSessionStore
	NoBrowser     bool
	Theme         *ui.Theme
	InitialPanels []ui.Panel
}

// App implements Bubble Tea v2 tea.Model for gobot's admin dashboard.
type App struct {
	theme   ui.Theme
	baseURL string
	client  *adminclient.Client
	store   *adminclient.FileSessionStore
	mode    appMode
	login   LoginModel

	principal     admin.PrincipalInfo
	allPanels     []ui.Panel
	visiblePanels []ui.Panel
	initialized   map[string]bool
	activeIdx     int
	showHelp      bool

	toastKind ui.ToastKind
	toastText string
	toastGen  uint64

	width  int
	height int
}

// NewApp creates the root Bubble Tea model and registers default panels if none are provided.
func NewApp(opts AppOptions) *App {
	theme := ui.DefaultTheme()
	if opts.Theme != nil {
		theme = *opts.Theme
	}
	baseURL := adminclient.NormalizeBaseURL(opts.BaseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	client := opts.Client
	if client == nil {
		client = adminclient.New(baseURL, "")
	}

	startMode := modeLogin
	if strings.TrimSpace(client.Token) != "" {
		startMode = modeChecking
	}

	a := &App{
		theme:       theme,
		baseURL:     baseURL,
		client:      client,
		store:       opts.Store,
		mode:        startMode,
		login:       NewLoginModel(theme, client, opts.Store, baseURL, opts.NoBrowser),
		initialized: make(map[string]bool),
		width:       100,
		height:      30,
	}

	if len(opts.InitialPanels) > 0 {
		for _, p := range opts.InitialPanels {
			a.RegisterPanel(p)
		}
	} else {
		a.RegisterPanel(panels.NewOverviewPanel(theme))
		a.RegisterPanel(panels.NewLogsPanel(theme))
		a.RegisterPanel(panels.NewGuildsPanel(theme))
		a.RegisterPanel(panels.NewOpsAuditPanel(theme))
		a.RegisterPanel(bbspanel.NewPanel(theme))
	}
	return a
}

// RegisterPanel appends a dashboard plugin panel to the shell's navigation registry.
func (a *App) RegisterPanel(p ui.Panel) {
	if p == nil {
		return
	}
	a.allPanels = append(a.allPanels, p)
	if a.mode == modeDashboard {
		a.rebuildVisiblePanels()
	}
}

// Close releases background resources (such as SSE streams) owned by registered panels.
func (a *App) Close() {
	for _, p := range a.allPanels {
		p.Close()
	}
}

// ActivePanel returns the currently visible dashboard panel, or nil when unavailable.
func (a *App) ActivePanel() ui.Panel {
	if len(a.visiblePanels) == 0 || a.activeIdx < 0 || a.activeIdx >= len(a.visiblePanels) {
		return nil
	}
	return a.visiblePanels[a.activeIdx]
}

// PanelContext builds the shared execution context passed to panels.
func (a *App) PanelContext() ui.PanelContext {
	return ui.PanelContext{
		Client:    a.client,
		Principal: a.principal,
		BaseURL:   a.baseURL,
	}
}

func (a *App) Init() tea.Cmd {
	if a.mode == modeChecking {
		client := a.client
		return func() tea.Msg {
			reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			me, err := client.Me(reqCtx)
			return sessionCheckedMsg{Principal: me, Err: err}
		}
	}
	var cmd tea.Cmd
	a.login, cmd = a.login.Start()
	return cmd
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		if m.Width > 0 {
			a.width = m.Width
		}
		if m.Height > 0 {
			a.height = m.Height
		}
		if a.mode == modeDashboard {
			return a, a.forwardActive(msg)
		}
		return a, nil

	case ui.ToastMsg:
		a.toastKind = m.Kind
		a.toastText = m.Text
		a.toastGen++
		gen := a.toastGen
		return a, tea.Tick(toastClearDuration, func(time.Time) tea.Msg {
			return toastClearMsg{Gen: gen}
		})

	case toastClearMsg:
		if m.Gen == a.toastGen {
			a.toastText = ""
		}
		return a, nil

	case ui.UnauthorizedMsg:
		a.Close()
		a.initialized = make(map[string]bool)
		if a.store != nil {
			_ = a.store.Delete(a.baseURL)
		}
		a.client.Token = ""
		a.principal = admin.PrincipalInfo{}
		a.visiblePanels = nil
		a.activeIdx = 0
		a.showHelp = false
		a.mode = modeLogin
		if m.Err != nil {
			a.login.notice = "Session expired or unauthorized. Please log in again."
		} else {
			a.login.notice = "Signed out."
		}
		var cmd tea.Cmd
		a.login, cmd = a.login.Start()
		return a, cmd

	case sessionCheckedMsg:
		if m.Err != nil {
			if a.store != nil && adminclient.IsUnauthorized(m.Err) {
				_ = a.store.Delete(a.baseURL)
				a.client.Token = ""
			}
			a.mode = modeLogin
			a.login.notice = "Existing session check failed (" + m.Err.Error() + "). Please log in."
			var cmd tea.Cmd
			a.login, cmd = a.login.Start()
			return a, cmd
		}
		a.principal = m.Principal
		a.mode = modeDashboard
		return a, a.activateInitialPanel()

	case LoginSuccessMsg:
		a.principal = m.Principal
		a.client.Token = m.Token
		a.mode = modeDashboard
		a.initialized = make(map[string]bool)
		return a, tea.Batch(
			ui.ShowToastCmd(ui.ToastSuccess, fmt.Sprintf("Logged in as @%s (%s)", m.Principal.Username, m.Principal.Role)),
			a.activateInitialPanel(),
		)

	case logoutDoneMsg:
		a.Close()
		a.initialized = make(map[string]bool)
		if a.store != nil {
			_ = a.store.Delete(a.baseURL)
		}
		a.client.Token = ""
		a.principal = admin.PrincipalInfo{}
		a.visiblePanels = nil
		a.activeIdx = 0
		a.showHelp = false
		a.mode = modeLogin
		a.login.notice = "Logged out."
		var cmd tea.Cmd
		a.login, cmd = a.login.Start()
		return a, cmd
	}

	switch a.mode {
	case modeChecking:
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			if kp.String() == "ctrl+c" || kp.String() == "q" {
				a.Close()
				return a, tea.Quit
			}
		}
		return a, nil

	case modeLogin:
		var cmd tea.Cmd
		a.login, cmd = a.login.Update(msg)
		return a, cmd

	case modeDashboard:
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			if kp.String() == "ctrl+c" {
				a.Close()
				return a, tea.Quit
			}
			if a.showHelp {
				switch kp.String() {
				case "?", "esc", "q":
					a.showHelp = false
				}
				return a, nil
			}
			active := a.ActivePanel()
			if active == nil || !active.CapturingInput() {
				switch kp.String() {
				case "q":
					a.Close()
					return a, tea.Quit
				case "?":
					a.showHelp = true
					return a, nil
				case "tab":
					return a, a.switchRelativePanel(1)
				case "shift+tab":
					return a, a.switchRelativePanel(-1)
				case "L":
					return a, a.logoutCmd()
				case "1", "2", "3", "4", "5", "6", "7", "8", "9":
					idx := int(kp.String()[0]-'0') - 1
					if idx >= 0 && idx < len(a.visiblePanels) {
						return a, a.switchToPanel(idx)
					}
				}
			}
		}
		return a, a.forwardMsgToPanels(msg)
	}
	return a, nil
}

func (a *App) View() tea.View {
	var body string
	var cur *tea.Cursor

	w := a.width
	h := a.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}

	switch a.mode {
	case modeChecking:
		box := a.theme.ModalBox.Width(min(max(w-4, 40), 60)).Render(
			a.theme.CardTitle.Render("gobot Admin Dashboard") + "\n\n" +
				a.theme.MutedText.Render("Verifying saved session with "+a.baseURL+"..."),
		)
		body = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)

	case modeLogin:
		body = a.login.View(w, h)

	case modeDashboard:
		header := a.renderHeader(w)
		footer := a.renderFooter(w)
		headerH := lipgloss.Height(header)
		footerH := lipgloss.Height(footer)
		availH := max(h-headerH-footerH, 6)

		wide := w >= 80
		var mainArea string
		var originX, originY, innerW, innerH int

		if wide {
			sideW := tuiSidebarWidth
			contentW := max(w-sideW-1, 30)
			innerW = max(contentW-a.theme.ContentBox.GetHorizontalFrameSize(), 20)
			innerH = max(availH-a.theme.ContentBox.GetVerticalFrameSize(), 6)

			sidebar := a.renderSidebar(sideW, availH)
			panelView := ui.ClampBox(a.renderActivePanel(innerW, innerH), innerW, innerH)
			contentBox := a.theme.ContentBox.Width(contentW).Height(availH).Render(panelView)
			mainArea = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", contentBox)

			originX = lipgloss.Width(sidebar) + 1 + a.theme.ContentBox.GetBorderLeftSize() + a.theme.ContentBox.GetPaddingLeft()
			originY = headerH + a.theme.ContentBox.GetBorderTopSize() + a.theme.ContentBox.GetPaddingTop()
		} else {
			topTabs := a.renderTopTabs(w)
			tabsH := lipgloss.Height(topTabs)
			boxH := max(availH-tabsH, 6)
			innerW = max(w-a.theme.ContentBox.GetHorizontalFrameSize(), 20)
			innerH = max(boxH-a.theme.ContentBox.GetVerticalFrameSize(), 4)

			panelView := ui.ClampBox(a.renderActivePanel(innerW, innerH), innerW, innerH)
			contentBox := a.theme.ContentBox.Width(w).Height(boxH).Render(panelView)
			mainArea = lipgloss.JoinVertical(lipgloss.Left, topTabs, contentBox)

			originX = a.theme.ContentBox.GetBorderLeftSize() + a.theme.ContentBox.GetPaddingLeft()
			originY = headerH + tabsH + a.theme.ContentBox.GetBorderTopSize() + a.theme.ContentBox.GetPaddingTop()
		}

		if a.showHelp {
			mainArea = a.renderHelpOverlay(w, availH)
		} else if ap := a.ActivePanel(); ap != nil {
			if pc := ap.Cursor(); pc != nil {
				copied := *pc
				copied.X = max(0, min(copied.X+originX, w-1))
				copied.Y = max(0, min(copied.Y+originY, h-1))
				cur = &copied
			}
		}

		body = lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)
	}

	v := tea.NewView(body)
	v.AltScreen = true
	v.Cursor = cur
	return v
}

func (a *App) rebuildVisiblePanels() {
	ctx := a.PanelContext()
	prev := a.ActivePanel()
	prevID := ""
	if prev != nil {
		prevID = prev.ID()
	}

	var visible []ui.Panel
	for _, p := range a.allPanels {
		if ctx.HasPerm(p.RequiredPermission()) {
			visible = append(visible, p)
		}
	}
	a.visiblePanels = visible
	a.activeIdx = 0
	for i, p := range a.visiblePanels {
		if p.ID() == prevID {
			a.activeIdx = i
			break
		}
	}
}

func (a *App) activateInitialPanel() tea.Cmd {
	a.rebuildVisiblePanels()
	ap := a.ActivePanel()
	if ap == nil {
		return nil
	}
	a.initialized[ap.ID()] = true
	return ap.Init(a.PanelContext())
}

func (a *App) switchRelativePanel(delta int) tea.Cmd {
	if len(a.visiblePanels) <= 1 {
		return nil
	}
	next := (a.activeIdx + delta + len(a.visiblePanels)) % len(a.visiblePanels)
	return a.switchToPanel(next)
}

func (a *App) switchToPanel(nextIdx int) tea.Cmd {
	if nextIdx < 0 || nextIdx >= len(a.visiblePanels) || nextIdx == a.activeIdx {
		return nil
	}
	ctx := a.PanelContext()
	var cmds []tea.Cmd
	if prev := a.ActivePanel(); prev != nil {
		updatedPrev, cmd := prev.Update(ui.PanelDeactivatedMsg{ID: prev.ID()}, ctx)
		a.replacePanel(updatedPrev)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	a.activeIdx = nextIdx
	next := a.ActivePanel()
	if next != nil {
		if !a.initialized[next.ID()] {
			a.initialized[next.ID()] = true
			if cmd := next.Init(ctx); cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			updatedNext, cmd := next.Update(ui.PanelActivatedMsg{ID: next.ID()}, ctx)
			a.replacePanel(updatedNext)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) forwardActive(msg tea.Msg) tea.Cmd {
	ap := a.ActivePanel()
	if ap == nil {
		return nil
	}
	updated, cmd := ap.Update(msg, a.PanelContext())
	a.replacePanel(updated)
	return cmd
}

func (a *App) forwardMsgToPanels(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg:
		return a.forwardActive(msg)
	}
	// Route async panel messages ONLY to initialized panels, and avoid calling Update redundantly on panels not handling it.
	ctx := a.PanelContext()
	var cmds []tea.Cmd
	for _, p := range a.visiblePanels {
		if !a.initialized[p.ID()] {
			continue
		}
		updated, cmd := p.Update(msg, ctx)
		a.replacePanel(updated)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) replacePanel(updated ui.Panel) {
	if updated == nil {
		return
	}
	id := updated.ID()
	for i, p := range a.allPanels {
		if p.ID() == id {
			a.allPanels[i] = updated
			break
		}
	}
	for i, p := range a.visiblePanels {
		if p.ID() == id {
			a.visiblePanels[i] = updated
			break
		}
	}
}

func (a *App) logoutCmd() tea.Cmd {
	client := a.client
	return func() tea.Msg {
		reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.Logout(reqCtx)
		return logoutDoneMsg{Err: err}
	}
}

func (a *App) renderHeader(width int) string {
	userStr := "@" + a.principal.Username
	if a.principal.DisplayName != "" && a.principal.DisplayName != a.principal.Username {
		userStr = fmt.Sprintf("%s (@%s)", a.principal.DisplayName, a.principal.Username)
	} else if a.principal.Username == "" {
		userStr = a.principal.UserID
	}
	left := fmt.Sprintf("gobot admin  |  %s  |  %s [%s]", a.baseURL, userStr, a.principal.Role)
	right := a.theme.BadgeOK.Render("● Connected")
	space := max(width-2-lipgloss.Width(left)-lipgloss.Width(right), 1)
	line := ui.TruncateWidth(left, max(width-2-lipgloss.Width(right)-1, 10)) + strings.Repeat(" ", space) + right
	return a.theme.Header.Width(width).Render(line)
}

func (a *App) renderSidebar(width, height int) string {
	innerW := max(width-a.theme.Sidebar.GetHorizontalFrameSize(), 8)
	innerH := max(height-a.theme.Sidebar.GetVerticalFrameSize(), 1)
	var rows []string
	rows = append(rows, a.theme.CardTitle.Render("Panels"), "")
	for i, p := range a.visiblePanels {
		label := fmt.Sprintf("[%d] %s", i+1, p.Title())
		st := a.theme.SidebarItem
		if i == a.activeIdx {
			st = a.theme.SidebarActive
		}
		rows = append(rows, st.Width(innerW).Render(ui.TruncateWidth(label, innerW)))
	}
	rows = append(rows, "", a.theme.MutedText.Render(ui.TruncateWidth("Tab/1-9 switch", innerW)))
	rows = append(rows, a.theme.MutedText.Render(ui.TruncateWidth("? help  L logout", innerW)))
	content := ui.ClampBox(strings.Join(rows, "\n"), innerW, innerH)
	return a.theme.Sidebar.Width(width).Height(height).Render(content)
}

func (a *App) renderTopTabs(width int) string {
	var parts []string
	for i, p := range a.visiblePanels {
		st := a.theme.SubTab
		if i == a.activeIdx {
			st = a.theme.SubTabActive
		}
		parts = append(parts, st.Render(fmt.Sprintf("[%d] %s", i+1, p.Title())))
	}
	return ui.TruncateWidth(lipgloss.JoinHorizontal(lipgloss.Center, parts...), width)
}

func (a *App) renderActivePanel(innerW, innerH int) string {
	ap := a.ActivePanel()
	if ap == nil {
		return a.theme.BadgeWarn.Render("No panels accessible for role " + a.principal.Role)
	}
	return ap.View(innerW, innerH)
}

func (a *App) renderFooter(width int) string {
	var hints []string
	if ap := a.ActivePanel(); ap != nil {
		for _, b := range ap.ShortHelp(a.PanelContext()) {
			h := b.Help()
			if h.Key != "" {
				hints = append(hints, fmt.Sprintf("%s:%s", h.Key, h.Desc))
			}
		}
	}
	hints = append(hints, "Tab:panel", "?:help", "L:logout", "q:quit")
	left := strings.Join(hints, "  ")

	var toastRendered string
	if a.toastText != "" {
		switch a.toastKind {
		case ui.ToastSuccess:
			toastRendered = a.theme.ToastSuccess.Render("[" + a.toastText + "]")
		case ui.ToastWarn:
			toastRendered = a.theme.ToastWarn.Render("[" + a.toastText + "]")
		case ui.ToastError:
			toastRendered = a.theme.ToastError.Render("[" + a.toastText + "]")
		default:
			toastRendered = a.theme.ToastInfo.Render("[" + a.toastText + "]")
		}
	}

	availW := max(width-2, 10)
	if toastRendered == "" {
		return a.theme.Footer.Width(width).Render(ui.TruncateWidth(left, availW))
	}
	maxLeft := max(availW-lipgloss.Width(toastRendered)-2, 8)
	leftTrimmed := ui.TruncateWidth(left, maxLeft)
	gap := max(availW-lipgloss.Width(leftTrimmed)-lipgloss.Width(toastRendered), 1)
	return a.theme.Footer.Width(width).Render(leftTrimmed + strings.Repeat(" ", gap) + toastRendered)
}

func (a *App) renderHelpOverlay(width, height int) string {
	boxW := max(min(width-4, 72), 40)
	lines := []string{
		a.theme.CardTitle.Render("Keyboard Shortcuts"),
		"",
		"Global:",
		"  Tab / Shift+Tab   Cycle dashboard panels",
		"  1..9              Jump directly to panel",
		"  ? / Esc           Toggle / close help",
		"  L                 Logout current session",
		"  q / Ctrl+C        Quit TUI",
	}
	if ap := a.ActivePanel(); ap != nil {
		lines = append(lines, "", "Panel ("+ap.Title()+"):")
		for _, b := range ap.ShortHelp(a.PanelContext()) {
			h := b.Help()
			lines = append(lines, fmt.Sprintf("  %-16s  %s", h.Key, h.Desc))
		}
	}
	lines = append(lines, "", a.theme.MutedText.Render("User Permissions: "+joinPerms(a.principal.Permissions)))
	box := a.theme.HelpBox.Width(boxW).Render(strings.Join(lines, "\n"))
	return ui.ClampBox(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box), width, height)
}

func joinPerms(perms []admin.Permission) string {
	if len(perms) == 0 {
		return "(none)"
	}
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		out = append(out, string(p))
	}
	return strings.Join(out, ", ")
}
