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

package bbs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

type subTab int

const (
	subTabReports subTab = iota
	subTabBoards
	subTabCategories
	subTabBans
	subTabCaps
	subTabSlip
)

const numSubTabs = 6

type statsLoadedMsg struct {
	Resp admin.BBSStatsResponse
	Err  error
}

type reportsLoadedMsg struct {
	Resp admin.BBSReportsResponse
	Err  error
}

type reportInspectedMsg struct {
	ReportID string
	Resp     admin.BBSReportInspectResponse
	Err      error
}

type reportResolvedMsg struct {
	Resp admin.BBSReportDTO
	Err  error
}

type boardsLoadedMsg struct {
	Resp admin.BBSBoardsResponse
	Err  error
}

type boardMountsLoadedMsg struct {
	BoardID string
	Resp    admin.BBSBoardMountsResponse
	Err     error
}

type boardSavedMsg struct {
	Created bool
	Resp    admin.BBSBoardDTO
	Err     error
}

type boardDeletedMsg struct {
	ID  string
	Err error
}

type threadsLoadedMsg struct {
	BoardID string
	Resp    admin.BBSThreadsResponse
	Err     error
}

type threadDetailLoadedMsg struct {
	ThreadID string
	Resp     admin.BBSThreadDetailResponse
	Err      error
}

type threadArchivedMsg struct {
	Resp admin.BBSArchiveThreadResponse
	Err  error
}

type postAbonedMsg struct {
	FromReport bool
	Resp       admin.BBSAbonePostResponse
	Err        error
}

type categoriesLoadedMsg struct {
	Resp admin.BBSCategoriesResponse
	Err  error
}

type categorySavedMsg struct {
	Created bool
	Resp    admin.BBSCategoryDTO
	Err     error
}

type categoryDeletedMsg struct {
	ID  string
	Err error
}

type bansLoadedMsg struct {
	Resp admin.BBSBansResponse
	Err  error
}

type banCreatedMsg struct {
	Resp admin.BBSBanDTO
	Err  error
}

type banDeletedMsg struct {
	ID  string
	Err error
}

type capsLoadedMsg struct {
	Resp admin.BBSCapsResponse
	Err  error
}

type capSavedMsg struct {
	Created bool
	Resp    admin.BBSCapDTO
	Err     error
}

type capDeletedMsg struct {
	ID  string
	Err error
}

type slipEventsLoadedMsg struct {
	Resp admin.BBSSlipEventsResponse
	Err  error
}

type slipEventSavedMsg struct {
	Created bool
	Resp    admin.BBSSlipEventDTO
	Err     error
}

type slipEventDeletedMsg struct {
	ID  string
	Err error
}

// Panel implements the BBS administration UI with 6 subtabs covering reports,
// boards/threads/posts drill-down, categories, bans, admin caps, and SLIP events.
type Panel struct {
	theme        ui.Theme
	subTab       subTab
	stats        *admin.BBSStatsResponse
	loadingStats bool

	formModal    ui.FormModal
	confirmModal ui.ConfirmModal
	selectModal  ui.SelectModal

	reports    reportsState
	boards     boardsState
	categories categoriesState
	bans       bansState
	caps       capsState
	slip       slipState

	width  int
	height int
}

// NewPanel constructs the BBS management TUI panel.
func NewPanel(theme ui.Theme) *Panel {
	return &Panel{
		theme:      theme,
		subTab:     subTabReports,
		reports:    newReportsState(),
		boards:     newBoardsState(),
		categories: newCategoriesState(),
		bans:       newBansState(),
		caps:       newCapsState(),
		slip:       newSlipState(),
	}
}

func (p *Panel) ID() string                           { return "bbs" }
func (p *Panel) Title() string                        { return "BBS" }
func (p *Panel) RequiredPermission() admin.Permission { return admin.PermBBSRead }
func (p *Panel) Close()                               {}

func (p *Panel) openForm(tag, title, subtitle string, fields []ui.FormField, payload any) tea.Cmd {
	p.formModal.SetSize(p.width, p.height, p.theme)
	return p.formModal.Open(tag, title, subtitle, fields, payload)
}

func (p *Panel) CapturingInput() bool {
	return p.formModal.Active || p.confirmModal.Active || p.selectModal.Active ||
		(p.subTab == subTabBoards && p.boards.level == boardLevelThreads && p.boards.searching)
}

func (p *Panel) Cursor() *tea.Cursor {
	if p.formModal.Active {
		return p.formModal.Cursor(p.width, p.height, p.theme)
	}
	if p.confirmModal.Active || p.selectModal.Active {
		return nil
	}
	if p.subTab == subTabBoards && p.boards.level == boardLevelThreads && p.boards.searching && p.boards.searchInput.Focused() {
		c := p.boards.searchInput.Cursor()
		if c == nil {
			return nil
		}
		cur := *c
		cur.X += lipgloss.Width(threadSearchPrompt)
		cur.Y += 3
		return &cur
	}
	return nil
}

func (p *Panel) ShortHelp(ctx ui.PanelContext) []key.Binding {
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("left", "right", "h", "l", "[", "]"), key.WithHelp("←/→,[/]", "subtab")),
		key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select")),
		key.NewBinding(key.WithKeys("J", "K"), key.WithHelp("J/K", "scroll")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
	switch p.subTab {
	case subTabReports:
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "status")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "scope")),
		)
		if ctx.HasPerm(admin.PermBBSModerate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
				key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "resolve")),
				key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "dismiss")),
				key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "abone")),
				key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "ban")),
			)
		}
	case subTabBoards:
		switch p.boards.level {
		case boardLevelBoards:
			bindings = append(bindings, key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "threads")))
			if ctx.HasPerm(admin.PermBBSAdmin) {
				bindings = append(bindings,
					key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
					key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
					key.NewBinding(key.WithKeys("D", "delete"), key.WithHelp("D/del", "delete")),
				)
			}
		case boardLevelThreads:
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
				key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "status")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "posts")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "back")),
			)
			if ctx.HasPerm(admin.PermBBSModerate) {
				bindings = append(bindings, key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "archive")))
			}
		case boardLevelPosts:
			bindings = append(bindings, key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "back")))
			if ctx.HasPerm(admin.PermBBSModerate) {
				bindings = append(bindings,
					key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "abone")),
					key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "ban")),
				)
			}
		}
	case subTabCategories:
		bindings = append(bindings, key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "scope")))
		if ctx.HasPerm(admin.PermBBSAdmin) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
				key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
				key.NewBinding(key.WithKeys("D", "delete"), key.WithHelp("D/del", "delete")),
			)
		}
	case subTabBans:
		bindings = append(bindings, key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "scope")))
		if ctx.HasPerm(admin.PermBBSModerate) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "ban")),
				key.NewBinding(key.WithKeys("D", "delete"), key.WithHelp("D/del", "lift")),
			)
		}
	case subTabCaps, subTabSlip:
		if ctx.HasPerm(admin.PermBBSAdmin) {
			bindings = append(bindings,
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
				key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
				key.NewBinding(key.WithKeys("D", "delete"), key.WithHelp("D/del", "delete")),
			)
		}
	}
	return bindings
}

func (p *Panel) Init(ctx ui.PanelContext) tea.Cmd {
	p.loadingStats = true
	p.reports.loading = true
	return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchReportsCmd(ctx.Client))
}

func (p *Panel) fetchStatsCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return statsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.BBSStats(reqCtx)
		return statsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) Update(msg tea.Msg, ctx ui.PanelContext) (ui.Panel, tea.Cmd) {
	if act, ok := msg.(ui.PanelActivatedMsg); ok && act.ID == p.ID() {
		return p, tea.Batch(p.fetchStatsCmd(ctx.Client), p.onSubTabActivated(ctx))
	}

	// Async responses run before modal/input interception so background updates always land cleanly.
	if cmd, handled := p.handleAsyncMsg(msg, ctx); handled {
		return p, cmd
	}

	if p.formModal.Active {
		submitted, canceled, tag, values, payload, cmd := p.formModal.Update(msg)
		if canceled {
			return p, nil
		}
		if submitted {
			return p, p.handleFormSubmit(ctx, tag, values, payload)
		}
		return p, cmd
	}

	if p.confirmModal.Active {
		confirmed, canceled, tag, payload := p.confirmModal.Update(msg)
		if canceled {
			return p, nil
		}
		if confirmed {
			return p, p.handleConfirmSubmit(ctx, tag, payload)
		}
		return p, nil
	}

	if p.selectModal.Active {
		_, canceled, _, _ := p.selectModal.Update(msg)
		if canceled {
			return p, nil
		}
		return p, nil
	}

	if k, ok := msg.(tea.KeyPressMsg); ok && !p.CapturingInput() {
		switch k.String() {
		case "[", "left", "h":
			p.subTab = (p.subTab + numSubTabs - 1) % numSubTabs
			return p, p.onSubTabActivated(ctx)
		case "]", "right", "l":
			p.subTab = (p.subTab + 1) % numSubTabs
			return p, p.onSubTabActivated(ctx)
		}
	}

	switch p.subTab {
	case subTabReports:
		return p, p.updateReports(msg, ctx)
	case subTabBoards:
		return p, p.updateBoards(msg, ctx)
	case subTabCategories:
		return p, p.updateCategories(msg, ctx)
	case subTabBans:
		return p, p.updateBans(msg, ctx)
	case subTabCaps:
		return p, p.updateCaps(msg, ctx)
	case subTabSlip:
		return p, p.updateSlip(msg, ctx)
	}
	return p, nil
}

func (p *Panel) onSubTabActivated(ctx ui.PanelContext) tea.Cmd {
	switch p.subTab {
	case subTabReports:
		if !p.reports.loaded && !p.reports.loading {
			p.reports.loading = true
			return p.fetchReportsCmd(ctx.Client)
		}
	case subTabBoards:
		var cmds []tea.Cmd
		if !p.boards.loaded && !p.boards.loading {
			p.boards.loading = true
			cmds = append(cmds, p.fetchBoardsCmd(ctx.Client))
		}
		if !p.categories.loaded && !p.categories.loading {
			p.categories.loading = true
			cmds = append(cmds, p.fetchCategoriesCmd(ctx.Client))
		}
		return tea.Batch(cmds...)
	case subTabCategories:
		if !p.categories.loaded && !p.categories.loading {
			p.categories.loading = true
			return p.fetchCategoriesCmd(ctx.Client)
		}
	case subTabBans:
		if !p.bans.loaded && !p.bans.loading {
			p.bans.loading = true
			return p.fetchBansCmd(ctx.Client)
		}
	case subTabCaps:
		if !p.caps.loaded && !p.caps.loading {
			p.caps.loading = true
			return p.fetchCapsCmd(ctx.Client)
		}
	case subTabSlip:
		if !p.slip.loaded && !p.slip.loading {
			p.slip.loading = true
			return p.fetchSlipEventsCmd(ctx.Client)
		}
	}
	return nil
}

func (p *Panel) handleAsyncMsg(msg tea.Msg, ctx ui.PanelContext) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case statsLoadedMsg:
		p.loadingStats = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return nil, true
		}
		resp := m.Resp
		p.stats = &resp
		return nil, true

	case reportsLoadedMsg:
		p.reports.loading = false
		p.reports.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.reports.errMsg = m.Err.Error()
			p.reports.rebuildDetail(p.theme)
			return nil, true
		}
		p.reports.errMsg = ""
		p.reports.items = m.Resp.Reports
		p.reports.selectedIdx = restoreIdxByID(p.reports.selectedID, len(p.reports.items), func(i int) string { return p.reports.items[i].ID })
		if rep, ok := p.reports.selected(); ok {
			p.reports.selectedID = rep.ID
		} else {
			p.reports.selectedID = ""
		}
		p.reports.rebuildDetail(p.theme)
		return nil, true

	case reportInspectedMsg:
		p.reports.inspecting = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Inspect failed: "+m.Err.Error()), true
		}
		resp := m.Resp
		p.reports.inspect = &resp
		p.reports.rebuildDetail(p.theme)
		return ui.ShowToastCmd(ui.ToastInfo, "Inspection loaded (audit logged)"), true

	case reportResolvedMsg:
		p.reports.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Resolve failed: "+m.Err.Error()), true
		}
		p.reports.loading = true
		return tea.Batch(p.fetchReportsCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Report "+m.Resp.Status)), true

	case boardsLoadedMsg:
		p.boards.loading = false
		p.boards.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.boards.errMsg = m.Err.Error()
			p.boards.rebuildDetail(p.theme)
			return nil, true
		}
		p.boards.errMsg = ""
		p.boards.items = m.Resp.Boards
		p.boards.selectedIdx = restoreIdxByID(p.boards.selectedID, len(p.boards.items), func(i int) string { return p.boards.items[i].ID })
		var cmd tea.Cmd
		if b, ok := p.boards.selectedBoard(); ok {
			p.boards.selectedID = b.ID
			if _, cached := p.boards.mountsByID[b.ID]; !cached {
				cmd = p.fetchBoardMountsCmd(ctx.Client, b.ID)
			}
		} else {
			p.boards.selectedID = ""
		}
		p.boards.rebuildDetail(p.theme)
		return cmd, true

	case boardMountsLoadedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return nil, true
		}
		p.boards.mountsByID[m.BoardID] = m.Resp.Mounts
		if p.subTab == subTabBoards && p.boards.level == boardLevelBoards {
			p.boards.rebuildDetail(p.theme)
		}
		return nil, true

	case boardSavedMsg:
		p.boards.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Board save failed: "+m.Err.Error()), true
		}
		p.boards.selectedID = m.Resp.ID
		p.boards.loading = true
		action := "Updated board "
		if m.Created {
			action = "Created board "
		}
		return tea.Batch(p.fetchBoardsCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, action+m.Resp.Name)), true

	case boardDeletedMsg:
		p.boards.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Delete board failed: "+m.Err.Error()), true
		}
		p.boards.loading = true
		return tea.Batch(p.fetchBoardsCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Board deleted")), true

	case threadsLoadedMsg:
		if b, ok := p.boards.selectedBoard(); ok && b.ID != m.BoardID {
			return nil, true
		}
		p.boards.loading = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.boards.errMsg = m.Err.Error()
			p.boards.rebuildDetail(p.theme)
			return nil, true
		}
		p.boards.errMsg = ""
		p.boards.threads = m.Resp.Threads
		p.boards.threadIdx = restoreIdxByID(p.boards.selectedThreadID, len(p.boards.threads), func(i int) string { return p.boards.threads[i].ID })
		if th, ok := p.boards.selectedThread(); ok {
			p.boards.selectedThreadID = th.ID
		} else {
			p.boards.selectedThreadID = ""
		}
		p.boards.rebuildDetail(p.theme)
		return nil, true

	case threadDetailLoadedMsg:
		if th, ok := p.boards.selectedThread(); ok && th.ID != m.ThreadID {
			return nil, true
		}
		p.boards.loading = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.boards.errMsg = m.Err.Error()
			p.boards.rebuildDetail(p.theme)
			return nil, true
		}
		p.boards.errMsg = ""
		resp := m.Resp
		p.boards.threadDetail = &resp
		if p.boards.postIdx >= len(resp.Posts) {
			p.boards.postIdx = max(len(resp.Posts)-1, 0)
		}
		p.boards.rebuildDetail(p.theme)
		return nil, true

	case threadArchivedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Archive failed: "+m.Err.Error()), true
		}
		var cmds []tea.Cmd
		if b, ok := p.boards.selectedBoard(); ok {
			p.boards.loading = true
			cmds = append(cmds, p.fetchThreadsCmd(ctx.Client, b.ID))
		}
		cmds = append(cmds, p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Archived thread "+m.Resp.Thread.Title))
		return tea.Batch(cmds...), true

	case postAbonedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Abone failed: "+m.Err.Error()), true
		}
		msgTxt := fmt.Sprintf("Aboned post #%d", m.Resp.Post.ResNumber)
		if m.Resp.SyncError != "" {
			msgTxt += " (sync warn: " + m.Resp.SyncError + ")"
		}
		if m.FromReport {
			return tea.Batch(p.fetchReportsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, msgTxt)), true
		}
		if th, ok := p.boards.selectedThread(); ok {
			return tea.Batch(p.fetchThreadDetailCmd(ctx.Client, th.ID), ui.ShowToastCmd(ui.ToastSuccess, msgTxt)), true
		}
		return ui.ShowToastCmd(ui.ToastSuccess, msgTxt), true

	case categoriesLoadedMsg:
		p.categories.loading = false
		p.categories.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.categories.errMsg = m.Err.Error()
			p.categories.rebuildDetail(p.theme)
			return nil, true
		}
		p.categories.errMsg = ""
		p.categories.items = m.Resp.Categories
		p.categories.selectedIdx = restoreIdxByID(p.categories.selectedID, len(p.categories.items), func(i int) string { return p.categories.items[i].ID })
		if cat, ok := p.categories.selected(); ok {
			p.categories.selectedID = cat.ID
		} else {
			p.categories.selectedID = ""
		}
		p.categories.rebuildDetail(p.theme)
		return nil, true

	case categorySavedMsg:
		p.categories.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Category save failed: "+m.Err.Error()), true
		}
		p.categories.selectedID = m.Resp.ID
		p.categories.loading = true
		action := "Updated category "
		if m.Created {
			action = "Created category "
		}
		return tea.Batch(p.fetchCategoriesCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, action+m.Resp.Name)), true

	case categoryDeletedMsg:
		p.categories.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Delete category failed: "+m.Err.Error()), true
		}
		p.categories.loading = true
		p.boards.loaded = false
		return tea.Batch(p.fetchCategoriesCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Category deleted")), true

	case bansLoadedMsg:
		p.bans.loading = false
		p.bans.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.bans.errMsg = m.Err.Error()
			p.bans.rebuildDetail(p.theme)
			return nil, true
		}
		p.bans.errMsg = ""
		p.bans.items = m.Resp.Bans
		p.bans.selectedIdx = restoreIdxByID(p.bans.selectedID, len(p.bans.items), func(i int) string { return p.bans.items[i].ID })
		if b, ok := p.bans.selected(); ok {
			p.bans.selectedID = b.ID
		} else {
			p.bans.selectedID = ""
		}
		p.bans.rebuildDetail(p.theme)
		return nil, true

	case banCreatedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Ban failed: "+m.Err.Error()), true
		}
		p.bans.selectedID = m.Resp.ID
		p.bans.loading = true
		return tea.Batch(p.fetchBansCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Ban created ("+m.Resp.Scope+")")), true

	case banDeletedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Lift ban failed: "+m.Err.Error()), true
		}
		p.bans.loading = true
		return tea.Batch(p.fetchBansCmd(ctx.Client), p.fetchStatsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Ban lifted")), true

	case capsLoadedMsg:
		p.caps.loading = false
		p.caps.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.caps.errMsg = m.Err.Error()
			p.caps.rebuildDetail(p.theme)
			return nil, true
		}
		p.caps.errMsg = ""
		p.caps.items = m.Resp.Caps
		p.caps.selectedIdx = restoreIdxByID(p.caps.selectedID, len(p.caps.items), func(i int) string { return p.caps.items[i].ID })
		if c, ok := p.caps.selected(); ok {
			p.caps.selectedID = c.ID
		} else {
			p.caps.selectedID = ""
		}
		p.caps.rebuildDetail(p.theme)
		return nil, true

	case capSavedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Save cap failed: "+m.Err.Error()), true
		}
		p.caps.selectedID = m.Resp.ID
		p.caps.loading = true
		lbl := "Updated cap "
		if m.Created {
			lbl = "Created cap "
		}
		return tea.Batch(p.fetchCapsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, lbl+m.Resp.ID)), true

	case capDeletedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Delete cap failed: "+m.Err.Error()), true
		}
		p.caps.loading = true
		return tea.Batch(p.fetchCapsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Deleted cap "+m.ID)), true

	case slipEventsLoadedMsg:
		p.slip.loading = false
		p.slip.loaded = true
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			p.slip.errMsg = m.Err.Error()
			p.slip.rebuildDetail(p.theme)
			return nil, true
		}
		p.slip.errMsg = ""
		p.slip.items = m.Resp.Events
		p.slip.selectedIdx = restoreIdxByID(p.slip.selectedID, len(p.slip.items), func(i int) string { return p.slip.items[i].ID })
		if ev, ok := p.slip.selected(); ok {
			p.slip.selectedID = ev.ID
		} else {
			p.slip.selectedID = ""
		}
		p.slip.rebuildDetail(p.theme)
		return nil, true

	case slipEventSavedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Save SLIP event failed: "+m.Err.Error()), true
		}
		p.slip.selectedID = m.Resp.ID
		p.slip.loading = true
		lbl := "Updated SLIP event "
		if m.Created {
			lbl = "Created SLIP event "
		}
		return tea.Batch(p.fetchSlipEventsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, lbl+m.Resp.ID)), true

	case slipEventDeletedMsg:
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }, true
			}
			return ui.ShowToastCmd(ui.ToastError, "Delete SLIP event failed: "+m.Err.Error()), true
		}
		p.slip.loading = true
		return tea.Batch(p.fetchSlipEventsCmd(ctx.Client), ui.ShowToastCmd(ui.ToastSuccess, "Deleted SLIP event "+m.ID)), true
	}
	return nil, false
}

func (p *Panel) handleFormSubmit(ctx ui.PanelContext, tag string, v map[string]string, payload any) tea.Cmd {
	switch tag {
	case "report.resolve":
		repID, _ := payload.(string)
		return p.resolveReportCmd(ctx.Client, repID, v["status"], v["note"])
	case "report.dismiss":
		repID, _ := payload.(string)
		return p.resolveReportCmd(ctx.Client, repID, "dismissed", v["note"])
	case "report.abone":
		postID, _ := payload.(string)
		return p.abonePostCmd(ctx.Client, postID, v["reason"], true)
	case "report.ban", "post.ban", "ban.create":
		hrs, err := strconv.Atoi(strings.TrimSpace(v["duration_hours"]))
		if err != nil || hrs < 0 {
			return ui.ShowToastCmd(ui.ToastError, "duration_hours must be >= 0")
		}
		return p.createBanCmd(ctx.Client, admin.CreateBBSBanRequest{
			Scope:            strings.TrimSpace(v["scope"]),
			GuildID:          strings.TrimSpace(v["guild_id"]),
			PostID:           strings.TrimSpace(v["post_id"]),
			TargetUserID:     strings.TrimSpace(v["target_user_id"]),
			TargetPosterHash: strings.TrimSpace(v["target_poster_hash"]),
			Reason:           strings.TrimSpace(v["reason"]),
			DurationHours:    hrs,
		})
	case "board.create":
		catVal := strings.TrimSpace(v["category"])
		if _, after, ok := strings.Cut(catVal, "|"); ok {
			catVal = strings.TrimSpace(after)
		}
		slip, err := strconv.Atoi(strings.TrimSpace(v["slip_level"]))
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid SLIP level")
		}
		maxRes, err := strconv.ParseUint(strings.TrimSpace(v["max_res"]), 10, 32)
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid max_res")
		}
		idleH, err := strconv.Atoi(strings.TrimSpace(v["dat_ochi_idle"]))
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid dat_ochi_idle")
		}
		return p.createBoardCmd(ctx.Client, admin.CreateBBSBoardRequest{
			CategoryID:  catVal,
			Slug:        strings.TrimSpace(v["slug"]),
			Name:        strings.TrimSpace(v["name"]),
			Description: strings.TrimSpace(v["description"]),
			Rules: admin.BBSBoardRulesDTO{
				DefaultName:      strings.TrimSpace(v["default_name"]),
				HideID:           v["hide_id"] == "true",
				SLIPLevel:        slip,
				MaxRes:           uint32(maxRes),
				DatOchiIdleHours: idleH,
			},
		})
	case "board.edit":
		b, _ := payload.(admin.BBSBoardDTO)
		name := strings.TrimSpace(v["name"])
		desc := strings.TrimSpace(v["description"])
		defName := strings.TrimSpace(v["default_name"])
		hideID := v["hide_id"] == "true"
		locked := v["locked"] == "true"
		slip, err := strconv.Atoi(strings.TrimSpace(v["slip_level"]))
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid SLIP level")
		}
		return p.updateBoardCmd(ctx.Client, b.ID, admin.UpdateBBSBoardRequest{
			Name:        &name,
			Description: &desc,
			DefaultName: &defName,
			HideID:      &hideID,
			SLIPLevel:   &slip,
			Locked:      &locked,
		})
	case "post.abone":
		post, _ := payload.(admin.BBSPostDTO)
		return p.abonePostCmd(ctx.Client, post.ID, v["reason"], false)
	case "category.create":
		sortOrd, err := strconv.Atoi(strings.TrimSpace(v["sort_order"]))
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid sort_order")
		}
		return p.createCategoryCmd(ctx.Client, admin.CreateBBSCategoryRequest{
			Scope:       strings.TrimSpace(v["scope"]),
			GuildID:     strings.TrimSpace(v["guild_id"]),
			Slug:        strings.TrimSpace(v["slug"]),
			Name:        strings.TrimSpace(v["name"]),
			Description: strings.TrimSpace(v["description"]),
			SortOrder:   sortOrd,
		})
	case "category.edit":
		cat, _ := payload.(admin.BBSCategoryDTO)
		name := strings.TrimSpace(v["name"])
		slug := strings.TrimSpace(v["slug"])
		desc := strings.TrimSpace(v["description"])
		sortOrd, err := strconv.Atoi(strings.TrimSpace(v["sort_order"]))
		if err != nil {
			return ui.ShowToastCmd(ui.ToastError, "invalid sort_order")
		}
		return p.updateCategoryCmd(ctx.Client, cat.ID, admin.UpdateBBSCategoryRequest{
			Name:        &name,
			Slug:        &slug,
			Description: &desc,
			SortOrder:   &sortOrd,
		})
	case "cap.create":
		return p.saveCapCmd(ctx.Client, true, "", admin.SaveBBSCapRequest{
			ID:             strings.TrimSpace(v["id"]),
			Name:           strings.TrimSpace(v["name"]),
			Description:    strings.TrimSpace(v["description"]),
			GrantOperators: v["grant_operators"] == "true",
			Users:          splitCSV(v["users"]),
		})
	case "cap.edit":
		c, _ := payload.(admin.BBSCapDTO)
		return p.saveCapCmd(ctx.Client, false, c.ID, admin.SaveBBSCapRequest{
			ID:             c.ID,
			Name:           strings.TrimSpace(v["name"]),
			Description:    strings.TrimSpace(v["description"]),
			GrantOperators: v["grant_operators"] == "true",
			Users:          splitCSV(v["users"]),
		})
	case "slip.create":
		return p.saveSlipEventCmd(ctx.Client, true, "", admin.SaveBBSSlipEventRequest{
			ID:        strings.TrimSpace(v["id"]),
			Label:     strings.TrimSpace(v["label"]),
			StartDate: strings.TrimSpace(v["start_date"]),
			EndDate:   strings.TrimSpace(v["end_date"]),
			Mode:      strings.TrimSpace(v["mode"]),
		})
	case "slip.edit":
		ev, _ := payload.(admin.BBSSlipEventDTO)
		return p.saveSlipEventCmd(ctx.Client, false, ev.ID, admin.SaveBBSSlipEventRequest{
			ID:        ev.ID,
			Label:     strings.TrimSpace(v["label"]),
			StartDate: strings.TrimSpace(v["start_date"]),
			EndDate:   strings.TrimSpace(v["end_date"]),
			Mode:      strings.TrimSpace(v["mode"]),
		})
	}
	return nil
}

func (p *Panel) handleConfirmSubmit(ctx ui.PanelContext, tag string, payload any) tea.Cmd {
	switch tag {
	case "board.delete":
		b, _ := payload.(admin.BBSBoardDTO)
		return p.deleteBoardCmd(ctx.Client, b.ID)
	case "thread.archive":
		th, _ := payload.(admin.BBSThreadDTO)
		return p.archiveThreadCmd(ctx.Client, th.ID)
	case "category.delete":
		cat, _ := payload.(admin.BBSCategoryDTO)
		return p.deleteCategoryCmd(ctx.Client, cat.ID)
	case "ban.delete":
		b, _ := payload.(admin.BBSBanDTO)
		return p.deleteBanCmd(ctx.Client, b.ID)
	case "cap.delete":
		c, _ := payload.(admin.BBSCapDTO)
		return p.deleteCapCmd(ctx.Client, c.ID)
	case "slip.delete":
		ev, _ := payload.(admin.BBSSlipEventDTO)
		return p.deleteSlipEventCmd(ctx.Client, ev.ID)
	}
	return nil
}

func (p *Panel) View(width, height int) string {
	bodyH := max(height-2, 6)
	if p.width != width || p.height != height {
		p.width = width
		p.height = height
		p.resizeViewports(width, max(bodyH-1, 5))
		p.formModal.SetSize(width, height, p.theme)
	}

	switch {
	case p.formModal.Active:
		return ui.ClampBox(p.formModal.View(width, height, p.theme), width, height)
	case p.confirmModal.Active:
		return ui.ClampBox(p.confirmModal.View(width, height, p.theme), width, height)
	case p.selectModal.Active:
		return ui.ClampBox(p.selectModal.View(width, height, p.theme), width, height)
	}

	statsTxt := p.theme.MutedText.Render("loading stats...")
	if p.stats != nil {
		if !p.stats.Available {
			statsTxt = p.theme.BadgeWarn.Render("BBS Unavailable")
		} else {
			statsTxt = p.theme.MutedText.Render(fmt.Sprintf(
				"Cats:%d  Boards:%d  Threads:%d  Reports:%d  Bans:%d",
				p.stats.Categories, p.stats.Boards, p.stats.ActiveThreads, p.stats.OpenReports, p.stats.ActiveBans,
			))
		}
	}
	header := ui.TruncateWidth(p.theme.CardTitle.Render("BBS Management")+"  "+statsTxt, width)
	subTabBar := ui.RenderSubTabBar(p.theme, []string{"Reports", "Boards", "Categories", "Bans", "Caps", "SLIP"}, int(p.subTab), width)

	var body string
	switch p.subTab {
	case subTabReports:
		body = p.viewReports(width, bodyH)
	case subTabBoards:
		body = p.viewBoards(width, bodyH)
	case subTabCategories:
		body = p.viewCategories(width, bodyH)
	case subTabBans:
		body = p.viewBans(width, bodyH)
	case subTabCaps:
		body = p.viewCaps(width, bodyH)
	case subTabSlip:
		body = p.viewSlip(width, bodyH)
	}

	out := lipgloss.JoinVertical(lipgloss.Left, header, subTabBar, body)
	return ui.ClampBox(out, width, height)
}

func (p *Panel) splitDimensions(width, paneH int) (split bool, listW, detailW, innerH int) {
	split = width >= 88
	listW = width
	detailW = width
	innerH = max(paneH-4, 4)
	if split {
		listW = max(width*44/100, 32)
		detailW = max(width-listW-1, 30)
	} else {
		innerH = max((paneH/2)-3, 3)
	}
	return split, listW, detailW, innerH
}

func (p *Panel) resizeViewports(width, paneH int) {
	ui.SetInputWidth(&p.boards.searchInput, max(width-lipgloss.Width(threadSearchPrompt)-30, 12))
	_, _, detailW, innerH := p.splitDimensions(width, paneH)
	p.syncDetailViewport(&p.reports.detailVP, detailW, innerH)
	p.syncDetailViewport(&p.boards.detailVP, detailW, innerH)
	p.syncDetailViewport(&p.categories.detailVP, detailW, innerH)
	p.syncDetailViewport(&p.bans.detailVP, detailW, innerH)
	p.syncDetailViewport(&p.caps.detailVP, detailW, innerH)
	p.syncDetailViewport(&p.slip.detailVP, detailW, innerH)
	p.reports.rebuildDetail(p.theme)
	p.boards.rebuildDetail(p.theme)
	p.categories.rebuildDetail(p.theme)
	p.bans.rebuildDetail(p.theme)
	p.caps.rebuildDetail(p.theme)
	p.slip.rebuildDetail(p.theme)
}

func (p *Panel) syncDetailViewport(vp *viewport.Model, detailW, innerH int) {
	w := max(detailW-4, 16)
	h := max(innerH-1, 1)
	if vp.Width() != w || vp.Height() != h {
		vp.SetWidth(w)
		vp.SetHeight(h)
	}
}

func (p *Panel) joinPanes(split bool, listCard, detailCard string) string {
	if split {
		return lipgloss.JoinHorizontal(lipgloss.Top, listCard, " ", detailCard)
	}
	return lipgloss.JoinVertical(lipgloss.Left, listCard, detailCard)
}

func (p *Panel) renderEmptyCard(title, msg string, cardW, innerH int) string {
	return p.theme.Card.Width(cardW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render(title) + "\n" + p.theme.MutedText.Render(msg),
	)
}

func kvLine(theme ui.Theme, key, val string) string {
	return theme.Label.Render(ui.PadRightWidth(key+":", 15)) + " " + theme.Value.Render(val)
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func visibleWindow(selected, total, visible int) (int, int) {
	if visible <= 0 || total <= visible {
		return 0, total
	}
	start := 0
	if selected >= visible {
		start = selected - visible + 1
	}
	end := min(total, start+visible)
	return start, end
}

func restoreIdxByID(targetID string, n int, idAt func(int) string) int {
	if n <= 0 {
		return 0
	}
	if targetID != "" {
		for i := range n {
			if idAt(i) == targetID {
				return i
			}
		}
	}
	return 0
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
