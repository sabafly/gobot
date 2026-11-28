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

type bansState struct {
	items       []admin.BBSBanDTO
	selectedIdx int
	selectedID  string
	scopeIdx    int
	detailVP    viewport.Model
	loaded      bool
	loading     bool
	errMsg      string
}

func newBansState() bansState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return bansState{detailVP: vp}
}

func (s *bansState) currentScopeFilter() string {
	return scopeCycle[s.scopeIdx]
}

func (s *bansState) selected() (admin.BBSBanDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSBanDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *bansState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && len(s.items) == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}
	b, ok := s.selected()
	if !ok {
		msg := "No active bans found."
		if s.loading {
			msg = "Loading bans..."
		}
		s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
		return
	}
	guildStr := "(global)"
	if b.GuildID != nil && *b.GuildID != "" {
		guildStr = ui.FormatIDWithName(b.GuildName, *b.GuildID)
	}
	targetUID := "(hidden or unset)"
	if b.TargetUserID != nil && *b.TargetUserID != "" {
		targetUID = ui.FormatIDWithName(b.TargetUserName, *b.TargetUserID)
	}
	expStr := "Permanent"
	if b.ExpiresAt != nil {
		expStr = b.ExpiresAt.Local().Format("2006-01-02 15:04:05")
	}
	lines := []string{
		kvLine(theme, "Ban ID", b.ID),
		kvLine(theme, "Scope", b.Scope),
		kvLine(theme, "Guild", guildStr),
		kvLine(theme, "Poster Hash", b.PosterHash),
		kvLine(theme, "Target User", targetUID),
		kvLine(theme, "Expires", expStr),
		kvLine(theme, "Created By", ui.FormatIDWithName(b.CreatedByName, b.CreatedBy)),
		kvLine(theme, "Created", b.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		"",
		theme.CardTitle.Render("Reason"),
		b.Reason,
		"",
		theme.MutedText.Render("Moderator actions (PermBBSModerate): [n] Create Ban  [D] Lift Ban"),
	}
	s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	s.detailVP.GotoTop()
}

func (p *Panel) fetchBansCmd(client *adminclient.Client) tea.Cmd {
	scope := p.bans.currentScopeFilter()
	return func() tea.Msg {
		if client == nil {
			return bansLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSBans(reqCtx, admin.BBSBanFilter{Scope: scope, Limit: 200})
		return bansLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) createBanCmd(client *adminclient.Client, req admin.CreateBBSBanRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return banCreatedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.CreateBBSBan(reqCtx, req)
		return banCreatedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) deleteBanCmd(client *adminclient.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return banDeletedMsg{ID: id, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteBBSBan(reqCtx, id)
		return banDeletedMsg{ID: id, Err: err}
	}
}

func (p *Panel) updateBans(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "j", "down":
		if p.bans.selectedIdx < len(p.bans.items)-1 {
			p.bans.selectedIdx++
			p.bans.selectedID = p.bans.items[p.bans.selectedIdx].ID
			p.bans.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.bans.selectedIdx > 0 {
			p.bans.selectedIdx--
			p.bans.selectedID = p.bans.items[p.bans.selectedIdx].ID
			p.bans.rebuildDetail(p.theme)
		}
	case "g", "home":
		if len(p.bans.items) > 0 && p.bans.selectedIdx != 0 {
			p.bans.selectedIdx = 0
			p.bans.selectedID = p.bans.items[0].ID
			p.bans.rebuildDetail(p.theme)
		}
	case "G", "end":
		if len(p.bans.items) > 0 && p.bans.selectedIdx != len(p.bans.items)-1 {
			p.bans.selectedIdx = len(p.bans.items) - 1
			p.bans.selectedID = p.bans.items[p.bans.selectedIdx].ID
			p.bans.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.bans.detailVP.PageDown()
	case "K", "pgup":
		p.bans.detailVP.PageUp()
	case "s":
		p.bans.scopeIdx = (p.bans.scopeIdx + 1) % len(scopeCycle)
		p.bans.loading = true
		return p.fetchBansCmd(ctx.Client)
	case "r":
		p.bans.loading = true
		return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchBansCmd(ctx.Client))
	case "n":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		return p.openForm("ban.create", "Create Ban", "Specify Post ID, Target User ID, or Poster Hash", []ui.FormField{
			{Key: "scope", Label: "Scope", Kind: ui.FieldCycle, Options: []string{"local", "global"}, Value: "local"},
			{Key: "guild_id", Label: "Guild ID (local)", Kind: ui.FieldText, Placeholder: "Discord Guild ID", CharLimit: 32},
			{Key: "post_id", Label: "Post ID", Kind: ui.FieldText, Placeholder: "Optional Post UUID", CharLimit: 40},
			{Key: "target_user_id", Label: "Target UID", Kind: ui.FieldText, Placeholder: "Optional Discord User ID", CharLimit: 32},
			{Key: "target_poster_hash", Label: "Poster Hash", Kind: ui.FieldText, Placeholder: "Optional Poster Hash", CharLimit: 64},
			{Key: "duration_hours", Label: "Hours (0=perm)", Kind: ui.FieldText, Value: "24", CharLimit: 8},
			{Key: "reason", Label: "Reason", Kind: ui.FieldText, Placeholder: "Ban reason", CharLimit: 200},
		}, nil)
	case "D", "delete":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if b, ok := p.bans.selected(); ok {
			p.confirmModal.Open(
				"ban.delete",
				"Lift Ban",
				fmt.Sprintf("Lift %s ban %s (hash: %s)?", b.Scope, shortID(b.ID), ui.TruncateWidth(b.PosterHash, 16)),
				"Lift Ban",
				true,
				b,
			)
		}
	}
	return nil
}

func (p *Panel) viewBans(width, bodyH int) string {
	scopeLbl := p.bans.currentScopeFilter()
	if scopeLbl == "" {
		scopeLbl = "all"
	}
	toolbar := ui.TruncateWidth(
		p.theme.MutedText.Render(fmt.Sprintf("Scope: [%s] (s)  Active Bans: %d", scopeLbl, len(p.bans.items))),
		width,
	)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)

	var listCard string
	innerW := max(listW-4, 18)
	if len(p.bans.items) == 0 {
		msg := "No active bans found."
		if p.bans.errMsg != "" {
			msg = "Error: " + p.bans.errMsg
		} else if p.bans.loading {
			msg = "Loading bans..."
		}
		listCard = p.renderEmptyCard("Active Bans", msg, listW, innerH)
	} else {
		visibleRows := max(innerH-2, 1)
		start, end := visibleWindow(p.bans.selectedIdx, len(p.bans.items), visibleRows)
		reasonW := max(innerW-22, 8)
		head := p.theme.TableHeader.Render(
			ui.PadRightWidth("Scp", 6) + " " + ui.PadRightWidth("Hash", 14) + " " + ui.PadRightWidth("Reason", reasonW),
		)
		rows := []string{p.theme.CardTitle.Render("Active Bans"), head}
		for i := start; i < end; i++ {
			b := p.bans.items[i]
			line := ui.PadRightWidth(b.Scope, 6) + " " +
				ui.PadRightWidth(ui.TruncateWidth(b.PosterHash, 14), 14) + " " +
				ui.PadRightWidth(b.Reason, reasonW)
			st := p.theme.TableRow
			if i == p.bans.selectedIdx {
				st = p.theme.TableSelected
			}
			rows = append(rows, st.Render(line))
		}
		listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
	}

	p.syncDetailViewport(&p.bans.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render("Ban Detail") + "\n" + p.bans.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
