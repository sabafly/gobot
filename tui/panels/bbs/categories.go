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

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

type categoriesState struct {
	items       []admin.BBSCategoryDTO
	selectedIdx int
	selectedID  string
	scopeIdx    int
	detailVP    viewport.Model
	loaded      bool
	loading     bool
	mutating    bool
	errMsg      string
}

func newCategoriesState() categoriesState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return categoriesState{detailVP: vp}
}

func (s *categoriesState) currentScopeFilter() string {
	return scopeCycle[s.scopeIdx]
}

func (s *categoriesState) selected() (admin.BBSCategoryDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSCategoryDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *categoriesState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	if s.errMsg != "" && len(s.items) == 0 {
		s.detailVP.SetContent(ui.WrapViewportText(theme.BadgeErr.Render("Error: "+s.errMsg), wrapW))
		return
	}
	cat, ok := s.selected()
	if !ok {
		msg := "No categories found."
		if s.loading {
			msg = "Loading categories..."
		}
		s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
		return
	}
	guildStr := "(global)"
	if cat.GuildID != nil && *cat.GuildID != "" {
		guildStr = *cat.GuildID
	}
	lines := []string{
		kvLine(theme, "Category ID", cat.ID),
		kvLine(theme, "Name", cat.Name),
		kvLine(theme, "Slug", cat.Slug),
		kvLine(theme, "Scope", cat.Scope),
		kvLine(theme, "Guild ID", guildStr),
		kvLine(theme, "Provider", cat.ProviderKey),
		kvLine(theme, "Sort Order", fmt.Sprintf("%d", cat.SortOrder)),
		kvLine(theme, "Created", cat.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		"",
		theme.CardTitle.Render("Description"),
		cat.Description,
		"",
		theme.MutedText.Render("Admin actions (PermBBSAdmin): [n] New  [e] Edit  [D] Delete"),
	}
	s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	s.detailVP.GotoTop()
}

func (p *Panel) fetchCategoriesCmd(client *adminclient.Client) tea.Cmd {
	scope := p.categories.currentScopeFilter()
	return func() tea.Msg {
		if client == nil {
			return categoriesLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSCategories(reqCtx, admin.BBSCategoryFilter{Scope: scope})
		return categoriesLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) createCategoryCmd(client *adminclient.Client, req admin.CreateBBSCategoryRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return categorySavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.CreateBBSCategory(reqCtx, req)
		return categorySavedMsg{Created: true, Resp: resp, Err: err}
	}
}

func (p *Panel) updateCategoryCmd(client *adminclient.Client, id string, req admin.UpdateBBSCategoryRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return categorySavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.UpdateBBSCategory(reqCtx, id, req)
		return categorySavedMsg{Created: false, Resp: resp, Err: err}
	}
}

func (p *Panel) deleteCategoryCmd(client *adminclient.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return categoryDeletedMsg{ID: id, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteBBSCategory(reqCtx, id)
		return categoryDeletedMsg{ID: id, Err: err}
	}
}

func (p *Panel) updateCategories(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "j", "down":
		if p.categories.selectedIdx < len(p.categories.items)-1 {
			p.categories.selectedIdx++
			p.categories.selectedID = p.categories.items[p.categories.selectedIdx].ID
			p.categories.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.categories.selectedIdx > 0 {
			p.categories.selectedIdx--
			p.categories.selectedID = p.categories.items[p.categories.selectedIdx].ID
			p.categories.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.categories.detailVP.PageDown()
	case "K", "pgup":
		p.categories.detailVP.PageUp()
	case "s":
		p.categories.scopeIdx = (p.categories.scopeIdx + 1) % len(scopeCycle)
		p.categories.loading = true
		return p.fetchCategoriesCmd(ctx.Client)
	case "r":
		p.categories.loading = true
		return p.fetchCategoriesCmd(ctx.Client)
	case "n":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		return p.formModal.Open("category.create", "Create Category", "Create a new global or local BBS category", []FormField{
			{Key: "scope", Label: "Scope", Kind: FieldCycle, Options: []string{"global", "local"}, Value: "global"},
			{Key: "guild_id", Label: "Guild ID (local)", Kind: FieldText, Placeholder: "Discord Guild ID", CharLimit: 32},
			{Key: "slug", Label: "Slug", Kind: FieldText, Placeholder: "news", CharLimit: 64},
			{Key: "name", Label: "Name", Kind: FieldText, Placeholder: "ニュース速報", CharLimit: 64},
			{Key: "description", Label: "Description", Kind: FieldText, Placeholder: "Optional summary", CharLimit: 256},
			{Key: "sort_order", Label: "Sort Order", Kind: FieldText, Value: "0", CharLimit: 8},
		}, nil)
	case "e":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if cat, ok := p.categories.selected(); ok {
			return p.formModal.Open("category.edit", "Edit Category", cat.ID, []FormField{
				{Key: "name", Label: "Name", Kind: FieldText, Value: cat.Name, CharLimit: 64},
				{Key: "slug", Label: "Slug", Kind: FieldText, Value: cat.Slug, CharLimit: 64},
				{Key: "description", Label: "Description", Kind: FieldText, Value: cat.Description, CharLimit: 256},
				{Key: "sort_order", Label: "Sort Order", Kind: FieldText, Value: strconv.Itoa(cat.SortOrder), CharLimit: 8},
			}, cat)
		}
	case "D":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if cat, ok := p.categories.selected(); ok {
			p.confirmModal.Open(
				"category.delete",
				"Delete Category",
				fmt.Sprintf("Delete category %q (%s)?\nAll boards and threads under it will also be deleted.", cat.Name, cat.Slug),
				"Delete",
				true,
				cat,
			)
		}
	}
	return nil
}

func (p *Panel) viewCategories(width, bodyH int) string {
	scopeLbl := p.categories.currentScopeFilter()
	if scopeLbl == "" {
		scopeLbl = "all"
	}
	toolbar := ui.TruncateWidth(
		p.theme.MutedText.Render(fmt.Sprintf("Scope: [%s] (s)  Categories: %d", scopeLbl, len(p.categories.items))),
		width,
	)

	paneH := max(bodyH-1, 5)
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)

	var listCard string
	innerW := max(listW-4, 18)
	if len(p.categories.items) == 0 {
		msg := "No categories found."
		if p.categories.errMsg != "" {
			msg = "Error: " + p.categories.errMsg
		} else if p.categories.loading {
			msg = "Loading categories..."
		}
		listCard = p.renderEmptyCard("Categories", msg, listW, innerH)
	} else {
		visibleRows := max(innerH-2, 1)
		start, end := visibleWindow(p.categories.selectedIdx, len(p.categories.items), visibleRows)
		nameW := max(innerW-20, 10)
		head := p.theme.TableHeader.Render(
			ui.PadRightWidth("Ord", 4) + " " + ui.PadRightWidth("Scp", 6) + " " + ui.PadRightWidth("Name", nameW) + " " + ui.PadRightWidth("Slug", 8),
		)
		rows := []string{p.theme.CardTitle.Render("Categories"), head}
		for i := start; i < end; i++ {
			cat := p.categories.items[i]
			line := ui.PadRightWidth(strconv.Itoa(cat.SortOrder), 4) + " " +
				ui.PadRightWidth(cat.Scope, 6) + " " +
				ui.PadRightWidth(cat.Name, nameW) + " " +
				ui.PadRightWidth(cat.Slug, 8)
			st := p.theme.TableRow
			if i == p.categories.selectedIdx {
				st = p.theme.TableSelected
			}
			rows = append(rows, st.Render(line))
		}
		listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
	}

	p.syncDetailViewport(&p.categories.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render("Category Detail") + "\n" + p.categories.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
