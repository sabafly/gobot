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

type capsState struct {
	items       []admin.BBSCapDTO
	selectedIdx int
	selectedID  string
	detailVP    viewport.Model
	loaded      bool
	loading     bool
	errMsg      string
}

type slipState struct {
	items       []admin.BBSSlipEventDTO
	selectedIdx int
	selectedID  string
	detailVP    viewport.Model
	loaded      bool
	loading     bool
	errMsg      string
}

func newCapsState() capsState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return capsState{detailVP: vp}
}

func newSlipState() slipState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return slipState{detailVP: vp}
}

func (s *capsState) selected() (admin.BBSCapDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSCapDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *slipState) selected() (admin.BBSSlipEventDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSSlipEventDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *capsState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && len(s.items) == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}
	c, ok := s.selected()
	if !ok {
		msg := "No admin caps configured."
		if s.loading {
			msg = "Loading admin caps..."
		}
		s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
		return
	}
	usersStr := "(none)"
	if len(c.Users) > 0 {
		formatted := make([]string, 0, len(c.Users))
		for _, u := range c.Users {
			formatted = append(formatted, ui.FormatIDWithName(c.UserNames[u], u))
		}
		usersStr = strings.Join(formatted, ", ")
	}
	lines := []string{
		kvLine(theme, "Cap ID", c.ID),
		kvLine(theme, "Name", c.Name),
		kvLine(theme, "Grant Ops", fmt.Sprintf("%v", c.GrantOperators)),
		kvLine(theme, "Description", c.Description),
		kvLine(theme, "Users", usersStr),
		"",
		theme.MutedText.Render("Admin actions (PermBBSAdmin): [n] New  [e] Edit  [D] Delete"),
	}
	s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	s.detailVP.GotoTop()
}

func (s *slipState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && len(s.items) == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}
	ev, ok := s.selected()
	if !ok {
		msg := "No SLIP events configured."
		if s.loading {
			msg = "Loading SLIP events..."
		}
		s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
		return
	}
	lines := []string{
		kvLine(theme, "Event ID", ev.ID),
		kvLine(theme, "Label", ev.Label),
		kvLine(theme, "Start Date", ev.StartDate),
		kvLine(theme, "End Date", ev.EndDate),
		kvLine(theme, "Mode", ev.Mode),
		"",
		theme.MutedText.Render("Admin actions (PermBBSAdmin): [n] New  [e] Edit  [D] Delete"),
	}
	s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	s.detailVP.GotoTop()
}

func (p *Panel) fetchCapsCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return capsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSCaps(reqCtx)
		return capsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) saveCapCmd(client *adminclient.Client, created bool, id string, req admin.SaveBBSCapRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return capSavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var (
			resp admin.BBSCapDTO
			err  error
		)
		if created {
			resp, err = client.CreateBBSCap(reqCtx, req)
		} else {
			resp, err = client.UpdateBBSCap(reqCtx, id, req)
		}
		return capSavedMsg{Created: created, Resp: resp, Err: err}
	}
}

func (p *Panel) deleteCapCmd(client *adminclient.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return capDeletedMsg{ID: id, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteBBSCap(reqCtx, id)
		return capDeletedMsg{ID: id, Err: err}
	}
}

func (p *Panel) fetchSlipEventsCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return slipEventsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSSlipEvents(reqCtx)
		return slipEventsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) saveSlipEventCmd(client *adminclient.Client, created bool, id string, req admin.SaveBBSSlipEventRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return slipEventSavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var (
			resp admin.BBSSlipEventDTO
			err  error
		)
		if created {
			resp, err = client.CreateBBSSlipEvent(reqCtx, req)
		} else {
			resp, err = client.UpdateBBSSlipEvent(reqCtx, id, req)
		}
		return slipEventSavedMsg{Created: created, Resp: resp, Err: err}
	}
}

func (p *Panel) deleteSlipEventCmd(client *adminclient.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return slipEventDeletedMsg{ID: id, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteBBSSlipEvent(reqCtx, id)
		return slipEventDeletedMsg{ID: id, Err: err}
	}
}

func (p *Panel) updateCaps(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "j", "down":
		if p.caps.selectedIdx < len(p.caps.items)-1 {
			p.caps.selectedIdx++
			p.caps.selectedID = p.caps.items[p.caps.selectedIdx].ID
			p.caps.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.caps.selectedIdx > 0 {
			p.caps.selectedIdx--
			p.caps.selectedID = p.caps.items[p.caps.selectedIdx].ID
			p.caps.rebuildDetail(p.theme)
		}
	case "g", "home":
		if len(p.caps.items) > 0 && p.caps.selectedIdx != 0 {
			p.caps.selectedIdx = 0
			p.caps.selectedID = p.caps.items[0].ID
			p.caps.rebuildDetail(p.theme)
		}
	case "G", "end":
		if len(p.caps.items) > 0 && p.caps.selectedIdx != len(p.caps.items)-1 {
			p.caps.selectedIdx = len(p.caps.items) - 1
			p.caps.selectedID = p.caps.items[p.caps.selectedIdx].ID
			p.caps.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.caps.detailVP.PageDown()
	case "K", "pgup":
		p.caps.detailVP.PageUp()
	case "r":
		p.caps.loading = true
		return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchCapsCmd(ctx.Client))
	case "n":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		return p.openForm("cap.create", "Create Admin Cap", "Persisted live in DB (no restart needed)", []ui.FormField{
			{Key: "id", Label: "Cap ID", Kind: ui.FieldText, Placeholder: "admin", CharLimit: 64},
			{Key: "name", Label: "Display Name", Kind: ui.FieldText, Placeholder: "管理人 ★", CharLimit: 48},
			{Key: "description", Label: "Description", Kind: ui.FieldText, Placeholder: "Usage description", CharLimit: 100},
			{Key: "grant_operators", Label: "Grant Ops", Kind: ui.FieldBool, BoolVal: true},
			{Key: "users", Label: "Users (CSV)", Kind: ui.FieldText, Placeholder: "111111, 222222", CharLimit: 512},
		}, nil)
	case "e":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if c, ok := p.caps.selected(); ok {
			return p.openForm("cap.edit", "Edit Admin Cap", c.ID, []ui.FormField{
				{Key: "id", Label: "Cap ID", Kind: ui.FieldText, Value: c.ID, ReadOnly: true},
				{Key: "name", Label: "Display Name", Kind: ui.FieldText, Value: c.Name, CharLimit: 48},
				{Key: "description", Label: "Description", Kind: ui.FieldText, Value: c.Description, CharLimit: 100},
				{Key: "grant_operators", Label: "Grant Ops", Kind: ui.FieldBool, BoolVal: c.GrantOperators},
				{Key: "users", Label: "Users (CSV)", Kind: ui.FieldText, Value: strings.Join(c.Users, ", "), CharLimit: 512},
			}, c)
		}
	case "D", "delete":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if c, ok := p.caps.selected(); ok {
			p.confirmModal.Open("cap.delete", "Delete Admin Cap", fmt.Sprintf("Delete cap %q (%s)?", c.Name, c.ID), "Delete", true, c)
		}
	}
	return nil
}

func (p *Panel) updateSlip(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "j", "down":
		if p.slip.selectedIdx < len(p.slip.items)-1 {
			p.slip.selectedIdx++
			p.slip.selectedID = p.slip.items[p.slip.selectedIdx].ID
			p.slip.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.slip.selectedIdx > 0 {
			p.slip.selectedIdx--
			p.slip.selectedID = p.slip.items[p.slip.selectedIdx].ID
			p.slip.rebuildDetail(p.theme)
		}
	case "g", "home":
		if len(p.slip.items) > 0 && p.slip.selectedIdx != 0 {
			p.slip.selectedIdx = 0
			p.slip.selectedID = p.slip.items[0].ID
			p.slip.rebuildDetail(p.theme)
		}
	case "G", "end":
		if len(p.slip.items) > 0 && p.slip.selectedIdx != len(p.slip.items)-1 {
			p.slip.selectedIdx = len(p.slip.items) - 1
			p.slip.selectedID = p.slip.items[p.slip.selectedIdx].ID
			p.slip.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.slip.detailVP.PageDown()
	case "K", "pgup":
		p.slip.detailVP.PageUp()
	case "r":
		p.slip.loading = true
		return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchSlipEventsCmd(ctx.Client))
	case "n":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		return p.openForm("slip.create", "Create SLIP Event", "Dates: MM-DD or YYYY-MM-DD (JST)", []ui.FormField{
			{Key: "id", Label: "Event ID", Kind: ui.FieldText, Placeholder: "winter_fest", CharLimit: 64},
			{Key: "label", Label: "Half-Kana Label", Kind: ui.FieldText, Placeholder: "ﾌﾕﾏﾂﾘ", CharLimit: 24},
			{Key: "start_date", Label: "Start Date", Kind: ui.FieldText, Placeholder: "MM-DD or YYYY-MM-DD", CharLimit: 10},
			{Key: "end_date", Label: "End Date", Kind: ui.FieldText, Placeholder: "MM-DD or YYYY-MM-DD", CharLimit: 10},
			{Key: "mode", Label: "Mode", Kind: ui.FieldCycle, Options: []string{"override", "suffix"}, Value: "override"},
		}, nil)
	case "e":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if ev, ok := p.slip.selected(); ok {
			return p.openForm("slip.edit", "Edit SLIP Event", ev.ID, []ui.FormField{
				{Key: "id", Label: "Event ID", Kind: ui.FieldText, Value: ev.ID, ReadOnly: true},
				{Key: "label", Label: "Half-Kana Label", Kind: ui.FieldText, Value: ev.Label, CharLimit: 24},
				{Key: "start_date", Label: "Start Date", Kind: ui.FieldText, Value: ev.StartDate, CharLimit: 10},
				{Key: "end_date", Label: "End Date", Kind: ui.FieldText, Value: ev.EndDate, CharLimit: 10},
				{Key: "mode", Label: "Mode", Kind: ui.FieldCycle, Options: []string{"override", "suffix"}, Value: ev.Mode},
			}, ev)
		}
	case "D", "delete":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if ev, ok := p.slip.selected(); ok {
			p.confirmModal.Open("slip.delete", "Delete SLIP Event", fmt.Sprintf("Delete SLIP event %q (%s)?", ev.Label, ev.ID), "Delete", true, ev)
		}
	}
	return nil
}

func (p *Panel) viewCaps(width, bodyH int) string {
	toolbar := ui.TruncateWidth(
		p.theme.MutedText.Render(fmt.Sprintf("Admin Caps: %d  (live DB runtime caps)", len(p.caps.items))),
		width,
	)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)
	innerW := max(listW-4, 18)

	var listCard string
	if len(p.caps.items) == 0 {
		msg := "No admin caps configured."
		if p.caps.errMsg != "" {
			msg = "Error: " + p.caps.errMsg
		} else if p.caps.loading {
			msg = "Loading admin caps..."
		}
		listCard = p.renderEmptyCard("Admin Caps", msg, listW, innerH)
	} else {
		visibleRows := max(innerH-2, 1)
		start, end := visibleWindow(p.caps.selectedIdx, len(p.caps.items), visibleRows)
		nameW := max(innerW-16, 10)
		head := p.theme.TableHeader.Render(
			ui.PadRightWidth("ID", 10) + " " + ui.PadRightWidth("Ops", 4) + " " + ui.PadRightWidth("Name", nameW),
		)
		rows := []string{p.theme.CardTitle.Render("Admin Caps"), head}
		for i := start; i < end; i++ {
			c := p.caps.items[i]
			ops := "no"
			if c.GrantOperators {
				ops = "yes"
			}
			line := ui.PadRightWidth(c.ID, 10) + " " + ui.PadRightWidth(ops, 4) + " " + ui.PadRightWidth(c.Name, nameW)
			st := p.theme.TableRow
			if i == p.caps.selectedIdx {
				st = p.theme.TableSelected
			}
			rows = append(rows, st.Render(line))
		}
		listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
	}

	p.syncDetailViewport(&p.caps.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render("Cap Detail") + "\n" + p.caps.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}

func (p *Panel) viewSlip(width, bodyH int) string {
	toolbar := ui.TruncateWidth(
		p.theme.MutedText.Render(fmt.Sprintf("SLIP Events: %d  (seasonal overrides, JST)", len(p.slip.items))),
		width,
	)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)
	innerW := max(listW-4, 18)

	var listCard string
	if len(p.slip.items) == 0 {
		msg := "No SLIP events configured."
		if p.slip.errMsg != "" {
			msg = "Error: " + p.slip.errMsg
		} else if p.slip.loading {
			msg = "Loading SLIP events..."
		}
		listCard = p.renderEmptyCard("SLIP Events", msg, listW, innerH)
	} else {
		visibleRows := max(innerH-2, 1)
		start, end := visibleWindow(p.slip.selectedIdx, len(p.slip.items), visibleRows)
		lblW := max(innerW-22, 10)
		head := p.theme.TableHeader.Render(
			ui.PadRightWidth("ID", 10) + " " + ui.PadRightWidth("Range", 11) + " " + ui.PadRightWidth("Label", lblW),
		)
		rows := []string{p.theme.CardTitle.Render("SLIP Events"), head}
		for i := start; i < end; i++ {
			ev := p.slip.items[i]
			rng := ev.StartDate + ".." + ev.EndDate
			line := ui.PadRightWidth(ev.ID, 10) + " " + ui.PadRightWidth(rng, 11) + " " + ui.PadRightWidth(ev.Label, lblW)
			st := p.theme.TableRow
			if i == p.slip.selectedIdx {
				st = p.theme.TableSelected
			}
			rows = append(rows, st.Render(line))
		}
		listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
	}

	p.syncDetailViewport(&p.slip.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render("SLIP Event Detail") + "\n" + p.slip.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
