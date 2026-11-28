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
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func assertCursorMatchesRenderedText(t *testing.T, rendered string, cur *tea.Cursor, fieldLabel, typedSuffix string) {
	t.Helper()
	if cur == nil {
		t.Fatalf("expected non-nil cursor for field %q", fieldLabel)
	}
	lines := strings.Split(rendered, "\n")
	if cur.Y < 0 || cur.Y >= len(lines) {
		t.Fatalf("cursor Y=%d out of rendered range 0..%d:\n%s", cur.Y, len(lines)-1, rendered)
	}
	plainLine := ansi.Strip(lines[cur.Y])
	if !strings.Contains(plainLine, fieldLabel+":") {
		t.Fatalf("cursor Y=%d landed on wrong row %q (expected label %q):\n%s", cur.Y, plainLine, fieldLabel, rendered)
	}
	wantSub := typedSuffix
	idx := strings.Index(plainLine, wantSub)
	if idx < 0 {
		t.Fatalf("rendered row %q does not contain %q", plainLine, wantSub)
	}
	wantX := ansi.StringWidth(plainLine[:idx]) + ansi.StringWidth(wantSub)
	if cur.X != wantX {
		t.Fatalf("cursor X mismatch on row %q: got X=%d want X=%d (plain=%q)", plainLine, cur.X, wantX, plainLine)
	}
}

func TestFormModal_CursorExactAlignment(t *testing.T) {
	theme := DefaultTheme()

	// Case 1: With Subtitle and Japanese multi-byte text + odd terminal sizes.
	var m FormModal
	m.SetSize(95, 25, theme)
	m.Open("test.sub", "Edit Board", "Board 12345678", []FormField{
		{Key: "mode", Label: "Mode", Kind: FieldCycle, Options: []string{"override", "suffix"}, Value: "override"},
		{Key: "name", Label: "Name", Kind: FieldText, Value: "日本語テスト", CharLimit: 64},
		{Key: "desc", Label: "Description", Kind: FieldText, CharLimit: 256},
	}, nil)
	_, _, _, _, _, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	view := m.View(95, 25, theme)
	assertCursorMatchesRenderedText(t, view, m.Cursor(95, 25, theme), "Name", "日本語テスト")

	// Type more Japanese text and re-check cursor cell.
	_, _, _, _, _, _ = m.Update(tea.KeyPressMsg{Text: "追加"})
	viewAfter := m.View(95, 25, theme)
	assertCursorMatchesRenderedText(t, viewAfter, m.Cursor(95, 25, theme), "Name", "日本語テスト追加")

	// Case 2: Without Subtitle.
	var mNoSub FormModal
	mNoSub.SetSize(80, 22, theme)
	mNoSub.Open("test.nosub", "Simple Form", "", []FormField{
		{Key: "reason", Label: "Reason", Kind: FieldText, Value: "abc", CharLimit: 120},
	}, nil)
	viewNoSub := mNoSub.View(80, 22, theme)
	assertCursorMatchesRenderedText(t, viewNoSub, mNoSub.Cursor(80, 22, theme), "Reason", "abc")

	// Case 3: Narrow window + scrolling long field list + long prefilled input.
	fields := make([]FormField, 10)
	for i := range fields {
		fields[i] = FormField{Key: "k", Label: "Field", Kind: FieldText, Value: "value"}
	}
	fields[8] = FormField{
		Key:       "target",
		Label:     "Target",
		Kind:      FieldText,
		Value:     strings.Repeat("あ", 30) + "TAIL",
		CharLimit: 256,
	}
	var mScroll FormModal
	mScroll.Open("test.scroll", "Very Long Modal Title Exceeding Narrow Width", "Subtitle also longer than box width for truncation", fields, nil)
	for range 8 {
		_, _, _, _, _, _ = mScroll.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	narrowW, narrowH := 52, 14
	narrowView := mScroll.View(narrowW, narrowH, theme)
	narrowLines := strings.Split(narrowView, "\n")
	if len(narrowLines) > narrowH {
		t.Fatalf("narrow View height %d exceeded %d", len(narrowLines), narrowH)
	}
	for i, l := range narrowLines {
		if w := ansi.StringWidth(l); w > narrowW {
			t.Fatalf("narrow View line %d width %d exceeded %d", i, w, narrowW)
		}
	}
	assertCursorMatchesRenderedText(t, narrowView, mScroll.Cursor(narrowW, narrowH, theme), "Target", "TAIL")
}

func TestConfirmAndSelectModal_NoBackgroundLeakAndFocusIndicator(t *testing.T) {
	theme := DefaultTheme()

	// Modal styles should not emit dark background escapes that break on child \x1b[0m resets.
	var cm ConfirmModal
	cm.Open("tag", "Reload i18n Locales", "Re-reads locale YAML files.", "Execute", true, nil)
	out := cm.View(80, 20, theme)
	if !strings.Contains(out, "▶ [Esc] Cancel") {
		t.Fatalf("expected focused button indicator on Cancel for danger modal, got:\n%s", out)
	}
	if strings.Contains(out, "#1E293B") || strings.Contains(out, "\x1b[48;") {
		t.Fatalf("expected no modal background escape codes causing cutouts, got:\n%q", out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 20 {
		t.Fatalf("ConfirmModal exceeded height 20: %d", len(lines))
	}

	var sm SelectModal
	sm.Open("sel", "Choose Locale", "Pick one", []SelectOption{{Label: "ja", Value: "ja"}}, "ja", nil)
	sout := sm.View(50, 14, theme)
	slines := strings.Split(sout, "\n")
	if len(slines) > 14 {
		t.Fatalf("SelectModal exceeded height 14: %d", len(slines))
	}
}

func TestRenderSubTabBarAndFormatIDWithName(t *testing.T) {
	theme := DefaultTheme()
	bar := RenderSubTabBar(theme, []string{"Reports", "Boards", "Caps"}, 1, 70)
	if ansi.StringWidth(bar) > 70 || !strings.Contains(ansi.Strip(bar), "Boards") || !strings.Contains(ansi.Strip(bar), "subtab") {
		t.Fatalf("unexpected RenderSubTabBar output: %q", ansi.Strip(bar))
	}
	if got := FormatIDWithName("Alice", "123"); got != "Alice (123)" {
		t.Fatalf("unexpected FormatIDWithName: %q", got)
	}
	if got := FormatIDWithName("", "123"); got != "123" {
		t.Fatalf("unexpected FormatIDWithName fallback: %q", got)
	}
	if got := FormatIDWithName("", ""); got != "-" {
		t.Fatalf("unexpected empty FormatIDWithName: %q", got)
	}
}
