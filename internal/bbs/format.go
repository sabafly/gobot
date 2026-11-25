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
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const MaxAnchorsPerPost = 10
const MaxMediaPerPost = 10

var (
	refRegex   = regexp.MustCompile(`\\?>\\?>([1-9]\d{0,3})`)
	urlRegex   = regexp.MustCompile(`https?://(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&/=]*)`)
	quoteRegex = regexp.MustCompile(`(?m)^(>|>{3})\s`)
)

var escapeChars = []struct {
	from string
	to   string
}{
	{"`", "\\`"},
	{"*", "\\*"},
	{"_", "\\_"},
	{"~", "\\~"},
	{"|", "\\|"},
	{":", "\\:"},
	{"@", "\\@"},
	{"<", "\\<"},
	{"[", "\\["},
	{".", "\\."},
	{"#", "\\#"},
	{"-", "\\-"},
}

var mediaExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".webp": {},
	".avif": {},
	".mp4":  {},
	".webm": {},
	".mov":  {},
}

var cdnMediaFormats = map[string]struct{}{
	"png":  {},
	"jpg":  {},
	"jpeg": {},
	"gif":  {},
	"webp": {},
}

// ClearFormat escapes Discord markdown, mentions, and quotes while preserving URLs inside <...>
// to prevent unwanted legacy auto-embeds alongside MediaGallery components.
func ClearFormat(str string) string {
	str = strings.ReplaceAll(str, "\uEFFD", "")
	str = strings.ReplaceAll(str, "\uEFFF", "")

	var urls []string
	str = urlRegex.ReplaceAllStringFunc(str, func(s string) string {
		urls = append(urls, s)
		return fmt.Sprintf("\uEFFD%d\uEFFF", len(urls)-1)
	})

	str = strings.ReplaceAll(str, "\\", "\\\\")
	for _, esc := range escapeChars {
		str = strings.ReplaceAll(str, esc.from, esc.to)
	}
	str = quoteRegex.ReplaceAllStringFunc(str, func(s string) string {
		return "\\" + s
	})

	for i, u := range urls {
		str = strings.ReplaceAll(str, fmt.Sprintf("\uEFFD%d\uEFFF", i), fmt.Sprintf("<%s>", u))
	}
	return str
}

// ExtractAnchors extracts unique >>N references (up to MaxAnchorsPerPost and maxRes) in order of appearance.
func ExtractAnchors(rawBody string, maxRes uint32) []uint32 {
	if maxRes == 0 {
		maxRes = DefaultMaxRes
	}
	matches := refRegex.FindAllStringSubmatch(rawBody, -1)
	seen := make(map[uint32]struct{}, len(matches))
	result := make([]uint32, 0, min(len(matches), MaxAnchorsPerPost))
	for _, m := range matches {
		if len(result) >= MaxAnchorsPerPost {
			break
		}
		if len(m) < 2 {
			continue
		}
		val, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil || val == 0 || uint32(val) > maxRes {
			continue
		}
		n := uint32(val)
		if _, exists := seen[n]; exists {
			continue
		}
		seen[n] = struct{}{}
		result = append(result, n)
	}
	return result
}

// RenderFormattedBody escapes raw post text and converts >>N anchors into clickable jump links
// whenever resolveURL returns a target message URL for the given guild.
func RenderFormattedBody(rawBody string, maxRes uint32, resolveURL func(resNum uint32) (string, bool)) string {
	if maxRes == 0 {
		maxRes = DefaultMaxRes
	}
	escaped := ClearFormat(rawBody)
	count := 0
	return refRegex.ReplaceAllStringFunc(escaped, func(match string) string {
		count++
		if count > MaxAnchorsPerPost {
			return match
		}
		sub := refRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		val, err := strconv.ParseUint(sub[1], 10, 32)
		if err != nil || val == 0 || uint32(val) > maxRes {
			return match
		}
		n := uint32(val)
		if resolveURL == nil {
			return fmt.Sprintf(">>%d", n)
		}
		targetURL, ok := resolveURL(n)
		if !ok || targetURL == "" {
			return fmt.Sprintf(">>%d", n)
		}
		return fmt.Sprintf("[>>%d](<%s>)", n, targetURL)
	})
}

// ExtractMediaURLs finds unique http/https image or video URLs in raw post text (capped at MaxMediaPerPost)
// without performing any server-side HTTP fetches.
func ExtractMediaURLs(rawBody string) []string {
	rawMatches := urlRegex.FindAllString(rawBody, -1)
	if len(rawMatches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rawMatches))
	out := make([]string, 0, min(len(rawMatches), MaxMediaPerPost))
	for _, raw := range rawMatches {
		if len(out) >= MaxMediaPerPost {
			break
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			continue
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			continue
		}
		if !isSupportedMediaURL(u) {
			continue
		}
		norm := u.String()
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out
}

func isSupportedMediaURL(u *url.URL) bool {
	ext := strings.ToLower(path.Ext(u.Path))
	if _, ok := mediaExtensions[ext]; ok {
		return true
	}
	host := strings.ToLower(u.Hostname())
	if host == "pbs.twimg.com" || host == "media.discordapp.net" || host == "cdn.discordapp.com" {
		if format := strings.ToLower(u.Query().Get("format")); format != "" {
			_, ok := cdnMediaFormats[format]
			return ok
		}
	}
	return false
}
