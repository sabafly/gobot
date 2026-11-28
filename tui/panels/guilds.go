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
	"context"
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

const (
	guildsSearchPrompt     = "Search: "
	guildModalLocaleTag    = "guild.locale"
	guildModalToggleBBSTag = "guild.bbs"
)

type guildsLoadedMsg struct {
	Resp admin.GuildsResponse
	Err  error
}

type guildDetailLoadedMsg struct {
	GuildID string
	Resp    admin.GuildDetailResponse
	Err     error
}

type guildUpdatedMsg struct {
	Resp admin.GuildDetailResponse
	Err  error
}

// GuildsPanel displays joined/known Discord guilds and allows locale override & BBS opt-in toggling.
type GuildsPanel struct {
	theme            ui.Theme
	searchInput      textinput.Model
	searching        bool
	appliedQuery     string
	guilds           []admin.GuildSummary
	availableLocales []string
	selectedIdx      int
	selectedID       string
	detail           *admin.GuildDetailResponse
	detailVP         viewport.Model
	loadingList      bool
	loadingDetail    bool
	mutating         bool
	errMsg           string

	selectModal  ui.SelectModal
	confirmModal ui.ConfirmModal
	width        int
	height       int
}

// NewGuildsPanel constructs the Guilds management panel.
func NewGuildsPanel(theme ui.Theme) *GuildsPanel {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(18))
	vp.SoftWrap = false
	return &GuildsPanel{
		theme:       theme,
		searchInput: ui.NewRealCursorInput("name or guild ID", 32),
		detailVP:    vp,
	}
}

func (p *GuildsPanel) ID() string                           { return "guilds" }
func (p *GuildsPanel) Title() string                        { return "Guilds" }
func (p *GuildsPanel) RequiredPermission() admin.Permission { return admin.PermGuildsRead }
func (p *GuildsPanel) Close()                               {}

func (p *GuildsPanel) CapturingInput() bool {
	return p.searching || p.selectModal.Active || p.confirmModal.Active
}

func (p *GuildsPanel) Cursor() *tea.Cursor {
	if !p.searching || p.selectModal.Active || p.confirmModal.Active || !p.searchInput.Focused() {
		return nil
	}
	c := p.searchInput.Cursor()
	if c == nil {
		return nil
	}
	cur := *c
	cur.X += lipgloss.Width(guildsSearchPrompt)
	cur.Y += 1
	return &cur
}

func (p *GuildsPanel) ShortHelp(ctx ui.PanelContext) []key.Binding {
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select")),
		key.NewBinding(key.WithKeys("J", "K"), key.WithHelp("J/K", "scroll")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
	if ctx.HasPerm(admin.PermGuildsWrite) {
		bindings = append(bindings,
			key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "locale")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "toggle bbs")),
		)
	}
	return bindings
}

func (p *GuildsPanel) Init(ctx ui.PanelContext) tea.Cmd {
	p.loadingList = true
	return p.fetchGuildsCmd(ctx.Client, p.appliedQuery)
}

func (p *GuildsPanel) Update(msg tea.Msg, ctx ui.PanelContext) (ui.Panel, tea.Cmd) {
	if p.selectModal.Active {
		opt, canceled, tag, payload := p.selectModal.Update(msg)
		if canceled {
			return p, nil
		}
		if opt != nil && tag == guildModalLocaleTag {
			gid, _ := payload.(string)
			val := opt.Value
			p.mutating = true
			return p, p.updateGuildCmd(ctx.Client, gid, admin.UpdateGuildRequest{LocaleOverride: &val})
		}
		return p, nil
	}

	if p.confirmModal.Active {
		confirmed, canceled, tag, payload := p.confirmModal.Update(msg)
		if canceled {
			return p, nil
		}
		if confirmed && tag == guildModalToggleBBSTag {
			if g, ok := payload.(admin.GuildSummary); ok {
				next := !g.BBSEnabled
				p.mutating = true
				return p, p.updateGuildCmd(ctx.Client, g.ID, admin.UpdateGuildRequest{BBSEnabled: &next})
			}
		}
		return p, nil
	}

	switch m := msg.(type) {
	case guildsLoadedMsg:
		p.loadingList = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.errMsg = m.Err.Error()
			p.rebuildDetail()
			return p, nil
		}
		p.errMsg = ""
		p.guilds = m.Resp.Guilds
		p.availableLocales = m.Resp.AvailableLocales
		if len(p.guilds) == 0 {
			p.selectedIdx = 0
			p.selectedID = ""
			p.detail = nil
			p.rebuildDetail()
			return p, nil
		}
		if idx := p.findIndexByID(p.selectedID); idx >= 0 {
			p.selectedIdx = idx
		} else {
			if p.selectedIdx >= len(p.guilds) {
				p.selectedIdx = len(p.guilds) - 1
			}
			if p.selectedIdx < 0 {
				p.selectedIdx = 0
			}
			p.selectedID = p.guilds[p.selectedIdx].ID
		}
		p.loadingDetail = true
		p.rebuildDetail()
		return p, p.fetchGuildDetailCmd(ctx.Client, p.selectedID)

	case guildDetailLoadedMsg:
		if m.GuildID != p.selectedID {
			return p, nil
		}
		p.loadingDetail = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.errMsg = m.Err.Error()
			p.rebuildDetail()
			return p, nil
		}
		resp := m.Resp
		p.detail = &resp
		if len(resp.AvailableLocales) > 0 {
			p.availableLocales = resp.AvailableLocales
		}
		p.rebuildDetail()
		return p, nil

	case guildUpdatedMsg:
		p.mutating = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			return p, ui.ShowToastCmd(ui.ToastError, "Update failed: "+m.Err.Error())
		}
		resp := m.Resp
		p.detail = &resp
		p.selectedID = resp.Guild.ID
		for i := range p.guilds {
			if p.guilds[i].ID == resp.Guild.ID {
				p.guilds[i] = resp.Guild
				p.selectedIdx = i
				break
			}
		}
		p.rebuildDetail()
		return p, ui.ShowToastCmd(ui.ToastSuccess, fmt.Sprintf("Updated guild %s", resp.Guild.Name))

	case tea.KeyPressMsg:
		if p.searching {
			switch m.String() {
			case "esc":
				p.searching = false
				p.searchInput.Blur()
				return p, nil
			case "enter":
				p.searching = false
				p.searchInput.Blur()
				p.appliedQuery = strings.TrimSpace(p.searchInput.Value())
				p.loadingList = true
				return p, p.fetchGuildsCmd(ctx.Client, p.appliedQuery)
			}
			var cmd tea.Cmd
			p.searchInput, cmd = p.searchInput.Update(msg)
			return p, cmd
		}

		switch m.String() {
		case "/":
			p.searching = true
			return p, p.searchInput.Focus()
		case "c":
			if p.appliedQuery != "" || p.searchInput.Value() != "" {
				p.appliedQuery = ""
				p.searchInput.SetValue("")
				p.loadingList = true
				return p, p.fetchGuildsCmd(ctx.Client, "")
			}
		case "r":
			p.loadingList = true
			return p, p.fetchGuildsCmd(ctx.Client, p.appliedQuery)
		case "j", "down":
			if p.selectedIdx < len(p.guilds)-1 {
				p.selectedIdx++
				p.selectedID = p.guilds[p.selectedIdx].ID
				p.loadingDetail = true
				p.rebuildDetail()
				return p, p.fetchGuildDetailCmd(ctx.Client, p.selectedID)
			}
		case "k", "up":
			if p.selectedIdx > 0 {
				p.selectedIdx--
				p.selectedID = p.guilds[p.selectedIdx].ID
				p.loadingDetail = true
				p.rebuildDetail()
				return p, p.fetchGuildDetailCmd(ctx.Client, p.selectedID)
			}
		case "g", "home":
			if len(p.guilds) > 0 && p.selectedIdx != 0 {
				p.selectedIdx = 0
				p.selectedID = p.guilds[0].ID
				p.loadingDetail = true
				p.rebuildDetail()
				return p, p.fetchGuildDetailCmd(ctx.Client, p.selectedID)
			}
		case "G", "end":
			if len(p.guilds) > 0 && p.selectedIdx != len(p.guilds)-1 {
				p.selectedIdx = len(p.guilds) - 1
				p.selectedID = p.guilds[p.selectedIdx].ID
				p.loadingDetail = true
				p.rebuildDetail()
				return p, p.fetchGuildDetailCmd(ctx.Client, p.selectedID)
			}
		case "J", "pgdown":
			p.detailVP.PageDown()
		case "K", "pgup":
			p.detailVP.PageUp()
		case "l":
			g, ok := p.selectedGuild()
			if !ok {
				return p, nil
			}
			if !ctx.HasPerm(admin.PermGuildsWrite) {
				return p, ui.ShowToastCmd(ui.ToastWarn, "Permission guilds.write required")
			}
			opts := []ui.SelectOption{{Label: "auto (clear override)", Value: ""}}
			for _, loc := range p.availableLocales {
				opts = append(opts, ui.SelectOption{Label: loc, Value: loc})
			}
			p.selectModal.Open(guildModalLocaleTag, "Set Guild Locale Override", g.Name+" ("+g.ID+")", opts, g.LocaleOverride, g.ID)
			return p, nil
		case "b":
			g, ok := p.selectedGuild()
			if !ok {
				return p, nil
			}
			if !ctx.HasPerm(admin.PermGuildsWrite) {
				return p, ui.ShowToastCmd(ui.ToastWarn, "Permission guilds.write required")
			}
			action := "Enable"
			danger := false
			if g.BBSEnabled {
				action = "Disable"
				danger = true
			}
			p.confirmModal.Open(
				guildModalToggleBBSTag,
				action+" BBS for Guild",
				fmt.Sprintf("Guild: %s (%s)\nCurrent BBS status: %v", g.Name, g.ID, g.BBSEnabled),
				action+" BBS",
				danger,
				g,
			)
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.detailVP, cmd = p.detailVP.Update(msg)
	return p, cmd
}

func (p *GuildsPanel) View(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	split := width >= 84
	listW := width - 2
	detailW := width - 2
	bodyH := max(height-2, 6)
	listInnerH := max(bodyH-4, 4)
	detailInnerH := max(bodyH-4, 4)
	if split {
		listW = max(width*46/100, 32)
		detailW = max(width-listW-1, 30)
	} else {
		listInnerH = max((bodyH/2)-3, 3)
		detailInnerH = max(bodyH-listInnerH-6, 4)
	}

	if p.width != width || p.height != height {
		p.width = width
		p.height = height
		ui.SetInputWidth(&p.searchInput, max(width-lipgloss.Width(guildsSearchPrompt)-28, 16))
		p.detailVP.SetWidth(max(detailW-4, 20))
		p.detailVP.SetHeight(max(detailInnerH-1, 1))
		p.rebuildDetail()
	}

	if p.selectModal.Active {
		return p.selectModal.View(width, height, p.theme)
	}
	if p.confirmModal.Active {
		return p.confirmModal.View(width, height, p.theme)
	}

	status := p.theme.MutedText.Render(fmt.Sprintf("%d guilds", len(p.guilds)))
	if p.loadingList {
		status = p.theme.BadgeInfo.Render("Loading...")
	} else if p.mutating {
		status = p.theme.BadgeWarn.Render("Updating...")
	}
	header := ui.TruncateWidth(p.theme.CardTitle.Render("Guilds")+"  "+status, width)

	prompt := p.theme.InputPrompt.Render(guildsSearchPrompt)
	searchSuffix := ""
	if p.searching {
		searchSuffix = "  " + p.theme.MutedText.Render("(Enter apply, Esc cancel)")
	} else if p.appliedQuery != "" {
		searchSuffix = "  " + p.theme.MutedText.Render("(active; [c] clear)")
	}
	filterLine := prompt + p.searchInput.View() + searchSuffix
	if lipgloss.Width(filterLine) > width {
		avail := max(width-lipgloss.Width(prompt)-lipgloss.Width(searchSuffix), 1)
		filterLine = prompt + ui.TruncateWidth(p.searchInput.View(), avail) + searchSuffix
		if lipgloss.Width(filterLine) > width {
			filterLine = ui.TruncateWidth(filterLine, max(width, 1))
		}
	}

	listView := p.renderGuildList(listW, listInnerH)
	detailBox := p.theme.Card.Width(detailW).Height(detailInnerH + 2).Render(
		p.theme.CardTitle.Render("Guild Detail") + "\n" + p.detailVP.View(),
	)

	var body string
	if split {
		body = lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailBox)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, listView, detailBox)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, filterLine, body)
}

func (p *GuildsPanel) renderGuildList(cardW, innerH int) string {
	innerW := max(cardW-4, 20)
	if len(p.guilds) == 0 {
		msg := "No guilds found."
		if p.errMsg != "" {
			msg = "Error: " + p.errMsg
		} else if p.loadingList {
			msg = "Loading guilds..."
		}
		return p.theme.Card.Width(cardW).Height(innerH + 2).Render(
			p.theme.CardTitle.Render("Joined Guilds") + "\n" + p.theme.MutedText.Render(msg),
		)
	}

	visibleRows := max(innerH-2, 1)
	start := 0
	if p.selectedIdx >= visibleRows {
		start = p.selectedIdx - visibleRows + 1
	}
	end := min(len(p.guilds), start+visibleRows)

	nameW := max(innerW-22, 12)
	head := p.theme.TableHeader.Render(
		ui.PadRightWidth("Guild", nameW) + " " + ui.PadRightWidth("Loc", 6) + " " + ui.PadRightWidth("BBS", 4) + " " + ui.PadRightWidth("Mem", 6),
	)
	rows := []string{p.theme.CardTitle.Render("Joined Guilds"), head}
	for i := start; i < end; i++ {
		g := p.guilds[i]
		bbsTxt := "OFF"
		if g.BBSEnabled {
			bbsTxt = "ON"
		}
		locTxt := g.EffectiveLocale
		if g.LocaleOverride != "" {
			locTxt = g.LocaleOverride + "*"
		}
		name := g.Name
		if strings.TrimSpace(name) == "" {
			name = g.ID
		}
		line := ui.PadRightWidth(name, nameW) + " " +
			ui.PadRightWidth(locTxt, 6) + " " +
			ui.PadRightWidth(bbsTxt, 4) + " " +
			ui.PadRightWidth(fmt.Sprintf("%d", g.MemberCount), 6)
		style := p.theme.TableRow
		if i == p.selectedIdx {
			style = p.theme.TableSelected
		}
		rows = append(rows, style.Render(line))
	}
	return p.theme.Card.Width(cardW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
}

func (p *GuildsPanel) rebuildDetail() {
	wrapW := max(p.detailVP.Width(), 1)
	if p.errMsg != "" && p.detail == nil {
		p.detailVP.SetContent(ui.WrapViewportText(p.theme.BadgeErr.Render("Error: "+p.errMsg), wrapW))
		return
	}
	if p.detail == nil {
		if p.loadingDetail {
			p.detailVP.SetContent(ui.WrapViewportText(p.theme.MutedText.Render("Loading guild detail..."), wrapW))
		} else {
			p.detailVP.SetContent(ui.WrapViewportText(p.theme.MutedText.Render("Select a guild to view details."), wrapW))
		}
		return
	}
	d := p.detail
	g := d.Guild
	kv := func(k, v string) string {
		return p.theme.Label.Render(ui.PadRightWidth(k+":", 17)) + " " + p.theme.Value.Render(v)
	}
	override := g.LocaleOverride
	if override == "" {
		override = "(auto)"
	}
	cachedStr := "No"
	if g.Cached {
		cachedStr = "Yes"
	}
	bbsStr := p.theme.BadgeWarn.Render("Disabled")
	if g.BBSEnabled {
		bbsStr = p.theme.BadgeOK.Render("Enabled")
	}
	forumDef := "-"
	if d.BBSForumDefault != nil {
		forumDef = fmt.Sprintf("%v", *d.BBSForumDefault)
	}
	modLog := "-"
	if d.BBSModLogChannel != "" {
		chName := strings.TrimSpace(d.BBSModLogChannelName)
		if chName != "" && !strings.HasPrefix(chName, "#") {
			chName = "#" + chName
		}
		modLog = ui.FormatIDWithName(chName, d.BBSModLogChannel)
	}
	lines := []string{
		kv("Guild ID", g.ID),
		kv("Name", g.Name),
		kv("Cached", cachedStr),
		kv("Members", fmt.Sprintf("%d", g.MemberCount)),
		kv("Preferred Loc", g.PreferredLocale),
		kv("Override Loc", override),
		kv("Effective Loc", g.EffectiveLocale),
		kv("BBS Opt-In", bbsStr),
		kv("BBS Mounts", fmt.Sprintf("%d", g.BBSMounts)),
		kv("Forum Default", forumDef),
		kv("Mod Log Ch", modLog),
		"",
		p.theme.MutedText.Render("Shortcuts: [l] set locale override  [b] toggle BBS  [J/K] scroll"),
	}
	p.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
}

func (p *GuildsPanel) findIndexByID(id string) int {
	if id == "" {
		return -1
	}
	for i, g := range p.guilds {
		if g.ID == id {
			return i
		}
	}
	return -1
}

func (p *GuildsPanel) selectedGuild() (admin.GuildSummary, bool) {
	if len(p.guilds) == 0 || p.selectedIdx < 0 || p.selectedIdx >= len(p.guilds) {
		return admin.GuildSummary{}, false
	}
	return p.guilds[p.selectedIdx], true
}

func (p *GuildsPanel) fetchGuildsCmd(client *adminclient.Client, query string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return guildsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListGuilds(reqCtx, query)
		return guildsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *GuildsPanel) fetchGuildDetailCmd(client *adminclient.Client, guildID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil || guildID == "" {
			return nil
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.GetGuild(reqCtx, guildID)
		return guildDetailLoadedMsg{GuildID: guildID, Resp: resp, Err: err}
	}
}

func (p *GuildsPanel) updateGuildCmd(client *adminclient.Client, guildID string, req admin.UpdateGuildRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return guildUpdatedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resp, err := client.UpdateGuild(reqCtx, guildID, req)
		return guildUpdatedMsg{Resp: resp, Err: err}
	}
}
