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
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/tui/ui"
)

// FieldKind classifies input controls inside a FormModal.
type FieldKind int

const (
	FieldText FieldKind = iota
	FieldBool
	FieldCycle
)

// FormField configures one interactive field in a FormModal.
type FormField struct {
	Key         string
	Label       string
	Kind        FieldKind
	Placeholder string
	CharLimit   int
	Value       string
	BoolVal     bool
	Options     []string
	OptIdx      int
	ReadOnly    bool
	input       textinput.Model
}

// FormModal is a keyboard-driven modal dialog supporting IME text inputs, toggles, and cyclers.
type FormModal struct {
	Active   bool
	Tag      string
	Title    string
	Subtitle string
	Fields   []FormField
	FocusIdx int
	Payload  any
	ErrText  string
}

// Open initializes and displays the form modal, returning the initial focus blink command if any.
func (m *FormModal) Open(tag, title, subtitle string, fields []FormField, payload any) tea.Cmd {
	m.Active = true
	m.Tag = tag
	m.Title = title
	m.Subtitle = subtitle
	m.Payload = payload
	m.ErrText = ""
	m.FocusIdx = 0

	prepared := make([]FormField, len(fields))
	copy(prepared, fields)
	for i := range prepared {
		f := &prepared[i]
		switch f.Kind {
		case FieldText:
			limit := f.CharLimit
			if limit <= 0 {
				limit = 256
			}
			in := ui.NewRealCursorInput(f.Placeholder, limit)
			in.SetValue(f.Value)
			in.SetWidth(42)
			in.Blur()
			f.input = in
		case FieldCycle:
			if len(f.Options) > 0 {
				if f.Value != "" {
					for oi, opt := range f.Options {
						if opt == f.Value {
							f.OptIdx = oi
							break
						}
					}
				}
				if f.OptIdx < 0 || f.OptIdx >= len(f.Options) {
					f.OptIdx = 0
				}
				f.Value = f.Options[f.OptIdx]
			}
		case FieldBool:
			if strings.EqualFold(strings.TrimSpace(f.Value), "true") {
				f.BoolVal = true
			}
			if f.BoolVal {
				f.Value = "true"
			} else {
				f.Value = "false"
			}
		}
	}
	m.Fields = prepared

	// Focus first editable field.
	for i := range m.Fields {
		if !m.Fields[i].ReadOnly {
			m.FocusIdx = i
			break
		}
	}
	if len(m.Fields) > 0 && m.Fields[m.FocusIdx].Kind == FieldText && !m.Fields[m.FocusIdx].ReadOnly {
		return m.Fields[m.FocusIdx].input.Focus()
	}
	return nil
}

// Close hides the modal and clears active state.
func (m *FormModal) Close() {
	for i := range m.Fields {
		if m.Fields[i].Kind == FieldText {
			m.Fields[i].input.Blur()
		}
	}
	m.Active = false
	m.Tag = ""
	m.Fields = nil
	m.Payload = nil
	m.ErrText = ""
}

// Values returns a snapshot map of field keys to current string values.
func (m *FormModal) Values() map[string]string {
	out := make(map[string]string, len(m.Fields))
	for i := range m.Fields {
		f := &m.Fields[i]
		switch f.Kind {
		case FieldText:
			out[f.Key] = f.input.Value()
		case FieldBool:
			if f.BoolVal {
				out[f.Key] = "true"
			} else {
				out[f.Key] = "false"
			}
		case FieldCycle:
			if len(f.Options) > 0 && f.OptIdx >= 0 && f.OptIdx < len(f.Options) {
				out[f.Key] = f.Options[f.OptIdx]
			} else {
				out[f.Key] = f.Value
			}
		}
	}
	return out
}

// Update handles keyboard and IME events while FormModal is active.
func (m *FormModal) Update(msg tea.Msg) (submitted bool, canceled bool, tag string, values map[string]string, payload any, cmd tea.Cmd) {
	if !m.Active {
		return false, false, "", nil, nil, nil
	}

	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc":
			tag = m.Tag
			payload = m.Payload
			m.Close()
			return false, true, tag, nil, payload, nil
		case "enter", "ctrl+s":
			tag = m.Tag
			values = m.Values()
			payload = m.Payload
			m.Close()
			return true, false, tag, values, payload, nil
		case "up", "shift+tab":
			return false, false, "", nil, nil, m.moveFocus(-1)
		case "down", "tab":
			return false, false, "", nil, nil, m.moveFocus(1)
		}

		if m.FocusIdx >= 0 && m.FocusIdx < len(m.Fields) {
			f := &m.Fields[m.FocusIdx]
			if !f.ReadOnly {
				switch f.Kind {
				case FieldBool:
					switch k.String() {
					case " ", "space", "left", "right", "h", "l":
						f.BoolVal = !f.BoolVal
						return false, false, "", nil, nil, nil
					case "y":
						f.BoolVal = true
						return false, false, "", nil, nil, nil
					case "n":
						f.BoolVal = false
						return false, false, "", nil, nil, nil
					}
				case FieldCycle:
					if len(f.Options) > 0 {
						switch k.String() {
						case " ", "space", "right", "l":
							f.OptIdx = (f.OptIdx + 1) % len(f.Options)
							return false, false, "", nil, nil, nil
						case "left", "h":
							f.OptIdx = (f.OptIdx + len(f.Options) - 1) % len(f.Options)
							return false, false, "", nil, nil, nil
						}
					}
				}
			}
		}
	}

	// Forward remaining events (including non-key IME composition / blink msgs) to focused textinput.
	if m.FocusIdx >= 0 && m.FocusIdx < len(m.Fields) {
		f := &m.Fields[m.FocusIdx]
		if f.Kind == FieldText && !f.ReadOnly {
			var c tea.Cmd
			f.input, c = f.input.Update(msg)
			return false, false, "", nil, nil, c
		}
	}
	return false, false, "", nil, nil, nil
}

func (m *FormModal) moveFocus(delta int) tea.Cmd {
	n := len(m.Fields)
	if n == 0 {
		return nil
	}
	if m.FocusIdx >= 0 && m.FocusIdx < n && m.Fields[m.FocusIdx].Kind == FieldText {
		m.Fields[m.FocusIdx].input.Blur()
	}
	next := m.FocusIdx
	for range n {
		next = (next + delta + n) % n
		if !m.Fields[next].ReadOnly {
			break
		}
	}
	m.FocusIdx = next
	if m.Fields[next].Kind == FieldText && !m.Fields[next].ReadOnly {
		return m.Fields[next].input.Focus()
	}
	return nil
}

func (m *FormModal) layoutMetrics(width, height int, theme ui.Theme) (boxW, labelW, inputW, startIdx, endIdx int) {
	boxW = max(min(width-4, 76), 38)
	if width > 0 && boxW > width {
		boxW = width
	}
	innerW := max(boxW-theme.ModalBox.GetHorizontalFrameSize(), 20)
	labelW = min(max(innerW*28/100, 12), 18)
	inputW = max(innerW-labelW-2, 12)

	overheadLines := 4 // title + blank + blank + help footer
	if m.Subtitle != "" {
		overheadLines += 2
	}
	if m.ErrText != "" {
		overheadLines += 2
	}
	maxFields := len(m.Fields)
	if height > 0 {
		maxFields = max((height-theme.ModalBox.GetVerticalFrameSize()-overheadLines)/1, 1)
	}
	startIdx = 0
	if m.FocusIdx >= maxFields {
		startIdx = m.FocusIdx - maxFields + 1
	}
	endIdx = min(len(m.Fields), startIdx+maxFields)
	return boxW, labelW, inputW, startIdx, endIdx
}

// Cursor translates the focused text field's hardware/IME cursor to panel coordinates.
func (m *FormModal) Cursor(width, height int, theme ui.Theme) *tea.Cursor {
	if !m.Active || m.FocusIdx < 0 || m.FocusIdx >= len(m.Fields) {
		return nil
	}
	f := &m.Fields[m.FocusIdx]
	if f.Kind != FieldText || f.ReadOnly || !f.input.Focused() {
		return nil
	}
	boxW, labelW, inputW, startIdx, endIdx := m.layoutMetrics(width, height, theme)
	if m.FocusIdx < startIdx || m.FocusIdx >= endIdx {
		return nil
	}
	f.input.SetWidth(inputW)
	c := f.input.Cursor()
	if c == nil {
		return nil
	}
	box := m.renderBox(boxW, labelW, inputW, startIdx, endIdx, theme)
	renderedW := lipgloss.Width(box)
	renderedH := lipgloss.Height(box)
	placeX := 0
	placeY := 0
	if width > renderedW {
		placeX = (width - renderedW) / 2
	}
	if height > renderedH {
		placeY = (height - renderedH) / 2
	}

	leftFrame := theme.ModalBox.GetBorderLeftSize() + theme.ModalBox.GetPaddingLeft()
	topFrame := theme.ModalBox.GetBorderTopSize() + theme.ModalBox.GetPaddingTop()
	headerRows := 2 // Title + blank line
	if m.Subtitle != "" {
		headerRows += 2
	}
	if startIdx > 0 {
		headerRows++
	}
	rowOffset := m.FocusIdx - startIdx

	cur := *c
	cur.X += placeX + leftFrame + labelW + 2
	cur.Y += placeY + topFrame + headerRows + rowOffset
	return &cur
}

func (m *FormModal) renderBox(boxW, labelW, inputW int, startIdx, endIdx int, theme ui.Theme) string {
	lines := []string{theme.CardTitle.Render(ui.TruncateWidth(m.Title, max(boxW-8, 12)))}
	if m.Subtitle != "" {
		lines = append(lines, theme.MutedText.Render(ui.TruncateWidth(m.Subtitle, max(boxW-8, 12))))
	}
	lines = append(lines, "")
	if startIdx > 0 {
		lines = append(lines, theme.MutedText.Render(fmt.Sprintf("  ▲ %d more above", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		f := &m.Fields[i]
		focused := i == m.FocusIdx
		lbl := ui.PadRightWidth(f.Label+":", labelW)
		lblStyle := theme.Label
		if focused {
			lblStyle = theme.InputPrompt
		}
		var valView string
		switch f.Kind {
		case FieldText:
			f.input.SetWidth(inputW)
			if f.ReadOnly {
				valView = theme.MutedText.Render(ui.TruncateWidth(f.input.Value(), inputW))
			} else {
				valView = ui.TruncateWidth(f.input.View(), inputW)
			}
		case FieldBool:
			txt := "[ ] No"
			if f.BoolVal {
				txt = "[x] Yes"
			}
			if focused {
				valView = theme.BadgeInfo.Render(txt)
			} else {
				valView = theme.Value.Render(txt)
			}
		case FieldCycle:
			curOpt := f.Value
			if len(f.Options) > 0 && f.OptIdx >= 0 && f.OptIdx < len(f.Options) {
				curOpt = f.Options[f.OptIdx]
			}
			txt := "< " + curOpt + " >"
			if focused {
				valView = theme.BadgeInfo.Render(ui.TruncateWidth(txt, inputW))
			} else {
				valView = theme.Value.Render(ui.TruncateWidth(txt, inputW))
			}
		}
		lines = append(lines, lblStyle.Render(lbl)+"  "+valView)
	}

	if endIdx < len(m.Fields) {
		lines = append(lines, theme.MutedText.Render(fmt.Sprintf("  ▼ %d more below", len(m.Fields)-endIdx)))
	}
	if m.ErrText != "" {
		lines = append(lines, "", theme.BadgeErr.Render(ui.TruncateWidth(m.ErrText, max(boxW-8, 12))))
	}
	lines = append(lines, "", theme.MutedText.Render("[Tab/↑↓] Field  [Space/←→] Toggle  [Enter] Submit  [Esc] Cancel"))

	return theme.ModalBox.Width(boxW).Render(strings.Join(lines, "\n"))
}

// View renders the form modal centered within width x height.
func (m *FormModal) View(width, height int, theme ui.Theme) string {
	if !m.Active {
		return ""
	}
	boxW, labelW, inputW, startIdx, endIdx := m.layoutMetrics(width, height, theme)
	box := m.renderBox(boxW, labelW, inputW, startIdx, endIdx, theme)
	if width > 0 && height > 0 {
		return ui.ClampBox(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box), width, height)
	}
	return box
}
