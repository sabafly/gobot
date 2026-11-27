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
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

const overviewRefreshInterval = 10 * time.Second

type overviewLoadedMsg struct {
	Data admin.OverviewResponse
	Err  error
	At   time.Time
}

type overviewTickMsg struct {
	Gen uint64
}

// OverviewPanel renders runtime metrics, database health, Discord shard latencies, and BBS summary stats.
type OverviewPanel struct {
	theme       ui.Theme
	vp          viewport.Model
	loading     bool
	active      bool
	tickGen     uint64
	data        *admin.OverviewResponse
	errMsg      string
	lastUpdated time.Time
	width       int
	height      int
}

// NewOverviewPanel constructs the Overview dashboard panel.
func NewOverviewPanel(theme ui.Theme) *OverviewPanel {
	vp := viewport.New(viewport.WithWidth(76), viewport.WithHeight(20))
	vp.SoftWrap = false
	return &OverviewPanel{
		theme: theme,
		vp:    vp,
	}
}

func (p *OverviewPanel) ID() string                          { return "overview" }
func (p *OverviewPanel) Title() string                       { return "Overview" }
func (p *OverviewPanel) RequiredPermission() admin.Permission { return admin.PermDashboardRead }
func (p *OverviewPanel) Cursor() *tea.Cursor                 { return nil }
func (p *OverviewPanel) CapturingInput() bool                { return false }

func (p *OverviewPanel) Close() {
	p.active = false
	p.tickGen++
}

func (p *OverviewPanel) ShortHelp(_ ui.PanelContext) []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "scroll")),
	}
}

func (p *OverviewPanel) Init(ctx ui.PanelContext) tea.Cmd {
	p.active = true
	p.loading = true
	p.tickGen++
	gen := p.tickGen
	p.rebuildContent()
	return tea.Batch(p.fetchOverviewCmd(ctx.Client), p.scheduleTick(gen))
}

func (p *OverviewPanel) Update(msg tea.Msg, ctx ui.PanelContext) (ui.Panel, tea.Cmd) {
	switch m := msg.(type) {
	case ui.PanelActivatedMsg:
		if m.ID == p.ID() {
			p.active = true
			p.tickGen++
			gen := p.tickGen
			p.loading = true
			p.rebuildContent()
			return p, tea.Batch(p.fetchOverviewCmd(ctx.Client), p.scheduleTick(gen))
		}
	case ui.PanelDeactivatedMsg:
		if m.ID == p.ID() {
			p.active = false
			p.tickGen++
		}
	case overviewTickMsg:
		if !p.active || m.Gen != p.tickGen {
			return p, nil
		}
		return p, tea.Batch(p.fetchOverviewCmd(ctx.Client), p.scheduleTick(m.Gen))
	case overviewLoadedMsg:
		p.loading = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.errMsg = m.Err.Error()
			p.rebuildContent()
			return p, nil
		}
		d := m.Data
		p.data = &d
		p.errMsg = ""
		p.lastUpdated = m.At
		p.rebuildContent()
		return p, nil
	case tea.KeyPressMsg:
		switch m.String() {
		case "r":
			p.loading = true
			p.rebuildContent()
			return p, p.fetchOverviewCmd(ctx.Client)
		case "j", "down":
			p.vp.ScrollDown(1)
			return p, nil
		case "k", "up":
			p.vp.ScrollUp(1)
			return p, nil
		case "g":
			p.vp.GotoTop()
			return p, nil
		case "G":
			p.vp.GotoBottom()
			return p, nil
		}
	}
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p *OverviewPanel) View(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	if p.width != width || p.height != height {
		p.width = width
		p.height = height
		p.vp.SetWidth(width)
		p.vp.SetHeight(max(height-2, 1))
		p.rebuildContent()
	}

	status := p.theme.MutedText.Render("Ready")
	if p.loading {
		status = p.theme.BadgeInfo.Render("Loading...")
	} else if p.errMsg != "" {
		status = p.theme.BadgeErr.Render("Error")
	} else if !p.lastUpdated.IsZero() {
		status = p.theme.MutedText.Render("Updated " + p.lastUpdated.Local().Format("15:04:05") + " (auto 10s)")
	}
	header := ui.TruncateWidth(
		p.theme.CardTitle.Render("Bot & System Overview")+"   "+status+"   "+p.theme.MutedText.Render("[r] Refresh"),
		width,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		p.vp.View(),
	)
}

func (p *OverviewPanel) fetchOverviewCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return overviewLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		data, err := client.Overview(reqCtx)
		return overviewLoadedMsg{
			Data: data,
			Err:  err,
			At:   time.Now(),
		}
	}
}

func (p *OverviewPanel) scheduleTick(gen uint64) tea.Cmd {
	return tea.Tick(overviewRefreshInterval, func(time.Time) tea.Msg {
		return overviewTickMsg{Gen: gen}
	})
}

func (p *OverviewPanel) rebuildContent() {
	w := p.width
	if w <= 0 {
		w = 80
	}
	wrapW := max(p.vp.Width(), 1)
	if p.errMsg != "" && p.data == nil {
		p.vp.SetContent(ui.WrapViewportText(p.theme.BadgeErr.Render("Failed to load overview: "+p.errMsg), wrapW))
		return
	}
	if p.data == nil {
		p.vp.SetContent(ui.WrapViewportText(p.theme.MutedText.Render("Loading overview metrics..."), wrapW))
		return
	}
	d := p.data

	twoCol := w >= 88
	cardW := w - 2
	if twoCol {
		cardW = max((w-3)/2, 36)
	}
	renderKV := func(label, val string) string {
		return p.theme.Label.Render(ui.PadRightWidth(label+":", 16)) + " " + p.theme.Value.Render(val)
	}

	botState := p.theme.BadgeErr.Render("OFFLINE")
	if d.BotConnected {
		botState = p.theme.BadgeOK.Render("ONLINE")
	}
	botName := d.BotUsername
	if botName == "" {
		botName = "-"
	}
	botCard := p.theme.Card.Width(cardW).Render(strings.Join([]string{
		p.theme.CardTitle.Render("Bot & Runtime"),
		renderKV("Version", d.Version),
		renderKV("Bot User", botName),
		renderKV("Gateway", botState),
		renderKV("Uptime", ui.FormatDuration(d.UptimeSeconds)),
		renderKV("Go Runtime", d.GoVersion),
		renderKV("Goroutines", fmt.Sprintf("%d", d.Goroutines)),
		renderKV("Memory", fmt.Sprintf("%s alloc / %s sys", ui.FormatBytes(d.MemAllocBytes), ui.FormatBytes(d.MemSysBytes))),
	}, "\n"))

	dbState := p.theme.BadgeOK.Render(fmt.Sprintf("CONNECTED (%d ms)", d.DBLatencyMs))
	if !d.DBConnected {
		dbState = p.theme.BadgeErr.Render("DISCONNECTED")
	}
	dbLines := []string{
		p.theme.CardTitle.Render("Database & System"),
		renderKV("Database", dbState),
	}
	if d.DBError != "" {
		dbLines = append(dbLines, renderKV("DB Error", p.theme.BadgeErr.Render(ui.TruncateWidth(d.DBError, max(cardW-22, 16)))))
	}
	dbLines = append(dbLines,
		renderKV("Cached Guilds", fmt.Sprintf("%d", d.CachedGuilds)),
		renderKV("DB Guilds", fmt.Sprintf("%d", d.DBGuilds)),
		renderKV("DB Users", fmt.Sprintf("%d", d.DBUsers)),
		renderKV("Sessions", fmt.Sprintf("%d active", d.ActiveSessions)),
		renderKV("Log Buffer", fmt.Sprintf("%d / %d", d.LogBufferLen, d.LogBufferCap)),
	)
	sysCard := p.theme.Card.Width(cardW).Render(strings.Join(dbLines, "\n"))

	shardLines := []string{
		p.theme.CardTitle.Render("Gateway Shards"),
		renderKV("Avg Latency", fmt.Sprintf("%d ms", d.AvgShardPingMs)),
	}
	if len(d.Shards) == 0 {
		shardLines = append(shardLines, p.theme.MutedText.Render("  No active shard telemetry"))
	} else {
		for _, sh := range d.Shards {
			shardLines = append(shardLines, fmt.Sprintf("  Shard #%-3d  %-14s  %d ms", sh.ShardID, sh.Status, sh.LatencyMs))
		}
	}
	shardCard := p.theme.Card.Width(cardW).Render(strings.Join(shardLines, "\n"))

	bbsLines := []string{
		p.theme.CardTitle.Render("BBS Summary"),
	}
	if !d.BBS.Available {
		bbsLines = append(bbsLines, p.theme.MutedText.Render("BBS tables unavailable or uninitialized"))
	} else {
		reportVal := fmt.Sprintf("%d", d.BBS.OpenReports)
		if d.BBS.OpenReports > 0 {
			reportVal = p.theme.BadgeWarn.Render(fmt.Sprintf("%d open!", d.BBS.OpenReports))
		}
		bbsLines = append(bbsLines,
			renderKV("Categories", fmt.Sprintf("%d", d.BBS.Categories)),
			renderKV("Boards", fmt.Sprintf("%d", d.BBS.Boards)),
			renderKV("Active Mounts", fmt.Sprintf("%d", d.BBS.ActiveMounts)),
			renderKV("Threads / Posts", fmt.Sprintf("%d / %d", d.BBS.Threads, d.BBS.Posts)),
			renderKV("Open Reports", reportVal),
			renderKV("Active Bans", fmt.Sprintf("%d", d.BBS.ActiveBans)),
		)
	}
	bbsCard := p.theme.Card.Width(cardW).Render(strings.Join(bbsLines, "\n"))

	var blocks []string
	if p.errMsg != "" {
		blocks = append(blocks, p.theme.BadgeWarn.Render("Last refresh error: "+p.errMsg))
	}
	if twoCol {
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, botCard, " ", sysCard)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, shardCard, " ", bbsCard)
		blocks = append(blocks, row1, "", row2)
	} else {
		blocks = append(blocks, botCard, "", sysCard, "", shardCard, "", bbsCard)
	}

	p.vp.SetContent(ui.WrapViewportText(strings.Join(blocks, "\n"), wrapW))
}
