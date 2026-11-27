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

type configSection int

const (
	configSectionCaps configSection = iota
	configSectionSlipEvents
)

type configState struct {
	section      configSection
	caps         []admin.BBSCapDTO
	capIdx       int
	selectedCap  string
	slipEvents   []admin.BBSSlipEventDTO
	slipIdx      int
	selectedSlip string
	detailVP     viewport.Model
	loadedCaps   bool
	loadedSlip   bool
	loading      bool
	errMsg       string
}

func newConfigState() configState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return configState{detailVP: vp}
}

func (s *configState) currentListLen() int {
	if s.section == configSectionCaps {
		return len(s.caps)
	}
	return len(s.slipEvents)
}

func (s *configState) selectedCapItem() (admin.BBSCapDTO, bool) {
	if s.capIdx < 0 || s.capIdx >= len(s.caps) {
		return admin.BBSCapDTO{}, false
	}
	return s.caps[s.capIdx], true
}

func (s *configState) selectedSlipItem() (admin.BBSSlipEventDTO, bool) {
	if s.slipIdx < 0 || s.slipIdx >= len(s.slipEvents) {
		return admin.BBSSlipEventDTO{}, false
	}
	return s.slipEvents[s.slipIdx], true
}

func (s *configState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && s.currentListLen() == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}

	var lines []string
	if s.section == configSectionCaps {
		c, ok := s.selectedCapItem()
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
			usersStr = strings.Join(c.Users, ", ")
		}
		lines = []string{
			kvLine(theme, "Cap ID", c.ID),
			kvLine(theme, "Name", c.Name),
			kvLine(theme, "Grant Ops", fmt.Sprintf("%v", c.GrantOperators)),
			kvLine(theme, "Description", c.Description),
			kvLine(theme, "User IDs", usersStr),
		}
	} else {
		ev, ok := s.selectedSlipItem()
		if !ok {
			msg := "No SLIP events configured."
			if s.loading {
				msg = "Loading SLIP events..."
			}
			s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
			return
		}
		lines = []string{
			kvLine(theme, "Event ID", ev.ID),
			kvLine(theme, "Label", ev.Label),
			kvLine(theme, "Start Date", ev.StartDate),
			kvLine(theme, "End Date", ev.EndDate),
			kvLine(theme, "Mode", ev.Mode),
		}
	}
	lines = append(lines, "", theme.MutedText.Render("Switch section: [s]  Admin actions (PermBBSAdmin): [n] New  [e] Edit  [D] Delete"))
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

func (p *Panel) updateConfig(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "s":
		if p.config.section == configSectionCaps {
			p.config.section = configSectionSlipEvents
			if !p.config.loadedSlip && !p.config.loading {
				p.config.loading = true
				return p.fetchSlipEventsCmd(ctx.Client)
			}
		} else {
			p.config.section = configSectionCaps
			if !p.config.loadedCaps && !p.config.loading {
				p.config.loading = true
				return p.fetchCapsCmd(ctx.Client)
			}
		}
		p.config.rebuildDetail(p.theme)
	case "j", "down":
		if p.config.section == configSectionCaps {
			if p.config.capIdx < len(p.config.caps)-1 {
				p.config.capIdx++
				p.config.selectedCap = p.config.caps[p.config.capIdx].ID
				p.config.rebuildDetail(p.theme)
			}
		} else if p.config.slipIdx < len(p.config.slipEvents)-1 {
			p.config.slipIdx++
			p.config.selectedSlip = p.config.slipEvents[p.config.slipIdx].ID
			p.config.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.config.section == configSectionCaps {
			if p.config.capIdx > 0 {
				p.config.capIdx--
				p.config.selectedCap = p.config.caps[p.config.capIdx].ID
				p.config.rebuildDetail(p.theme)
			}
		} else if p.config.slipIdx > 0 {
			p.config.slipIdx--
			p.config.selectedSlip = p.config.slipEvents[p.config.slipIdx].ID
			p.config.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.config.detailVP.PageDown()
	case "K", "pgup":
		p.config.detailVP.PageUp()
	case "r":
		p.config.loading = true
		if p.config.section == configSectionCaps {
			return p.fetchCapsCmd(ctx.Client)
		}
		return p.fetchSlipEventsCmd(ctx.Client)
	case "n":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if p.config.section == configSectionCaps {
			return p.formModal.Open("cap.create", "Create Admin Cap", "Persisted live in DB (no restart needed)", []FormField{
				{Key: "id", Label: "Cap ID", Kind: FieldText, Placeholder: "admin", CharLimit: 64},
				{Key: "name", Label: "Display Name", Kind: FieldText, Placeholder: "管理人 ★", CharLimit: 48},
				{Key: "description", Label: "Description", Kind: FieldText, Placeholder: "Usage description", CharLimit: 100},
				{Key: "grant_operators", Label: "Grant Ops", Kind: FieldBool, BoolVal: true},
				{Key: "users", Label: "Users (CSV)", Kind: FieldText, Placeholder: "111111, 222222", CharLimit: 512},
			}, nil)
		}
		return p.formModal.Open("slip.create", "Create SLIP Event", "Dates: MM-DD or YYYY-MM-DD (JST)", []FormField{
			{Key: "id", Label: "Event ID", Kind: FieldText, Placeholder: "winter_fest", CharLimit: 64},
			{Key: "label", Label: "Half-Kana Label", Kind: FieldText, Placeholder: "ﾌﾕﾏﾂﾘ", CharLimit: 24},
			{Key: "start_date", Label: "Start Date", Kind: FieldText, Placeholder: "MM-DD or YYYY-MM-DD", CharLimit: 10},
			{Key: "end_date", Label: "End Date", Kind: FieldText, Placeholder: "MM-DD or YYYY-MM-DD", CharLimit: 10},
			{Key: "mode", Label: "Mode", Kind: FieldCycle, Options: []string{"override", "suffix"}, Value: "override"},
		}, nil)
	case "e":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if p.config.section == configSectionCaps {
			if c, ok := p.config.selectedCapItem(); ok {
				return p.formModal.Open("cap.edit", "Edit Admin Cap", c.ID, []FormField{
					{Key: "id", Label: "Cap ID", Kind: FieldText, Value: c.ID, ReadOnly: true},
					{Key: "name", Label: "Display Name", Kind: FieldText, Value: c.Name, CharLimit: 48},
					{Key: "description", Label: "Description", Kind: FieldText, Value: c.Description, CharLimit: 100},
					{Key: "grant_operators", Label: "Grant Ops", Kind: FieldBool, BoolVal: c.GrantOperators},
					{Key: "users", Label: "Users (CSV)", Kind: FieldText, Value: strings.Join(c.Users, ", "), CharLimit: 512},
				}, c)
			}
		} else if ev, ok := p.config.selectedSlipItem(); ok {
			return p.formModal.Open("slip.edit", "Edit SLIP Event", ev.ID, []FormField{
				{Key: "id", Label: "Event ID", Kind: FieldText, Value: ev.ID, ReadOnly: true},
				{Key: "label", Label: "Half-Kana Label", Kind: FieldText, Value: ev.Label, CharLimit: 24},
				{Key: "start_date", Label: "Start Date", Kind: FieldText, Value: ev.StartDate, CharLimit: 10},
				{Key: "end_date", Label: "End Date", Kind: FieldText, Value: ev.EndDate, CharLimit: 10},
				{Key: "mode", Label: "Mode", Kind: FieldCycle, Options: []string{"override", "suffix"}, Value: ev.Mode},
			}, ev)
		}
	case "D":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if p.config.section == configSectionCaps {
			if c, ok := p.config.selectedCapItem(); ok {
				p.confirmModal.Open("cap.delete", "Delete Admin Cap", fmt.Sprintf("Delete cap %q (%s)?", c.Name, c.ID), "Delete", true, c)
			}
		} else if ev, ok := p.config.selectedSlipItem(); ok {
			p.confirmModal.Open("slip.delete", "Delete SLIP Event", fmt.Sprintf("Delete SLIP event %q (%s)?", ev.Label, ev.ID), "Delete", true, ev)
		}
	}
	return nil
}

func (p *Panel) viewConfig(width, bodyH int) string {
	capsTab := p.theme.SubTab.Render("Admin Caps")
	slipTab := p.theme.SubTab.Render("SLIP Events")
	if p.config.section == configSectionCaps {
		capsTab = p.theme.SubTabActive.Render("Admin Caps")
	} else {
		slipTab = p.theme.SubTabActive.Render("SLIP Events")
	}
	toolbar := ui.TruncateWidth(capsTab+" "+slipTab+"  "+p.theme.MutedText.Render("[s] switch section"), width)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)
	innerW := max(listW-4, 18)

	var listCard string
	if p.config.section == configSectionCaps {
		if len(p.config.caps) == 0 {
			msg := "No admin caps configured."
			if p.config.errMsg != "" {
				msg = "Error: " + p.config.errMsg
			} else if p.config.loading {
				msg = "Loading admin caps..."
			}
			listCard = p.renderEmptyCard("Admin Caps", msg, listW, innerH)
		} else {
			visibleRows := max(innerH-2, 1)
			start, end := visibleWindow(p.config.capIdx, len(p.config.caps), visibleRows)
			nameW := max(innerW-16, 10)
			head := p.theme.TableHeader.Render(
				ui.PadRightWidth("ID", 10) + " " + ui.PadRightWidth("Ops", 4) + " " + ui.PadRightWidth("Name", nameW),
			)
			rows := []string{p.theme.CardTitle.Render("Admin Caps"), head}
			for i := start; i < end; i++ {
				c := p.config.caps[i]
				ops := "no"
				if c.GrantOperators {
					ops = "yes"
				}
				line := ui.PadRightWidth(c.ID, 10) + " " + ui.PadRightWidth(ops, 4) + " " + ui.PadRightWidth(c.Name, nameW)
				st := p.theme.TableRow
				if i == p.config.capIdx {
					st = p.theme.TableSelected
				}
				rows = append(rows, st.Render(line))
			}
			listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
		}
	} else {
		if len(p.config.slipEvents) == 0 {
			msg := "No SLIP events configured."
			if p.config.errMsg != "" {
				msg = "Error: " + p.config.errMsg
			} else if p.config.loading {
				msg = "Loading SLIP events..."
			}
			listCard = p.renderEmptyCard("SLIP Events", msg, listW, innerH)
		} else {
			visibleRows := max(innerH-2, 1)
			start, end := visibleWindow(p.config.slipIdx, len(p.config.slipEvents), visibleRows)
			lblW := max(innerW-22, 10)
			head := p.theme.TableHeader.Render(
				ui.PadRightWidth("ID", 10) + " " + ui.PadRightWidth("Range", 11) + " " + ui.PadRightWidth("Label", lblW),
			)
			rows := []string{p.theme.CardTitle.Render("SLIP Events"), head}
			for i := start; i < end; i++ {
				ev := p.config.slipEvents[i]
				rng := ev.StartDate + ".." + ev.EndDate
				line := ui.PadRightWidth(ev.ID, 10) + " " + ui.PadRightWidth(rng, 11) + " " + ui.PadRightWidth(ev.Label, lblW)
				st := p.theme.TableRow
				if i == p.config.slipIdx {
					st = p.theme.TableSelected
				}
				rows = append(rows, st.Render(line))
			}
			listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
		}
	}

	p.syncDetailViewport(&p.config.detailVP, detailW, innerH)
	detailTitle := "Cap Detail"
	if p.config.section == configSectionSlipEvents {
		detailTitle = "SLIP Event Detail"
	}
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render(detailTitle) + "\n" + p.config.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
