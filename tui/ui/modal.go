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
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmModal asks the operator to explicitly confirm or cancel a state-changing operation.
type ConfirmModal struct {
	Active       bool
	Title        string
	Body         string
	ConfirmLabel string
	CancelLabel  string
	Danger       bool
	FocusConfirm bool
	Tag          string
	Payload      any
}

// Open activates the confirmation modal with the provided prompt and metadata.
func (m *ConfirmModal) Open(tag, title, body, confirmLabel string, danger bool, payload any) {
	if confirmLabel == "" {
		confirmLabel = "Confirm"
	}
	m.Active = true
	m.Tag = tag
	m.Title = title
	m.Body = body
	m.ConfirmLabel = confirmLabel
	m.CancelLabel = "Cancel"
	m.Danger = danger
	m.FocusConfirm = !danger
	m.Payload = payload
}

// Reset closes and clears the confirmation modal state.
func (m *ConfirmModal) Reset() {
	m.Active = false
	m.Tag = ""
	m.Payload = nil
}

// Update handles keyboard navigation inside the active modal.
func (m *ConfirmModal) Update(msg tea.Msg) (confirmed bool, canceled bool, tag string, payload any) {
	if !m.Active {
		return false, false, "", nil
	}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, false, "", nil
	}
	switch kp.String() {
	case "left", "right", "h", "l", "tab":
		m.FocusConfirm = !m.FocusConfirm
	case "y", "Y":
		tag, payload = m.Tag, m.Payload
		m.Reset()
		return true, false, tag, payload
	case "n", "N", "esc":
		tag, payload = m.Tag, m.Payload
		m.Reset()
		return false, true, tag, payload
	case "enter":
		tag, payload = m.Tag, m.Payload
		confirm := m.FocusConfirm
		m.Reset()
		return confirm, !confirm, tag, payload
	}
	return false, false, "", nil
}

// View renders the confirmation modal centered within width x height.
func (m *ConfirmModal) View(width, height int, theme Theme) string {
	if !m.Active {
		return ""
	}
	confirmLbl := m.ConfirmLabel
	if confirmLbl == "" {
		confirmLbl = "Confirm"
	}
	cancelLbl := m.CancelLabel
	if cancelLbl == "" {
		cancelLbl = "Cancel"
	}

	confirmStyle := theme.Button
	cancelStyle := theme.Button
	if m.FocusConfirm {
		if m.Danger {
			confirmStyle = theme.ButtonDanger
		} else {
			confirmStyle = theme.ButtonActive
		}
	} else {
		cancelStyle = theme.ButtonActive
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		confirmStyle.Render("[Enter] "+confirmLbl),
		"  ",
		cancelStyle.Render("[Esc] "+cancelLbl),
	)

	maxBoxW := max(min(width-4, 68), 32)
	titleStyle := theme.CardTitle
	if m.Danger {
		titleStyle = theme.BadgeErr
	}
	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(m.Title),
		"",
		lipgloss.NewStyle().Width(maxBoxW-6).Render(m.Body),
		"",
		buttons,
		"",
		theme.MutedText.Render("←/→ or Tab: switch  |  y: confirm  |  n/Esc: cancel"),
	)

	boxStyle := theme.ModalBox
	if m.Danger {
		boxStyle = theme.ModalDanger
	}
	box := boxStyle.Width(maxBoxW).Render(inner)
	if width > 0 && height > 0 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

// SelectOption represents an item selectable within SelectModal.
type SelectOption struct {
	Label       string
	Value       string
	Description string
}

// SelectModal displays a scrollable option list centered inside the content area.
type SelectModal struct {
	Active   bool
	Title    string
	Subtitle string
	Options  []SelectOption
	Cursor   int
	Tag      string
	Payload  any
}

// Open activates the selection modal with the given options and metadata.
func (m *SelectModal) Open(tag, title, subtitle string, opts []SelectOption, currentVal string, payload any) {
	m.Active = true
	m.Tag = tag
	m.Title = title
	m.Subtitle = subtitle
	m.Options = opts
	m.Cursor = 0
	for i, o := range opts {
		if o.Value == currentVal {
			m.Cursor = i
			break
		}
	}
	m.Tag = tag
	m.Payload = payload
}

// Reset closes and clears the selection modal state.
func (m *SelectModal) Reset() {
	m.Active = false
	m.Options = nil
	m.Cursor = 0
	m.Tag = ""
	m.Payload = nil
}

// Update handles navigation and selection key presses.
func (m *SelectModal) Update(msg tea.Msg) (selected *SelectOption, canceled bool, tag string, payload any) {
	if !m.Active {
		return nil, false, "", nil
	}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, "", nil
	}
	switch kp.String() {
	case "j", "down":
		if m.Cursor < len(m.Options)-1 {
			m.Cursor++
		}
	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "esc", "q":
		tag, payload = m.Tag, m.Payload
		m.Reset()
		return nil, true, tag, payload
	case "enter":
		if len(m.Options) > 0 && m.Cursor >= 0 && m.Cursor < len(m.Options) {
			opt := m.Options[m.Cursor]
			tag, payload = m.Tag, m.Payload
			m.Reset()
			return &opt, false, tag, payload
		}
	}
	return nil, false, "", nil
}

// View renders the option picker centered within width x height.
func (m *SelectModal) View(width, height int, theme Theme) string {
	if !m.Active {
		return ""
	}
	maxBoxW := max(min(width-4, 68), 36)
	innerW := maxBoxW - 6

	maxVisible := max(min(height-10, 12), 4)
	start := 0
	if m.Cursor >= maxVisible {
		start = m.Cursor - maxVisible + 1
	}
	end := min(len(m.Options), start+maxVisible)

	var rows []string
	for i := start; i < end; i++ {
		opt := m.Options[i]
		line := opt.Label
		if opt.Description != "" && opt.Description != opt.Label {
			line = fmt.Sprintf("%s (%s)", opt.Label, opt.Description)
		}
		prefix := "  "
		style := theme.TableRow
		if i == m.Cursor {
			prefix = "> "
			style = theme.TableSelected
		}
		rows = append(rows, style.Render(PadRightWidth(prefix+line, innerW)))
	}
	if len(rows) == 0 {
		rows = append(rows, theme.MutedText.Render("(no options)"))
	}

	parts := []string{
		theme.CardTitle.Render(m.Title),
	}
	if m.Subtitle != "" {
		parts = append(parts, theme.MutedText.Render(TruncateWidth(m.Subtitle, innerW)))
	}
	parts = append(parts, "", strings.Join(rows, "\n"), "", theme.MutedText.Render("↑/↓: move  |  Enter: select  |  Esc: cancel"))

	box := theme.ModalBox.Width(maxBoxW).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	if width > 0 && height > 0 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}
