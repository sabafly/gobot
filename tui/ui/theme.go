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
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme centralizes colors and Lip Gloss v2 styles across the TUI shell and panels.
type Theme struct {
	AccentColor      color.Color
	CyanColor        color.Color
	SuccessColor     color.Color
	WarnColor        color.Color
	ErrorColor       color.Color
	TextColor        color.Color
	MutedColor       color.Color
	BorderColor      color.Color
	FocusBorderColor color.Color
	HeaderBgColor    color.Color
	SelectedBgColor  color.Color

	Header        lipgloss.Style
	Sidebar       lipgloss.Style
	SidebarItem   lipgloss.Style
	SidebarActive lipgloss.Style
	ContentBox    lipgloss.Style
	Card          lipgloss.Style
	CardTitle     lipgloss.Style
	Label         lipgloss.Style
	Value         lipgloss.Style
	BadgeOK       lipgloss.Style
	BadgeWarn     lipgloss.Style
	BadgeErr      lipgloss.Style
	BadgeInfo     lipgloss.Style
	TableHeader   lipgloss.Style
	TableRow      lipgloss.Style
	TableSelected lipgloss.Style
	Footer        lipgloss.Style
	ToastInfo     lipgloss.Style
	ToastSuccess  lipgloss.Style
	ToastWarn     lipgloss.Style
	ToastError    lipgloss.Style
	ModalBox      lipgloss.Style
	ModalDanger   lipgloss.Style
	HelpBox       lipgloss.Style
	SubTab        lipgloss.Style
	SubTabActive  lipgloss.Style
	InputPrompt   lipgloss.Style
	MutedText     lipgloss.Style
	Button        lipgloss.Style
	ButtonActive  lipgloss.Style
	ButtonDanger  lipgloss.Style
}

// DefaultTheme constructs the default dark palette and styles for gobot's admin TUI.
func DefaultTheme() Theme {
	accent := lipgloss.Color("#5865F2")
	cyan := lipgloss.Color("#38BDF8")
	success := lipgloss.Color("#22C55E")
	warn := lipgloss.Color("#F59E0B")
	errCol := lipgloss.Color("#EF4444")
	text := lipgloss.Color("#E2E8F0")
	muted := lipgloss.Color("#94A3B8")
	border := lipgloss.Color("#334155")
	focusBorder := lipgloss.Color("#5865F2")
	headerBg := lipgloss.Color("#1E293B")
	selectedBg := lipgloss.Color("#312E81")

	return Theme{
		AccentColor:      accent,
		CyanColor:        cyan,
		SuccessColor:     success,
		WarnColor:        warn,
		ErrorColor:       errCol,
		TextColor:        text,
		MutedColor:       muted,
		BorderColor:      border,
		FocusBorderColor: focusBorder,
		HeaderBgColor:    headerBg,
		SelectedBgColor:  selectedBg,

		Header: lipgloss.NewStyle().
			Background(headerBg).
			Foreground(text).
			Padding(0, 1).
			Bold(true),
		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),
		SidebarItem: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),
		SidebarActive: lipgloss.NewStyle().
			Background(selectedBg).
			Foreground(text).
			Bold(true).
			Padding(0, 1),
		ContentBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(focusBorder).
			Padding(0, 1),
		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),
		CardTitle: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		Label: lipgloss.NewStyle().
			Foreground(muted),
		Value: lipgloss.NewStyle().
			Foreground(text),
		BadgeOK: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
		BadgeWarn: lipgloss.NewStyle().
			Foreground(warn).
			Bold(true),
		BadgeErr: lipgloss.NewStyle().
			Foreground(errCol).
			Bold(true),
		BadgeInfo: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		TableHeader: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		TableRow: lipgloss.NewStyle().
			Foreground(text),
		TableSelected: lipgloss.NewStyle().
			Background(selectedBg).
			Foreground(text).
			Bold(true),
		Footer: lipgloss.NewStyle().
			Background(headerBg).
			Foreground(muted).
			Padding(0, 1),
		ToastInfo: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		ToastSuccess: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
		ToastWarn: lipgloss.NewStyle().
			Foreground(warn).
			Bold(true),
		ToastError: lipgloss.NewStyle().
			Foreground(errCol).
			Bold(true),
		ModalBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Background(headerBg).
			Foreground(text).
			Padding(1, 2),
		ModalDanger: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errCol).
			Background(headerBg).
			Foreground(text).
			Padding(1, 2),
		HelpBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyan).
			Background(headerBg).
			Foreground(text).
			Padding(1, 2),
		SubTab: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),
		SubTabActive: lipgloss.NewStyle().
			Background(selectedBg).
			Foreground(text).
			Bold(true).
			Padding(0, 1),
		InputPrompt: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		MutedText: lipgloss.NewStyle().
			Foreground(muted),
		Button: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Foreground(muted).
			Padding(0, 1),
		ButtonActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Background(selectedBg).
			Foreground(text).
			Bold(true).
			Padding(0, 1),
		ButtonDanger: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errCol).
			Background(lipgloss.Color("#7F1D1D")).
			Foreground(text).
			Bold(true).
			Padding(0, 1),
	}
}

// TruncateWidth truncates single-line s so its terminal display width does not exceed maxWidth.
func TruncateWidth(s string, maxWidth int) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " ")
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		var b strings.Builder
		w := 0
		for _, r := range s {
			rw := lipgloss.Width(string(r))
			if w+rw > maxWidth {
				break
			}
			b.WriteRune(r)
			w += rw
		}
		return b.String()
	}

	target := maxWidth - 3
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > target {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	b.WriteString("...")
	return b.String()
}

// PadRightWidth fits s within width display columns and pads trailing spaces to align columns cleanly.
func PadRightWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	t := TruncateWidth(s, width)
	w := lipgloss.Width(t)
	if w < width {
		return t + strings.Repeat(" ", width-w)
	}
	return t
}

// FormatDuration formats seconds into a compact human-readable uptime string.
func FormatDuration(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	secs := seconds % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, mins, secs)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// FormatBytes formats a byte count using binary IEC units.
func FormatBytes(b uint64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)
	switch {
	case b >= gib:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
