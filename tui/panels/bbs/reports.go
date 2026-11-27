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
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

var (
	reportStatusCycle = []string{"open", "", "resolved", "dismissed"}
	scopeCycle        = []string{"", "local", "global"}
)

type reportsState struct {
	items       []admin.BBSReportDTO
	selectedIdx int
	selectedID  string
	statusIdx   int
	scopeIdx    int
	inspect     *admin.BBSReportInspectResponse
	detailVP    viewport.Model
	loaded      bool
	loading     bool
	inspecting  bool
	mutating    bool
	errMsg      string
}

func newReportsState() reportsState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return reportsState{detailVP: vp}
}

func (s *reportsState) currentStatusFilter() string {
	return reportStatusCycle[s.statusIdx]
}

func (s *reportsState) currentScopeFilter() string {
	return scopeCycle[s.scopeIdx]
}

func (s *reportsState) selected() (admin.BBSReportDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSReportDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *reportsState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && len(s.items) == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}
	rep, ok := s.selected()
	if !ok {
		msg := "No reports match filter."
		if s.loading {
			msg = "Loading reports..."
		}
		s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
		return
	}

	lines := []string{
		kvLine(theme, "Report ID", rep.ID),
		kvLine(theme, "Status", rep.Status),
		kvLine(theme, "Scope", rep.Scope),
		kvLine(theme, "Guild ID", rep.GuildID),
		kvLine(theme, "Board ID", rep.BoardID),
		kvLine(theme, "Thread ID", rep.ThreadID),
		kvLine(theme, "Post ID", rep.PostID),
		kvLine(theme, "Reporter ID", rep.ReporterUserID),
		kvLine(theme, "Created", rep.CreatedAt.Local().Format("2006-01-02 15:04:05")),
	}
	if rep.HandledBy != nil {
		lines = append(lines, kvLine(theme, "Handled By", *rep.HandledBy))
	}
	if rep.ResolutionNote != "" {
		lines = append(lines, kvLine(theme, "Resolution", rep.ResolutionNote))
	}
	lines = append(lines, "", theme.CardTitle.Render("Report Reason"), rep.Reason)

	if s.inspecting {
		lines = append(lines, "", theme.BadgeInfo.Render("Inspecting report attribution..."))
	} else if s.inspect != nil && s.inspect.Report.ID == rep.ID {
		ins := s.inspect
		lines = append(lines, "", theme.CardTitle.Render("Operator Inspection (Audited)"))
		if ins.AuthorUserID != "" {
			lines = append(lines, kvLine(theme, "Author UID", ins.AuthorUserID))
		}
		if ins.PosterHash != "" {
			lines = append(lines, kvLine(theme, "Poster Hash", ins.PosterHash))
		}
		if ins.PostGuildID != "" {
			lines = append(lines, kvLine(theme, "Origin Guild", ins.PostGuildID))
		}
		if ins.Post != nil {
			p := ins.Post
			lines = append(lines,
				kvLine(theme, "Res #", fmt.Sprintf("%d (Aboned=%v)", p.ResNumber, p.Abone)),
				kvLine(theme, "Post Name", fmt.Sprintf("%s %s %s %s", p.Name, p.Trip, p.Cap, p.SlipText)),
				theme.Label.Render("Post Body:"),
				p.Body,
			)
		}
	} else {
		lines = append(lines, "", theme.MutedText.Render("Press [i] to inspect author attribution (PermBBSModerate, audited)."))
	}

	s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	s.detailVP.GotoTop()
}

func (p *Panel) fetchReportsCmd(client *adminclient.Client) tea.Cmd {
	status := p.reports.currentStatusFilter()
	scope := p.reports.currentScopeFilter()
	return func() tea.Msg {
		if client == nil {
			return reportsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSReports(reqCtx, admin.BBSReportFilter{
			Status: status,
			Scope:  scope,
			Limit:  100,
		})
		return reportsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) inspectReportCmd(client *adminclient.Client, reportID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return reportInspectedMsg{ReportID: reportID, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.InspectBBSReport(reqCtx, reportID)
		return reportInspectedMsg{ReportID: reportID, Resp: resp, Err: err}
	}
}

func (p *Panel) resolveReportCmd(client *adminclient.Client, reportID, status, note string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return reportResolvedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ResolveBBSReport(reqCtx, reportID, admin.ResolveBBSReportRequest{
			Status:         status,
			ResolutionNote: note,
		})
		return reportResolvedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) updateReports(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "j", "down":
		if p.reports.selectedIdx < len(p.reports.items)-1 {
			p.reports.selectedIdx++
			p.reports.selectedID = p.reports.items[p.reports.selectedIdx].ID
			p.reports.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.reports.selectedIdx > 0 {
			p.reports.selectedIdx--
			p.reports.selectedID = p.reports.items[p.reports.selectedIdx].ID
			p.reports.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.reports.detailVP.PageDown()
	case "K", "pgup":
		p.reports.detailVP.PageUp()
	case "f":
		p.reports.statusIdx = (p.reports.statusIdx + 1) % len(reportStatusCycle)
		p.reports.loading = true
		return p.fetchReportsCmd(ctx.Client)
	case "s":
		p.reports.scopeIdx = (p.reports.scopeIdx + 1) % len(scopeCycle)
		p.reports.loading = true
		return p.fetchReportsCmd(ctx.Client)
	case "r":
		p.reports.loading = true
		return p.fetchReportsCmd(ctx.Client)
	case "i":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if rep, ok := p.reports.selected(); ok {
			p.reports.inspecting = true
			p.reports.rebuildDetail(p.theme)
			return p.inspectReportCmd(ctx.Client, rep.ID)
		}
	case "v":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if rep, ok := p.reports.selected(); ok {
			return p.formModal.Open("report.resolve", "Resolve Report", "Report "+shortID(rep.ID), []FormField{
				{Key: "status", Label: "Status", Kind: FieldCycle, Options: []string{"resolved", "dismissed"}, Value: "resolved"},
				{Key: "note", Label: "Resolution Note", Kind: FieldText, Placeholder: "Action taken / note", CharLimit: 240},
			}, rep.ID)
		}
	case "d":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if rep, ok := p.reports.selected(); ok {
			return p.formModal.Open("report.dismiss", "Dismiss Report", "Report "+shortID(rep.ID), []FormField{
				{Key: "note", Label: "Resolution Note", Kind: FieldText, Placeholder: "Dismissal reason", CharLimit: 240},
			}, rep.ID)
		}
	case "a":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if rep, ok := p.reports.selected(); ok {
			return p.formModal.Open("report.abone", "Abone Reported Post", "Post "+shortID(rep.PostID), []FormField{
				{Key: "reason", Label: "Abone Reason", Kind: FieldText, Value: "モデレーター対応", CharLimit: 120},
			}, rep.PostID)
		}
	case "b":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if rep, ok := p.reports.selected(); ok {
			return p.formModal.Open("report.ban", "Ban Reported Poster", "Post "+shortID(rep.PostID), []FormField{
				{Key: "scope", Label: "Scope", Kind: FieldCycle, Options: []string{"local", "global"}, Value: rep.Scope},
				{Key: "guild_id", Label: "Guild ID", Kind: FieldText, Value: rep.GuildID, CharLimit: 32},
				{Key: "post_id", Label: "Post ID", Kind: FieldText, Value: rep.PostID, ReadOnly: true},
				{Key: "duration_hours", Label: "Hours (0=perm)", Kind: FieldText, Value: "24", CharLimit: 8},
				{Key: "reason", Label: "Reason", Kind: FieldText, Value: rep.Reason, CharLimit: 200},
			}, nil)
		}
	}
	return nil
}

func (p *Panel) viewReports(width, bodyH int) string {
	statusLbl := p.reports.currentStatusFilter()
	if statusLbl == "" {
		statusLbl = "all"
	}
	scopeLbl := p.reports.currentScopeFilter()
	if scopeLbl == "" {
		scopeLbl = "all"
	}
	toolbar := ui.TruncateWidth(
		p.theme.MutedText.Render(fmt.Sprintf("Status: [%s] (f)  Scope: [%s] (s)  Count: %d", statusLbl, scopeLbl, len(p.reports.items))),
		width,
	)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)

	var listCard string
	innerW := max(listW-4, 18)
	if len(p.reports.items) == 0 {
		msg := "No reports found."
		if p.reports.errMsg != "" {
			msg = "Error: " + p.reports.errMsg
		} else if p.reports.loading {
			msg = "Loading reports..."
		}
		listCard = p.renderEmptyCard("Reports", msg, listW, innerH)
	} else {
		visibleRows := max(innerH-2, 1)
		start, end := visibleWindow(p.reports.selectedIdx, len(p.reports.items), visibleRows)
		reasonW := max(innerW-22, 10)
		head := p.theme.TableHeader.Render(
			ui.PadRightWidth("ID", 8) + " " + ui.PadRightWidth("Stat", 5) + " " + ui.PadRightWidth("Scp", 5) + " " + ui.PadRightWidth("Reason", reasonW),
		)
		rows := []string{p.theme.CardTitle.Render("Reports"), head}
		for i := start; i < end; i++ {
			rep := p.reports.items[i]
			line := ui.PadRightWidth(shortID(rep.ID), 8) + " " +
				ui.PadRightWidth(rep.Status, 5) + " " +
				ui.PadRightWidth(rep.Scope, 5) + " " +
				ui.PadRightWidth(rep.Reason, reasonW)
			st := p.theme.TableRow
			if i == p.reports.selectedIdx {
				st = p.theme.TableSelected
			}
			rows = append(rows, st.Render(line))
		}
		listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
	}

	p.syncDetailViewport(&p.reports.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render("Report Detail") + "\n" + p.reports.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
