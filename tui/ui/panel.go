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

package ui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
)

// PanelContext provides shared runtime state and authenticated API access to dashboard panels.
type PanelContext struct {
	Client    *adminclient.Client
	Principal admin.PrincipalInfo
	BaseURL   string
}

// HasPerm reports whether the authenticated operator holds perm (empty perm always returns true).
func (c PanelContext) HasPerm(perm admin.Permission) bool {
	if perm == "" {
		return true
	}
	return slices.Contains(c.Principal.Permissions, perm)
}

// Panel defines the plugin interface implemented by every dashboard screen.
type Panel interface {
	ID() string
	Title() string
	RequiredPermission() admin.Permission
	Init(ctx PanelContext) tea.Cmd
	Update(msg tea.Msg, ctx PanelContext) (Panel, tea.Cmd)
	View(width, height int) string
	Cursor() *tea.Cursor
	CapturingInput() bool
	ShortHelp(ctx PanelContext) []key.Binding
	Close()
}

// ToastKind classifies ephemeral status notifications rendered in the footer bar.
type ToastKind int

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarn
	ToastError
)

// ToastMsg displays a temporary notification banner in the TUI footer.
type ToastMsg struct {
	Kind ToastKind
	Text string
}

// ShowToastCmd emits a ToastMsg asynchronously inside Bubble Tea's Update loop.
func ShowToastCmd(kind ToastKind, text string) tea.Cmd {
	return func() tea.Msg {
		return ToastMsg{Kind: kind, Text: text}
	}
}

// UnauthorizedMsg notifies the root app that the current session token is invalid or revoked.
type UnauthorizedMsg struct {
	Err error
}

// PanelActivatedMsg notifies a panel when it becomes the active visible screen.
type PanelActivatedMsg struct {
	ID string
}

// PanelDeactivatedMsg notifies a panel when the operator switches away from it.
type PanelDeactivatedMsg struct {
	ID string
}

// NewRealCursorInput constructs a single-line textinput configured for hardware/IME cursor rendering.
func NewRealCursorInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetVirtualCursor(false)
	ti.SetStyles(textinput.DefaultDarkStyles())
	if width > 0 {
		ti.SetWidth(width)
	}
	return ti
}

// SetInputWidth updates a textinput display width and forces internal horizontal
// scroll offsets to resynchronize with the current cursor position.
func SetInputWidth(in *textinput.Model, width int) {
	if in == nil {
		return
	}
	width = max(width, 1)
	if in.Width() == width {
		return
	}
	pos := in.Position()
	val := in.Value()
	in.SetWidth(width)
	if val == "" {
		in.SetCursor(0)
		return
	}
	// Reset internal offset/offsetRight before re-applying value and cursor so
	// shrinking width never leaves offsetRight wider than the visible viewport.
	in.SetValue("")
	in.SetWidth(width)
	in.SetValue(val)
	in.SetCursor(pos)
}
