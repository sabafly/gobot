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
	"maps"
	"slices"
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
	maxLogsRetained      = 2000
	logsReconnectDelay   = 3 * time.Second
	logsFilterPromptText = "Filter: "
)

var logLevelCycle = []string{"", "DEBUG", "INFO", "WARN", "ERROR"}

type logsInitialLoadedMsg struct {
	Gen  uint64
	Resp admin.LogsResponse
	Err  error
}

type logStreamEvent struct {
	Entry *admin.LogEntry
	Err   error
}

type logStreamMsg struct {
	Gen uint64
	Ev  logStreamEvent
}

type logReconnectTickMsg struct {
	Gen uint64
}

// LogsPanel renders buffered and live SSE slog streams with filtering and auto-follow scrolling.
type LogsPanel struct {
	theme        ui.Theme
	vp           viewport.Model
	searchInput  textinput.Model
	searching    bool
	paused       bool
	follow       bool
	loading      bool
	streaming    bool
	reconnecting bool
	streamErr    string

	levelIdx     int
	appliedQuery string
	lastSeq      uint64
	entries      []admin.LogEntry
	pausedBuf    []admin.LogEntry

	streamGen    uint64
	streamCancel context.CancelFunc
	streamCh     chan logStreamEvent

	width  int
	height int
}

// NewLogsPanel constructs the real-time slog viewer panel.
func NewLogsPanel(theme ui.Theme) *LogsPanel {
	vp := viewport.New(viewport.WithWidth(76), viewport.WithHeight(20))
	vp.SoftWrap = true
	si := ui.NewRealCursorInput("keyword (Enter apply, Esc cancel)", 36)
	return &LogsPanel{
		theme:       theme,
		vp:          vp,
		searchInput: si,
		follow:      true,
	}
}

func (p *LogsPanel) ID() string                           { return "logs" }
func (p *LogsPanel) Title() string                        { return "Logs" }
func (p *LogsPanel) RequiredPermission() admin.Permission { return admin.PermLogsRead }
func (p *LogsPanel) CapturingInput() bool                 { return p.searching }

func (p *LogsPanel) Close() {
	p.stopStream()
}

func (p *LogsPanel) Cursor() *tea.Cursor {
	if !p.searching || !p.searchInput.Focused() {
		return nil
	}
	c := p.searchInput.Cursor()
	if c == nil {
		return nil
	}
	cur := *c
	cur.X += lipgloss.Width(logsFilterPromptText)
	cur.Y += 1
	return &cur
}

func (p *LogsPanel) ShortHelp(_ ui.PanelContext) []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "level")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "pause/resume")),
		key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "follow")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	}
}

func (p *LogsPanel) Init(ctx ui.PanelContext) tea.Cmd {
	return p.reloadAndStream(ctx)
}

func (p *LogsPanel) Update(msg tea.Msg, ctx ui.PanelContext) (ui.Panel, tea.Cmd) {
	switch m := msg.(type) {
	case ui.PanelActivatedMsg:
		if m.ID == p.ID() && !p.streaming && !p.loading {
			return p, p.reloadAndStream(ctx)
		}
	case logsInitialLoadedMsg:
		if m.Gen != p.streamGen {
			return p, nil
		}
		p.loading = false
		if m.Err != nil {
			if adminclient.IsUnauthorized(m.Err) {
				p.stopStream()
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Err} }
			}
			p.streamErr = m.Err.Error()
			p.rebuildContent()
			return p, nil
		}
		p.entries = nil
		p.pausedBuf = nil
		p.lastSeq = 0
		for _, e := range m.Resp.Entries {
			p.appendEntry(&p.entries, e)
			if e.Seq > p.lastSeq {
				p.lastSeq = e.Seq
			}
		}
		if m.Resp.NextSeq > p.lastSeq {
			p.lastSeq = m.Resp.NextSeq
		}
		p.streamErr = ""
		p.rebuildContent()
		return p, p.startStreamCmd(ctx.Client, m.Gen)

	case logStreamMsg:
		if m.Gen != p.streamGen {
			return p, nil
		}
		if m.Ev.Err != nil {
			p.streaming = false
			if adminclient.IsUnauthorized(m.Ev.Err) {
				p.stopStream()
				return p, func() tea.Msg { return ui.UnauthorizedMsg{Err: m.Ev.Err} }
			}
			p.reconnecting = true
			p.streamErr = m.Ev.Err.Error()
			return p, tea.Tick(logsReconnectDelay, func(time.Time) tea.Msg {
				return logReconnectTickMsg{Gen: m.Gen}
			})
		}
		if m.Ev.Entry != nil {
			entry := *m.Ev.Entry
			if entry.Seq == 0 || entry.Seq > p.lastSeq {
				if entry.Seq > p.lastSeq {
					p.lastSeq = entry.Seq
				}
				if p.paused {
					p.appendEntry(&p.pausedBuf, entry)
				} else {
					p.appendEntry(&p.entries, entry)
					p.rebuildContent()
				}
			}
		}
		return p, p.waitStreamCmd(m.Gen, p.streamCh)

	case logReconnectTickMsg:
		if m.Gen != p.streamGen || !p.reconnecting {
			return p, nil
		}
		p.reconnecting = false
		return p, p.startStreamCmd(ctx.Client, m.Gen)

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
				return p, p.reloadAndStream(ctx)
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
				return p, p.reloadAndStream(ctx)
			}
			return p, nil
		case "l":
			p.levelIdx = (p.levelIdx + 1) % len(logLevelCycle)
			return p, p.reloadAndStream(ctx)
		case "space":
			p.paused = !p.paused
			if !p.paused && len(p.pausedBuf) > 0 {
				for _, e := range p.pausedBuf {
					p.appendEntry(&p.entries, e)
				}
				p.pausedBuf = nil
				p.rebuildContent()
			}
			return p, nil
		case "r":
			return p, p.reloadAndStream(ctx)
		case "j", "down":
			p.vp.ScrollDown(1)
			p.follow = p.vp.AtBottom()
			return p, nil
		case "k", "up":
			p.vp.ScrollUp(1)
			p.follow = p.vp.AtBottom()
			return p, nil
		case "pgdown":
			p.vp.PageDown()
			p.follow = p.vp.AtBottom()
			return p, nil
		case "pgup":
			p.vp.PageUp()
			p.follow = false
			return p, nil
		case "g":
			p.vp.GotoTop()
			p.follow = false
			return p, nil
		case "G":
			p.follow = true
			p.vp.GotoBottom()
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p *LogsPanel) View(width, height int) string {
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
		p.searchInput.SetWidth(max(width-lipgloss.Width(logsFilterPromptText)-22, 16))
		p.rebuildContent()
	}

	lvlLabel := logLevelCycle[p.levelIdx]
	if lvlLabel == "" {
		lvlLabel = "ALL"
	}

	streamBadge := p.theme.MutedText.Render("IDLE")
	switch {
	case p.loading:
		streamBadge = p.theme.BadgeInfo.Render("LOADING")
	case p.paused:
		streamBadge = p.theme.BadgeWarn.Render(fmt.Sprintf("PAUSED (+%d)", len(p.pausedBuf)))
	case p.reconnecting:
		streamBadge = p.theme.BadgeWarn.Render("RECONNECTING")
	case p.streaming:
		streamBadge = p.theme.BadgeOK.Render("LIVE")
	case p.streamErr != "":
		streamBadge = p.theme.BadgeErr.Render("ERROR")
	}

	followLabel := "follow:off"
	if p.follow {
		followLabel = "follow:on"
	}
	header := ui.TruncateWidth(
		p.theme.CardTitle.Render("Live Logs")+"  "+
			streamBadge+"  "+
			p.theme.BadgeInfo.Render("Level:"+lvlLabel)+"  "+
			p.theme.MutedText.Render(fmt.Sprintf("entries:%d  %s  [l] level [space] pause [G] bottom", len(p.entries), followLabel)),
		width,
	)

	prompt := p.theme.InputPrompt.Render(logsFilterPromptText)
	searchSuffix := ""
	if p.searching {
		searchSuffix = "  " + p.theme.MutedText.Render("(Enter apply, Esc cancel)")
	} else if p.appliedQuery != "" {
		searchSuffix = "  " + p.theme.MutedText.Render("(active; [c] clear)")
	}
	filterLine := prompt + p.searchInput.View() + searchSuffix
	if lipgloss.Width(filterLine) > width {
		// Keep cursor position exact while preventing line wrapping on narrow widths.
		maxRest := max(width-lipgloss.Width(prompt), 1)
		filterLine = prompt + ui.TruncateWidth(p.searchInput.View(), maxRest)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		filterLine,
		p.vp.View(),
	)
}

func (p *LogsPanel) currentFilter() admin.LogFilter {
	return admin.LogFilter{
		MinLevel: logLevelCycle[p.levelIdx],
		Query:    p.appliedQuery,
		Limit:    300,
	}
}

func (p *LogsPanel) reloadAndStream(ctx ui.PanelContext) tea.Cmd {
	p.stopStream()
	p.loading = true
	p.reconnecting = false
	p.streamErr = ""
	gen := p.streamGen
	filter := p.currentFilter()
	client := ctx.Client
	return func() tea.Msg {
		if client == nil {
			return logsInitialLoadedMsg{Gen: gen, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListLogs(reqCtx, filter)
		return logsInitialLoadedMsg{Gen: gen, Resp: resp, Err: err}
	}
}

func (p *LogsPanel) stopStream() {
	p.streamGen++
	p.streaming = false
	p.reconnecting = false
	if p.streamCancel != nil {
		p.streamCancel()
		p.streamCancel = nil
	}
	p.streamCh = nil
}

func (p *LogsPanel) startStreamCmd(client *adminclient.Client, gen uint64) tea.Cmd {
	if client == nil || gen != p.streamGen {
		return nil
	}
	streamCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan logStreamEvent, 64)
	p.streamCancel = cancel
	p.streamCh = ch
	p.streaming = true

	filter := p.currentFilter()
	filter.AfterSeq = p.lastSeq

	go func() {
		defer close(ch)
		err := client.StreamLogs(streamCtx, filter, func(entry admin.LogEntry) error {
			eCopy := entry
			if entry.Attrs != nil {
				eCopy.Attrs = maps.Clone(entry.Attrs)
			}
			select {
			case ch <- logStreamEvent{Entry: &eCopy}:
				return nil
			case <-streamCtx.Done():
				return streamCtx.Err()
			}
		})
		if streamCtx.Err() != nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("log stream closed by server")
		}
		select {
		case ch <- logStreamEvent{Err: err}:
		case <-streamCtx.Done():
		}
	}()

	return p.waitStreamCmd(gen, ch)
}

func (p *LogsPanel) waitStreamCmd(gen uint64, ch <-chan logStreamEvent) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return logStreamMsg{Gen: gen, Ev: ev}
	}
}

func (p *LogsPanel) appendEntry(dst *[]admin.LogEntry, e admin.LogEntry) {
	*dst = append(*dst, e)
	if len(*dst) > maxLogsRetained {
		drop := len(*dst) - maxLogsRetained
		*dst = slices.Clone((*dst)[drop:])
	}
}

func (p *LogsPanel) rebuildContent() {
	if len(p.entries) == 0 {
		if p.streamErr != "" {
			p.vp.SetContent(p.theme.BadgeErr.Render("Failed to load logs: " + p.streamErr))
		} else if p.loading {
			p.vp.SetContent(p.theme.MutedText.Render("Loading log buffer..."))
		} else {
			p.vp.SetContent(p.theme.MutedText.Render("No log entries match current filter."))
		}
		return
	}

	var lines []string
	if p.streamErr != "" {
		lines = append(lines, p.theme.BadgeWarn.Render("Stream notice: "+p.streamErr))
	}
	for _, e := range p.entries {
		lines = append(lines, p.formatEntry(e))
	}
	p.vp.SetContent(strings.Join(lines, "\n"))
	if p.follow {
		p.vp.GotoBottom()
	}
}

func (p *LogsPanel) formatEntry(e admin.LogEntry) string {
	ts := e.Time.Local().Format("15:04:05.000")
	lvl := strings.ToUpper(e.Level)
	var lvlRendered string
	switch lvl {
	case "ERROR":
		lvlRendered = p.theme.BadgeErr.Render("ERROR")
	case "WARN":
		lvlRendered = p.theme.BadgeWarn.Render("WARN ")
	case "DEBUG":
		lvlRendered = p.theme.MutedText.Render("DEBUG")
	default:
		lvlRendered = p.theme.BadgeInfo.Render(ui.PadRightWidth(lvl, 5))
	}

	var attrStr strings.Builder
	if len(e.Attrs) > 0 {
		keys := make([]string, 0, len(e.Attrs))
		for k := range e.Attrs {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fmt.Fprintf(&attrStr, " %s=%v", k, e.Attrs[k])
		}
	}
	if e.Source != "" {
		fmt.Fprintf(&attrStr, " src=%s", e.Source)
	}
	return fmt.Sprintf("%s [%s] %s%s",
		p.theme.MutedText.Render(ts),
		lvlRendered,
		p.theme.Value.Render(e.Message),
		p.theme.MutedText.Render(attrStr.String()),
	)
}
