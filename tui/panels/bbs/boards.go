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

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

type boardDrillLevel int

const (
	boardLevelBoards boardDrillLevel = iota
	boardLevelThreads
	boardLevelPosts
)

const threadSearchPrompt = "Search Threads: "

var threadStatusCycle = []string{"", "active", "archived"}

type boardsState struct {
	level boardDrillLevel

	items       []admin.BBSBoardDTO
	selectedIdx int
	selectedID  string
	mountsByID  map[string][]admin.BBSBoardMountDTO

	threads          []admin.BBSThreadDTO
	threadIdx        int
	selectedThreadID string
	threadStatusIdx  int
	searchInput      textinput.Model
	searching        bool
	appliedQuery     string

	threadDetail *admin.BBSThreadDetailResponse
	postIdx      int

	detailVP viewport.Model
	loaded   bool
	loading  bool
	mutating bool
	errMsg   string
}

func newBoardsState() boardsState {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(16))
	vp.SoftWrap = false
	return boardsState{
		mountsByID:  make(map[string][]admin.BBSBoardMountDTO),
		searchInput: ui.NewRealCursorInput("title keyword", 80),
		detailVP:    vp,
	}
}

func (s *boardsState) selectedBoard() (admin.BBSBoardDTO, bool) {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.items) {
		return admin.BBSBoardDTO{}, false
	}
	return s.items[s.selectedIdx], true
}

func (s *boardsState) selectedThread() (admin.BBSThreadDTO, bool) {
	if s.threadIdx < 0 || s.threadIdx >= len(s.threads) {
		return admin.BBSThreadDTO{}, false
	}
	return s.threads[s.threadIdx], true
}

func (s *boardsState) selectedPost() (admin.BBSPostDTO, bool) {
	if s.threadDetail == nil || s.postIdx < 0 || s.postIdx >= len(s.threadDetail.Posts) {
		return admin.BBSPostDTO{}, false
	}
	return s.threadDetail.Posts[s.postIdx], true
}

func formatDatOchiHours(h int) string {
	if h < 0 {
		return "off"
	}
	return fmt.Sprintf("%dh", h)
}

func (s *boardsState) currentThreadStatus() string {
	return threadStatusCycle[s.threadStatusIdx]
}

func (s *boardsState) rebuildDetail(theme ui.Theme) {
	wrapW := max(s.detailVP.Width(), 1)
	switch s.level {
	case boardLevelBoards:
		b, ok := s.selectedBoard()
		if !ok {
			msg := "No boards found."
			if s.errMsg != "" {
				msg = "Error: " + s.errMsg
			} else if s.loading {
				msg = "Loading boards..."
			}
			s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
			return
		}
		guildStr := "(global)"
		if b.GuildID != nil && *b.GuildID != "" {
			guildStr = ui.FormatIDWithName(b.GuildName, *b.GuildID)
		}
		lines := []string{
			kvLine(theme, "Board ID", b.ID),
			kvLine(theme, "Category", ui.FormatIDWithName(b.CategoryName, b.CategoryID)),
			kvLine(theme, "Name", b.Name),
			kvLine(theme, "Slug", b.Slug),
			kvLine(theme, "Scope", b.Scope),
			kvLine(theme, "Guild", guildStr),
			kvLine(theme, "Locked", fmt.Sprintf("%v", b.Locked)),
			kvLine(theme, "Default Name", b.Rules.DefaultName),
			kvLine(theme, "Hide ID", fmt.Sprintf("%v", b.Rules.HideID)),
			kvLine(theme, "SLIP Level", fmt.Sprintf("%d", b.Rules.SLIPLevel)),
			kvLine(theme, "Max Res", fmt.Sprintf("%d", b.Rules.MaxRes)),
			kvLine(theme, "Idle/LowRes", fmt.Sprintf("%s / %s", formatDatOchiHours(b.Rules.DatOchiIdleHours), formatDatOchiHours(b.Rules.DatOchiLowResHours))),
			"",
			theme.CardTitle.Render("Description"),
			b.Description,
			"",
			theme.CardTitle.Render("Discord Mounts"),
		}
		if mounts, ok := s.mountsByID[b.ID]; ok {
			if len(mounts) == 0 {
				lines = append(lines, theme.MutedText.Render("(not mounted in any Discord channel)"))
			} else {
				for _, m := range mounts {
					chName := strings.TrimSpace(m.ChannelName)
					if chName != "" && !strings.HasPrefix(chName, "#") {
						chName = "#" + chName
					}
					lines = append(lines, fmt.Sprintf("- guild=%s ch=%s locale=%s forum=%v active=%v",
						ui.FormatIDWithName(m.GuildName, m.GuildID),
						ui.FormatIDWithName(chName, m.ChannelID),
						m.Locale, m.IsForum, m.Active))
				}
			}
		} else {
			lines = append(lines, theme.MutedText.Render("Loading mounts..."))
		}
		lines = append(lines, "", theme.MutedText.Render("[Enter] Open Threads  Admin (PermBBSAdmin): [n] New  [e] Edit  [D] Delete"))
		s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))

	case boardLevelThreads:
		th, ok := s.selectedThread()
		if !ok {
			msg := "No threads match filter."
			if s.errMsg != "" {
				msg = "Error: " + s.errMsg
			} else if s.loading {
				msg = "Loading threads..."
			}
			s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
			return
		}
		boardName := ""
		if b, okB := s.selectedBoard(); okB {
			boardName = b.Name
		}
		lines := []string{
			kvLine(theme, "Thread ID", th.ID),
			kvLine(theme, "Board", ui.FormatIDWithName(boardName, th.BoardID)),
			kvLine(theme, "Title", th.Title),
			kvLine(theme, "Status", th.Status),
			kvLine(theme, "Res Count", fmt.Sprintf("%d / %d", th.ResCount, th.Rules.MaxRes)),
			kvLine(theme, "Bumped", th.BumpedAt.Local().Format("2006-01-02 15:04:05")),
			kvLine(theme, "Created", th.CreatedAt.Local().Format("2006-01-02 15:04:05")),
			"",
			theme.MutedText.Render("[Enter] View Posts  [Esc] Back  Moderator: [A] Archive Thread"),
		}
		s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))

	case boardLevelPosts:
		post, ok := s.selectedPost()
		if !ok {
			msg := "No posts in thread."
			if s.errMsg != "" {
				msg = "Error: " + s.errMsg
			} else if s.loading {
				msg = "Loading thread detail..."
			}
			s.detailVP.SetContent(ui.WrapViewportText(theme.MutedText.Render(msg), wrapW))
			return
		}
		lines := []string{
			kvLine(theme, "Post ID", post.ID),
			kvLine(theme, "Res #", fmt.Sprintf("%d", post.ResNumber)),
			kvLine(theme, "Name", strings.TrimSpace(strings.Join([]string{post.Name, post.Trip, post.Cap}, " "))),
			kvLine(theme, "AnonID", post.AnonID),
			kvLine(theme, "SLIP", post.SlipText),
			kvLine(theme, "Poster Hash", post.PosterHash),
			kvLine(theme, "Aboned", fmt.Sprintf("%v (%s)", post.Abone, post.AboneReason)),
			kvLine(theme, "Created", post.CreatedAt.Local().Format("2006-01-02 15:04:05")),
			"",
			theme.CardTitle.Render("Body"),
			post.Body,
			"",
			theme.MutedText.Render("[Esc] Back  Moderator: [a] Abone Post  [b] Ban Poster"),
		}
		s.detailVP.SetContent(ui.WrapViewportText(strings.Join(lines, "\n"), wrapW))
	}
	s.detailVP.GotoTop()
}

func (p *Panel) fetchBoardsCmd(client *adminclient.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return boardsLoadedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSBoards(reqCtx, admin.BBSBoardFilter{})
		return boardsLoadedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) fetchBoardMountsCmd(client *adminclient.Client, boardID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil || boardID == "" {
			return nil
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSBoardMounts(reqCtx, boardID)
		return boardMountsLoadedMsg{BoardID: boardID, Resp: resp, Err: err}
	}
}

func (p *Panel) createBoardCmd(client *adminclient.Client, req admin.CreateBBSBoardRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return boardSavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.CreateBBSBoard(reqCtx, req)
		return boardSavedMsg{Created: true, Resp: resp, Err: err}
	}
}

func (p *Panel) updateBoardCmd(client *adminclient.Client, id string, req admin.UpdateBBSBoardRequest) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return boardSavedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.UpdateBBSBoard(reqCtx, id, req)
		return boardSavedMsg{Created: false, Resp: resp, Err: err}
	}
}

func (p *Panel) deleteBoardCmd(client *adminclient.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return boardDeletedMsg{ID: id, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteBBSBoard(reqCtx, id)
		return boardDeletedMsg{ID: id, Err: err}
	}
}

func (p *Panel) fetchThreadsCmd(client *adminclient.Client, boardID string) tea.Cmd {
	status := p.boards.currentThreadStatus()
	query := p.boards.appliedQuery
	return func() tea.Msg {
		if client == nil {
			return threadsLoadedMsg{BoardID: boardID, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.ListBBSThreads(reqCtx, admin.BBSThreadFilter{
			BoardID: boardID,
			Status:  status,
			Query:   query,
			Limit:   100,
		})
		return threadsLoadedMsg{BoardID: boardID, Resp: resp, Err: err}
	}
}

func (p *Panel) fetchThreadDetailCmd(client *adminclient.Client, threadID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return threadDetailLoadedMsg{ThreadID: threadID, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.GetBBSThreadDetail(reqCtx, threadID, 0, 200)
		return threadDetailLoadedMsg{ThreadID: threadID, Resp: resp, Err: err}
	}
}

func (p *Panel) archiveThreadCmd(client *adminclient.Client, threadID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return threadArchivedMsg{Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resp, err := client.ArchiveBBSThread(reqCtx, threadID)
		return threadArchivedMsg{Resp: resp, Err: err}
	}
}

func (p *Panel) abonePostCmd(client *adminclient.Client, postID, reason string, fromReport bool) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return postAbonedMsg{FromReport: fromReport, Err: fmt.Errorf("admin client not initialized")}
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resp, err := client.AboneBBSPost(reqCtx, postID, reason)
		return postAbonedMsg{FromReport: fromReport, Resp: resp, Err: err}
	}
}

func (p *Panel) updateBoards(msg tea.Msg, ctx ui.PanelContext) tea.Cmd {
	if p.boards.searching && p.boards.level == boardLevelThreads {
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch k.String() {
			case "esc":
				p.boards.searching = false
				p.boards.searchInput.Blur()
				return nil
			case "enter":
				p.boards.searching = false
				p.boards.searchInput.Blur()
				p.boards.appliedQuery = strings.TrimSpace(p.boards.searchInput.Value())
				if b, ok := p.boards.selectedBoard(); ok {
					p.boards.loading = true
					return p.fetchThreadsCmd(ctx.Client, b.ID)
				}
				return nil
			}
		}
		var cmd tea.Cmd
		p.boards.searchInput, cmd = p.boards.searchInput.Update(msg)
		return cmd
	}

	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch p.boards.level {
	case boardLevelBoards:
		return p.updateBoardsRoot(k, ctx)
	case boardLevelThreads:
		return p.updateBoardThreads(k, ctx)
	case boardLevelPosts:
		return p.updateBoardPosts(k, ctx)
	}
	return nil
}

func (p *Panel) selectBoardIdx(idx int, ctx ui.PanelContext) tea.Cmd {
	if idx < 0 || idx >= len(p.boards.items) || idx == p.boards.selectedIdx {
		return nil
	}
	p.boards.selectedIdx = idx
	b := p.boards.items[idx]
	p.boards.selectedID = b.ID
	p.boards.rebuildDetail(p.theme)
	if _, ok := p.boards.mountsByID[b.ID]; !ok && ctx.Client != nil {
		return p.fetchBoardMountsCmd(ctx.Client, b.ID)
	}
	return nil
}

func (p *Panel) updateBoardsRoot(k tea.KeyPressMsg, ctx ui.PanelContext) tea.Cmd {
	switch k.String() {
	case "j", "down":
		return p.selectBoardIdx(p.boards.selectedIdx+1, ctx)
	case "k", "up":
		return p.selectBoardIdx(p.boards.selectedIdx-1, ctx)
	case "g", "home":
		return p.selectBoardIdx(0, ctx)
	case "G", "end":
		return p.selectBoardIdx(len(p.boards.items)-1, ctx)
	case "J", "pgdown":
		p.boards.detailVP.PageDown()
	case "K", "pgup":
		p.boards.detailVP.PageUp()
	case "r":
		p.boards.loading = true
		return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchBoardsCmd(ctx.Client))
	case "enter":
		if b, ok := p.boards.selectedBoard(); ok {
			p.boards.level = boardLevelThreads
			p.boards.threads = nil
			p.boards.threadIdx = 0
			p.boards.selectedThreadID = ""
			p.boards.appliedQuery = ""
			p.boards.searchInput.SetValue("")
			p.boards.loading = true
			p.boards.errMsg = ""
			p.boards.rebuildDetail(p.theme)
			return p.fetchThreadsCmd(ctx.Client, b.ID)
		}
	case "n":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		catField := ui.FormField{Key: "category", Label: "Category", Kind: ui.FieldText, Placeholder: "Category UUID", CharLimit: 40}
		if len(p.categories.items) > 0 {
			opts := make([]string, 0, len(p.categories.items))
			for _, c := range p.categories.items {
				opts = append(opts, c.Name+" | "+c.ID)
			}
			catField = ui.FormField{Key: "category", Label: "Category", Kind: ui.FieldCycle, Options: opts, Value: opts[0]}
		}
		return p.openForm("board.create", "Create Board", "Inherits Scope/Guild from selected category", []ui.FormField{
			catField,
			{Key: "slug", Label: "Slug", Kind: ui.FieldText, Placeholder: "vip", CharLimit: 64},
			{Key: "name", Label: "Name", Kind: ui.FieldText, Placeholder: "雑談板", CharLimit: 80},
			{Key: "description", Label: "Description", Kind: ui.FieldText, Placeholder: "Optional description", CharLimit: 256},
			{Key: "default_name", Label: "Default Name", Kind: ui.FieldText, Value: "名無しさん", CharLimit: 32},
			{Key: "hide_id", Label: "Hide ID", Kind: ui.FieldBool, BoolVal: false},
			{Key: "slip_level", Label: "SLIP (0/1/2)", Kind: ui.FieldCycle, Options: []string{"0", "1", "2"}, Value: "0"},
			{Key: "max_res", Label: "Max Res", Kind: ui.FieldText, Value: "1000", CharLimit: 6},
			{Key: "dat_ochi_idle", Label: "Idle Ochi (h, -1=off)", Kind: ui.FieldText, Value: "1200", CharLimit: 6},
			{Key: "dat_ochi_low_res", Label: "LowRes Ochi (h, -1=off)", Kind: ui.FieldText, Value: "240", CharLimit: 6},
		}, nil)
	case "e":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if b, ok := p.boards.selectedBoard(); ok {
			return p.openForm("board.edit", "Edit Board", b.ID, []ui.FormField{
				{Key: "name", Label: "Name", Kind: ui.FieldText, Value: b.Name, CharLimit: 80},
				{Key: "description", Label: "Description", Kind: ui.FieldText, Value: b.Description, CharLimit: 256},
				{Key: "default_name", Label: "Default Name", Kind: ui.FieldText, Value: b.Rules.DefaultName, CharLimit: 32},
				{Key: "hide_id", Label: "Hide ID", Kind: ui.FieldBool, BoolVal: b.Rules.HideID},
				{Key: "slip_level", Label: "SLIP (0/1/2)", Kind: ui.FieldCycle, Options: []string{"0", "1", "2"}, Value: strconv.Itoa(b.Rules.SLIPLevel)},
				{Key: "max_res", Label: "Max Res", Kind: ui.FieldText, Value: strconv.FormatUint(uint64(b.Rules.MaxRes), 10), CharLimit: 6},
				{Key: "dat_ochi_idle", Label: "Idle Ochi (h, -1=off)", Kind: ui.FieldText, Value: strconv.Itoa(b.Rules.DatOchiIdleHours), CharLimit: 6},
				{Key: "dat_ochi_low_res", Label: "LowRes Ochi (h, -1=off)", Kind: ui.FieldText, Value: strconv.Itoa(b.Rules.DatOchiLowResHours), CharLimit: 6},
				{Key: "locked", Label: "Locked", Kind: ui.FieldBool, BoolVal: b.Locked},
			}, b)
		}
	case "D", "delete":
		if !ctx.HasPerm(admin.PermBBSAdmin) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.admin permission")
		}
		if b, ok := p.boards.selectedBoard(); ok {
			p.confirmModal.Open("board.delete", "Delete Board", fmt.Sprintf("Delete board %q (%s) and all its threads?", b.Name, b.Slug), "Delete", true, b)
		}
	}
	return nil
}

func (p *Panel) updateBoardThreads(k tea.KeyPressMsg, ctx ui.PanelContext) tea.Cmd {
	switch k.String() {
	case "esc", "backspace":
		p.boards.level = boardLevelBoards
		p.boards.errMsg = ""
		p.boards.rebuildDetail(p.theme)
	case "/":
		p.boards.searching = true
		return p.boards.searchInput.Focus()
	case "c":
		if p.boards.appliedQuery != "" || p.boards.searchInput.Value() != "" {
			p.boards.appliedQuery = ""
			p.boards.searchInput.SetValue("")
			if b, ok := p.boards.selectedBoard(); ok {
				p.boards.loading = true
				return p.fetchThreadsCmd(ctx.Client, b.ID)
			}
		}
	case "f":
		p.boards.threadStatusIdx = (p.boards.threadStatusIdx + 1) % len(threadStatusCycle)
		if b, ok := p.boards.selectedBoard(); ok {
			p.boards.loading = true
			return p.fetchThreadsCmd(ctx.Client, b.ID)
		}
	case "r":
		if b, ok := p.boards.selectedBoard(); ok {
			p.boards.loading = true
			return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchThreadsCmd(ctx.Client, b.ID))
		}
	case "j", "down":
		if p.boards.threadIdx < len(p.boards.threads)-1 {
			p.boards.threadIdx++
			p.boards.selectedThreadID = p.boards.threads[p.boards.threadIdx].ID
			p.boards.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.boards.threadIdx > 0 {
			p.boards.threadIdx--
			p.boards.selectedThreadID = p.boards.threads[p.boards.threadIdx].ID
			p.boards.rebuildDetail(p.theme)
		}
	case "g", "home":
		if len(p.boards.threads) > 0 && p.boards.threadIdx != 0 {
			p.boards.threadIdx = 0
			p.boards.selectedThreadID = p.boards.threads[0].ID
			p.boards.rebuildDetail(p.theme)
		}
	case "G", "end":
		if len(p.boards.threads) > 0 && p.boards.threadIdx != len(p.boards.threads)-1 {
			p.boards.threadIdx = len(p.boards.threads) - 1
			p.boards.selectedThreadID = p.boards.threads[p.boards.threadIdx].ID
			p.boards.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.boards.detailVP.PageDown()
	case "K", "pgup":
		p.boards.detailVP.PageUp()
	case "enter":
		if th, ok := p.boards.selectedThread(); ok {
			p.boards.level = boardLevelPosts
			p.boards.threadDetail = nil
			p.boards.postIdx = 0
			p.boards.loading = true
			p.boards.errMsg = ""
			p.boards.rebuildDetail(p.theme)
			return p.fetchThreadDetailCmd(ctx.Client, th.ID)
		}
	case "A":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if th, ok := p.boards.selectedThread(); ok {
			p.confirmModal.Open("thread.archive", "Archive Thread", fmt.Sprintf("Archive thread %q (dat落ち)?", th.Title), "Archive", true, th)
		}
	}
	return nil
}

func (p *Panel) updateBoardPosts(k tea.KeyPressMsg, ctx ui.PanelContext) tea.Cmd {
	switch k.String() {
	case "esc", "backspace":
		p.boards.level = boardLevelThreads
		p.boards.errMsg = ""
		p.boards.rebuildDetail(p.theme)
	case "r":
		if th, ok := p.boards.selectedThread(); ok {
			p.boards.loading = true
			return tea.Batch(p.fetchStatsCmd(ctx.Client), p.fetchThreadDetailCmd(ctx.Client, th.ID))
		}
	case "j", "down":
		if p.boards.threadDetail != nil && p.boards.postIdx < len(p.boards.threadDetail.Posts)-1 {
			p.boards.postIdx++
			p.boards.rebuildDetail(p.theme)
		}
	case "k", "up":
		if p.boards.postIdx > 0 {
			p.boards.postIdx--
			p.boards.rebuildDetail(p.theme)
		}
	case "g", "home":
		if p.boards.threadDetail != nil && len(p.boards.threadDetail.Posts) > 0 && p.boards.postIdx != 0 {
			p.boards.postIdx = 0
			p.boards.rebuildDetail(p.theme)
		}
	case "G", "end":
		if p.boards.threadDetail != nil && len(p.boards.threadDetail.Posts) > 0 && p.boards.postIdx != len(p.boards.threadDetail.Posts)-1 {
			p.boards.postIdx = len(p.boards.threadDetail.Posts) - 1
			p.boards.rebuildDetail(p.theme)
		}
	case "J", "pgdown":
		p.boards.detailVP.PageDown()
	case "K", "pgup":
		p.boards.detailVP.PageUp()
	case "a":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if post, ok := p.boards.selectedPost(); ok {
			return p.openForm("post.abone", fmt.Sprintf("Abone Post #%d", post.ResNumber), shortID(post.ID), []ui.FormField{
				{Key: "reason", Label: "Reason", Kind: ui.FieldText, Value: "モデレーター対応", CharLimit: 120},
			}, post)
		}
	case "b":
		if !ctx.HasPerm(admin.PermBBSModerate) {
			return ui.ShowToastCmd(ui.ToastWarn, "Requires bbs.moderate permission")
		}
		if post, ok := p.boards.selectedPost(); ok {
			defScope := "local"
			defGuild := ""
			if b, okB := p.boards.selectedBoard(); okB {
				defScope = b.Scope
				if b.GuildID != nil {
					defGuild = *b.GuildID
				}
			}
			return p.openForm("post.ban", fmt.Sprintf("Ban Poster (Res #%d)", post.ResNumber), "Target Post "+shortID(post.ID), []ui.FormField{
				{Key: "scope", Label: "Scope", Kind: ui.FieldCycle, Options: []string{"local", "global"}, Value: defScope},
				{Key: "guild_id", Label: "Guild ID (local)", Kind: ui.FieldText, Value: defGuild, CharLimit: 32},
				{Key: "post_id", Label: "Post ID", Kind: ui.FieldText, Value: post.ID, ReadOnly: true},
				{Key: "duration_hours", Label: "Hours (0=perm)", Kind: ui.FieldText, Value: "24", CharLimit: 8},
				{Key: "reason", Label: "Reason", Kind: ui.FieldText, Value: "荒らし行為", CharLimit: 200},
			}, nil)
		}
	}
	return nil
}

func (p *Panel) viewBoards(width, bodyH int) string {
	var toolbar string
	paneH := max(bodyH-1, 5)
	switch p.boards.level {
	case boardLevelBoards:
		toolbar = ui.TruncateWidth(
			p.theme.MutedText.Render(fmt.Sprintf("Boards (%d)  [Enter] Open Threads", len(p.boards.items))),
			width,
		)
	case boardLevelThreads:
		bName := ""
		if b, ok := p.boards.selectedBoard(); ok {
			bName = b.Name
		}
		stLbl := p.boards.currentThreadStatus()
		if stLbl == "" {
			stLbl = "all"
		}
		crumb := p.theme.MutedText.Render(
			fmt.Sprintf("Boards > %s > Threads (%d, status:%s [f])  [Esc] Back", ui.TruncateWidth(bName, 24), len(p.boards.threads), stLbl),
		)
		prompt := p.theme.InputPrompt.Render(threadSearchPrompt)
		ui.SetInputWidth(&p.boards.searchInput, max(width-lipgloss.Width(threadSearchPrompt)-30, 12))
		searchSuffix := ""
		if p.boards.searching {
			searchSuffix = "  " + p.theme.MutedText.Render("(Enter apply, Esc cancel)")
		} else if p.boards.appliedQuery != "" {
			searchSuffix = "  " + p.theme.MutedText.Render("(active; [c] clear)")
		}
		searchLine := prompt + p.boards.searchInput.View() + searchSuffix
		if lipgloss.Width(searchLine) > width {
			avail := max(width-lipgloss.Width(prompt)-lipgloss.Width(searchSuffix), 1)
			searchLine = prompt + ui.TruncateWidth(p.boards.searchInput.View(), avail) + searchSuffix
			if lipgloss.Width(searchLine) > width {
				searchLine = ui.TruncateWidth(searchLine, max(width, 1))
			}
		}
		toolbar = ui.TruncateWidth(crumb, width) + "\n" + searchLine
		paneH = max(bodyH-2, 5)
	case boardLevelPosts:
		thTitle := ""
		if th, ok := p.boards.selectedThread(); ok {
			thTitle = th.Title
		}
		cnt := 0
		if p.boards.threadDetail != nil {
			cnt = len(p.boards.threadDetail.Posts)
		}
		toolbar = ui.TruncateWidth(
			p.theme.MutedText.Render(fmt.Sprintf("Boards > ... > %s > Posts (%d)  [Esc] Back", ui.TruncateWidth(thTitle, 28), cnt)),
			width,
		)
	}
	split, listW, detailW, innerH := p.splitDimensions(width, paneH)
	innerW := max(listW-4, 18)

	var listCard, detailTitle string
	switch p.boards.level {
	case boardLevelBoards:
		detailTitle = "Board Detail"
		if len(p.boards.items) == 0 {
			msg := "No boards found."
			if p.boards.errMsg != "" {
				msg = "Error: " + p.boards.errMsg
			} else if p.boards.loading {
				msg = "Loading boards..."
			}
			listCard = p.renderEmptyCard("Boards", msg, listW, innerH)
		} else {
			visibleRows := max(innerH-2, 1)
			start, end := visibleWindow(p.boards.selectedIdx, len(p.boards.items), visibleRows)
			nameW := max(innerW-16, 10)
			head := p.theme.TableHeader.Render(
				ui.PadRightWidth("Scp", 6) + " " + ui.PadRightWidth("Lock", 5) + " " + ui.PadRightWidth("Name", nameW),
			)
			rows := []string{p.theme.CardTitle.Render("Boards"), head}
			for i := start; i < end; i++ {
				b := p.boards.items[i]
				lk := "-"
				if b.Locked {
					lk = "LOCK"
				}
				line := ui.PadRightWidth(b.Scope, 6) + " " + ui.PadRightWidth(lk, 5) + " " + ui.PadRightWidth(b.Name, nameW)
				st := p.theme.TableRow
				if i == p.boards.selectedIdx {
					st = p.theme.TableSelected
				}
				rows = append(rows, st.Render(line))
			}
			listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
		}

	case boardLevelThreads:
		detailTitle = "Thread Detail"
		if len(p.boards.threads) == 0 {
			msg := "No threads found."
			if p.boards.errMsg != "" {
				msg = "Error: " + p.boards.errMsg
			} else if p.boards.loading {
				msg = "Loading threads..."
			}
			listCard = p.renderEmptyCard("Threads", msg, listW, innerH)
		} else {
			visibleRows := max(innerH-2, 1)
			start, end := visibleWindow(p.boards.threadIdx, len(p.boards.threads), visibleRows)
			titleW := max(innerW-16, 10)
			head := p.theme.TableHeader.Render(
				ui.PadRightWidth("Stat", 6) + " " + ui.PadRightWidth("Res", 5) + " " + ui.PadRightWidth("Title", titleW),
			)
			rows := []string{p.theme.CardTitle.Render("Threads"), head}
			for i := start; i < end; i++ {
				th := p.boards.threads[i]
				line := ui.PadRightWidth(th.Status, 6) + " " +
					ui.PadRightWidth(strconv.FormatUint(uint64(th.ResCount), 10), 5) + " " +
					ui.PadRightWidth(th.Title, titleW)
				st := p.theme.TableRow
				if i == p.boards.threadIdx {
					st = p.theme.TableSelected
				}
				rows = append(rows, st.Render(line))
			}
			listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
		}

	case boardLevelPosts:
		detailTitle = "Post Detail"
		var posts []admin.BBSPostDTO
		if p.boards.threadDetail != nil {
			posts = p.boards.threadDetail.Posts
		}
		if len(posts) == 0 {
			msg := "No posts loaded."
			if p.boards.errMsg != "" {
				msg = "Error: " + p.boards.errMsg
			} else if p.boards.loading {
				msg = "Loading posts..."
			}
			listCard = p.renderEmptyCard("Posts", msg, listW, innerH)
		} else {
			visibleRows := max(innerH-2, 1)
			start, end := visibleWindow(p.boards.postIdx, len(posts), visibleRows)
			bodyW := max(innerW-18, 8)
			head := p.theme.TableHeader.Render(
				ui.PadRightWidth("#", 4) + " " + ui.PadRightWidth("ID", 9) + " " + ui.PadRightWidth("Body", bodyW),
			)
			rows := []string{p.theme.CardTitle.Render("Posts"), head}
			for i := start; i < end; i++ {
				pt := posts[i]
				preview := pt.Body
				if pt.Abone {
					preview = "[あぼーん]"
				}
				line := ui.PadRightWidth(strconv.FormatUint(uint64(pt.ResNumber), 10), 4) + " " +
					ui.PadRightWidth(pt.AnonID, 9) + " " +
					ui.PadRightWidth(preview, bodyW)
				st := p.theme.TableRow
				if i == p.boards.postIdx {
					st = p.theme.TableSelected
				}
				rows = append(rows, st.Render(line))
			}
			listCard = p.theme.Card.Width(listW).Height(innerH + 2).Render(strings.Join(rows, "\n"))
		}
	}

	p.syncDetailViewport(&p.boards.detailVP, detailW, innerH)
	detailCard := p.theme.Card.Width(detailW).Height(innerH + 2).Render(
		p.theme.CardTitle.Render(detailTitle) + "\n" + p.boards.detailVP.View(),
	)

	return toolbar + "\n" + p.joinPanes(split, listCard, detailCard)
}
