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
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type parsedSlipEvent struct {
	id       string
	label    string
	mode     SlipEventMode
	annual   bool
	startMD  int
	endMD    int
	startYMD int
	endYMD   int
}

func (e parsedSlipEvent) matches(jst time.Time) bool {
	if e.annual {
		curMD := int(jst.Month())*100 + jst.Day()
		if e.startMD <= e.endMD {
			return curMD >= e.startMD && curMD <= e.endMD
		}
		// Annual range wraps across year boundary (e.g. 12-29..01-03).
		return curMD >= e.startMD || curMD <= e.endMD
	}
	curYMD := jst.Year()*10000 + int(jst.Month())*100 + jst.Day()
	return curYMD >= e.startYMD && curYMD <= e.endYMD
}

var builtinParsedSlipEvents = []parsedSlipEvent{
	{id: "new_year", label: "ｵｼｮｳｶﾞﾂ", mode: SlipEventModeAppend, annual: true, startMD: 101, endMD: 103},
	{id: "setsubun", label: "ｾﾂﾌﾞﾝ", mode: SlipEventModeAppend, annual: true, startMD: 203, endMD: 203},
	{id: "valentine", label: "ﾊﾞﾚﾝﾀｲﾝ", mode: SlipEventModeAppend, annual: true, startMD: 214, endMD: 214},
	{id: "cat_day", label: "ﾆｬﾝﾆｬﾝ", mode: SlipEventModeAppend, annual: true, startMD: 222, endMD: 222},
	{id: "april_fools", label: "ｳｿﾂｷ", mode: SlipEventModeOverride, annual: true, startMD: 401, endMD: 401},
	{id: "kodomo_day", label: "ｺｲﾉﾎﾞﾘ", mode: SlipEventModeAppend, annual: true, startMD: 505, endMD: 505},
	{id: "tanabata", label: "ﾀﾅﾊﾞﾀ", mode: SlipEventModeAppend, annual: true, startMD: 707, endMD: 707},
	{id: "halloween", label: "ﾊﾛｳｨﾝ", mode: SlipEventModeAppend, annual: true, startMD: 1031, endMD: 1031},
	{id: "pocky_day", label: "ﾎﾟｯｷｰ", mode: SlipEventModeAppend, annual: true, startMD: 1111, endMD: 1111},
	{id: "christmas", label: "ﾒﾘｸﾘ", mode: SlipEventModeAppend, annual: true, startMD: 1224, endMD: 1225},
	{id: "omisoka", label: "ｵｵﾐｿｶ", mode: SlipEventModeAppend, annual: true, startMD: 1231, endMD: 1231},
}

func parseSlipDateSpec(raw string) (annual bool, md int, ymd int, ok bool) {
	s := SanitizeSingleLine(raw)
	switch len(s) {
	case 5: // MM-DD
		t, err := time.Parse("2006-01-02", "2024-"+s)
		if err != nil {
			return false, 0, 0, false
		}
		return true, int(t.Month())*100 + t.Day(), 0, true
	case 10: // YYYY-MM-DD
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return false, 0, 0, false
		}
		return false, 0, t.Year()*10000 + int(t.Month())*100 + t.Day(), true
	default:
		return false, 0, 0, false
	}
}

func parseSlipEventDefinitions(defs []SlipEventDefinition) ([]SlipEventDefinition, []parsedSlipEvent, error) {
	if len(defs) == 0 {
		return nil, nil, nil
	}
	seenIDs := make(map[string]struct{}, len(defs))
	norm := make([]SlipEventDefinition, 0, len(defs))
	parsed := make([]parsedSlipEvent, 0, len(defs))

	for _, def := range defs {
		id := SanitizeSingleLine(def.ID)
		if id == "" || utf8.RuneCountInString(id) > MaxSlipEventIDRunes {
			return nil, nil, fmt.Errorf("%w: invalid slip event id %q", ErrInvalidSlipConfig, def.ID)
		}
		if _, dup := seenIDs[id]; dup {
			return nil, nil, fmt.Errorf("%w: duplicate slip event id %q", ErrInvalidSlipConfig, id)
		}
		seenIDs[id] = struct{}{}

		label := SanitizeDisplayName(def.Label)
		if label == "" || strings.TrimSpace(ClearFormat(label)) == "" || utf8.RuneCountInString(label) > MaxSlipEventLabelRunes {
			return nil, nil, fmt.Errorf("%w: invalid slip event label for %q", ErrInvalidSlipConfig, id)
		}

		mode := SlipEventMode(strings.ToLower(SanitizeSingleLine(string(def.Mode))))
		if mode == "" {
			mode = SlipEventModeAppend
		}
		if mode != SlipEventModeAppend && mode != SlipEventModeOverride {
			return nil, nil, fmt.Errorf("%w: unsupported slip event mode %q for %q", ErrInvalidSlipConfig, def.Mode, id)
		}

		startRaw := SanitizeSingleLine(def.StartDate)
		endRaw := SanitizeSingleLine(def.EndDate)
		if endRaw == "" {
			endRaw = startRaw
		}
		startAnnual, startMD, startYMD, startOK := parseSlipDateSpec(startRaw)
		endAnnual, endMD, endYMD, endOK := parseSlipDateSpec(endRaw)
		if !startOK || !endOK || startAnnual != endAnnual {
			return nil, nil, fmt.Errorf("%w: invalid date range (%q..%q) for %q", ErrInvalidSlipConfig, def.StartDate, def.EndDate, id)
		}
		if !startAnnual && endYMD < startYMD {
			return nil, nil, fmt.Errorf("%w: end_date before start_date (%q..%q) for %q", ErrInvalidSlipConfig, startRaw, endRaw, id)
		}

		norm = append(norm, SlipEventDefinition{
			ID:        id,
			Label:     label,
			StartDate: startRaw,
			EndDate:   endRaw,
			Mode:      mode,
		})
		parsed = append(parsed, parsedSlipEvent{
			id:       id,
			label:    label,
			mode:     mode,
			annual:   startAnnual,
			startMD:  startMD,
			endMD:    endMD,
			startYMD: startYMD,
			endYMD:   endYMD,
		})
	}
	return norm, parsed, nil
}

// ValidateSlipEventDefinitions normalizes and validates static date-based SLIP event definitions.
func ValidateSlipEventDefinitions(defs []SlipEventDefinition) ([]SlipEventDefinition, error) {
	norm, _, err := parseSlipEventDefinitions(defs)
	return norm, err
}

// ResolveKorokoroName selects the Discord display name input used for KOROKORO hash derivation.
// Local boards prefer server nickname -> global display name -> username;
// global boards prefer global display name -> username.
func ResolveKorokoroName(scope Scope, sc SlipContext) string {
	candidates := []string{sc.GlobalName, sc.Username}
	if scope == ScopeLocal {
		candidates = []string{sc.MemberNick, sc.GlobalName, sc.Username}
	}
	for _, c := range candidates {
		if trimmed := SanitizeSingleLine(c); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatAlphabet4(sum [32]byte) string {
	var b strings.Builder
	b.Grow(4)
	for i := range 4 {
		b.WriteByte(idAlphabet[int(sum[i])%len(idAlphabet)])
	}
	return b.String()
}

// GenerateSlipCode derives a MMMM-XXXX SLIP code where MMMM is fixed per user ID
// and XXXX (KOROKORO) varies with the user's Discord display name.
func (g *IdentityGenerator) GenerateSlipCode(scope Scope, userID snowflake.ID, sc SlipContext) string {
	mainSum := g.hmacSum(fmt.Sprintf("slip-main:%d", userID))
	koroName := ResolveKorokoroName(scope, sc)
	koroSum := g.hmacSum(fmt.Sprintf("slip-koro:%s:%d:%s", scope, userID, koroName))
	return formatAlphabet4(mainSum) + "-" + formatAlphabet4(koroSum)
}

func pickSlipWord(sum [32]byte, offset int, words []string) string {
	return words[int(sum[offset%len(sum)])%len(words)]
}

func matchSlipEvent(custom []parsedSlipEvent, jst time.Time) (string, SlipEventMode, bool) {
	for _, ev := range custom {
		if ev.matches(jst) {
			return ev.label, ev.mode, true
		}
	}
	for _, ev := range builtinParsedSlipEvents {
		if ev.matches(jst) {
			return ev.label, ev.mode, true
		}
	}
	if jst.Day() == 13 && jst.Weekday() == time.Friday {
		return "ｼﾞｪｲｿﾝ", SlipEventModeAppend, true
	}
	if jst.Day() == 29 {
		return "ﾆｸﾉﾋ", SlipEventModeAppend, true
	}
	return "", "", false
}

// GenerateSlipNickname produces a playful 5ch-style half-width Katakana nickname
// derived from anonymous Discord traits, JST time/weekday/rare rolls, and date events.
func (g *IdentityGenerator) GenerateSlipNickname(
	boardID uuid.UUID,
	scope Scope,
	guildID snowflake.ID,
	userID snowflake.ID,
	sc SlipContext,
	at time.Time,
) string {
	loc := g.loc
	if loc == nil {
		loc = JSTLocation
	}
	if at.IsZero() {
		at = time.Now()
	}
	jst := at.In(loc)

	attrSum := g.hmacSum(fmt.Sprintf("slip-attr:%s:%s:%d:%d", scope, boardID, guildID, userID))
	timeSum := g.hmacSum(fmt.Sprintf("slip-time:%s:%d:%s", boardID, userID, jst.Format("2006-01-02")))

	var base string
	hhmm := jst.Hour()*100 + jst.Minute()
	switch hhmm {
	case 0:
		base = "ｼﾝﾃﾞﾚﾗ"
	case 1111, 2222:
		base = "ｿﾞﾛﾒﾉﾀﾐ"
	}

	if base == "" {
		rareSum := g.hmacSum(fmt.Sprintf("slip-rare:%s:%d:%s", boardID, userID, jst.Format("2006-01-02 15:04")))
		if rareSum[0] < 4 {
			base = pickSlipWord(rareSum, 1, []string{"ｷﾝﾉｻﾊﾞ", "ﾂﾁﾉｺ", "ﾏﾎﾞﾛｼ"})
		}
	}

	if base == "" && scope == ScopeLocal && sc.InVoiceChannel {
		switch {
		case sc.IsVoiceStreaming:
			base = pickSlipWord(attrSum, 0, []string{"ﾅﾏﾊｲｼﾝ", "ｶﾒﾗﾏﾝ"})
		case sc.IsVoiceMutedOrDeaf:
			base = pickSlipWord(attrSum, 0, []string{"ｷｷｾﾝ", "ﾑｺﾞﾝﾂｳﾜ"})
		default:
			base = pickSlipWord(attrSum, 0, []string{"ﾂｳﾜﾁｭｳ", "ﾎﾞｲﾁｬﾐﾝ"})
		}
	}

	if base == "" {
		hour := jst.Hour()
		if hour >= 2 && hour <= 4 {
			base = pickSlipWord(timeSum, 0, []string{"ﾖﾌｶｼ", "ｳｼﾐﾂ", "ﾌｸﾛｳ"})
		} else if (jst.Weekday() == time.Saturday || jst.Weekday() == time.Sunday) && hour >= 5 && hour <= 8 {
			base = "ﾊﾔｵｷ"
		}
	}

	if base == "" {
		var userCandidates []string
		locNorm := strings.ToLower(strings.TrimSpace(sc.UserLocale))
		if locNorm != "" && !strings.HasPrefix(locNorm, "ja") {
			userCandidates = append(userCandidates, "ｶｲｶﾞｲ", "ｵｰﾊﾞｰｼｰ")
		}
		if sc.HasSpecialBadge {
			userCandidates = append(userCandidates, "ｴﾘｰﾄ", "ﾊﾞｯｼﾞﾓﾁ")
		}
		if sc.IsDefaultAvatar {
			userCandidates = append(userCandidates, "ﾃﾞﾌｫｱｲｺﾝ", "ﾀﾏｺﾞ")
		}
		if sc.HasProfileDecor {
			userCandidates = append(userCandidates, "ｵﾒｶｼ", "ﾃﾞｺﾓﾘ")
		}
		if !sc.UserCreatedAt.IsZero() {
			age := jst.Sub(sc.UserCreatedAt)
			switch {
			case age >= 8*365*24*time.Hour:
				userCandidates = append(userCandidates, "ｲﾆｼｴ", "ｶｾｷ")
			case age >= 5*365*24*time.Hour:
				userCandidates = append(userCandidates, "ｺｻﾝｾﾞｲ", "ﾍﾞﾃﾗﾝ")
			case age >= 0 && age < 365*24*time.Hour:
				userCandidates = append(userCandidates, "ｼﾝｻﾞﾝ", "ﾜｶﾊﾞ")
			}
		}
		switch sc.HypeSquadHouse {
		case "bravery":
			userCandidates = append(userCandidates, "ﾌﾞﾚｲﾌﾞ", "ﾕｳｶﾝ")
		case "brilliance":
			userCandidates = append(userCandidates, "ﾌﾞﾘﾘｱﾝｽ", "ｻｴﾜﾀﾙ")
		case "balance":
			userCandidates = append(userCandidates, "ﾊﾞﾗﾝｻｰ", "ﾁｮｳﾜ")
		}

		if scope == ScopeLocal {
			var localCandidates []string
			if sc.IsGuildBooster {
				localCandidates = append(localCandidates, "ﾌﾞｰｽﾄｳｵ", "ﾆﾄﾛｾﾞｲ")
			}
			if sc.MemberJoinedAt != nil {
				joinAge := jst.Sub(*sc.MemberJoinedAt)
				switch {
				case joinAge >= 0 && joinAge < 7*24*time.Hour:
					localCandidates = append(localCandidates, "ﾋﾖｯｺ", "ｼﾝｲﾘ")
				case joinAge >= 2*365*24*time.Hour:
					localCandidates = append(localCandidates, "ﾇｼ", "ﾁｮｳﾛｳ")
				}
			}
			if sc.RoleCount >= 10 {
				localCandidates = append(localCandidates, "ｸﾝｼｮｳ", "ｼﾞｭｳｿｳﾋﾞ")
			} else if sc.HasZeroRoles {
				localCandidates = append(localCandidates, "ﾑｼﾞﾙｼ", "ｽｯﾋﾟﾝ")
			}
			if sc.HasServerNick || sc.HasServerAvatar {
				localCandidates = append(localCandidates, "ﾍﾝｿｳﾁｭｳ", "ｶﾒﾝ")
			}
			if len(localCandidates) > 0 {
				base = pickSlipWord(attrSum, 1, localCandidates)
			} else if len(userCandidates) > 0 {
				base = pickSlipWord(attrSum, 2, userCandidates)
			}
		} else {
			var guildCandidates []string
			if sc.GuildMemberCount >= 500 || sc.GuildHasVanityURL {
				guildCandidates = append(guildCandidates, "ｵｵﾃｻﾊﾞ", "ﾀﾞｲﾄｶｲ")
			} else if sc.GuildMemberCount > 0 && sc.GuildMemberCount <= 20 {
				guildCandidates = append(guildCandidates, "ﾋﾐﾂｷﾁ", "ｶｸﾚｶﾞ")
			}
			if sc.GuildBoostTier >= 3 {
				guildCandidates = append(guildCandidates, "ｺﾞｰｼﾞｬｽ", "ｷﾗｷﾗｻﾊﾞ")
			} else if sc.GuildBoostTier >= 1 {
				guildCandidates = append(guildCandidates, "ﾌﾞｰｽﾄｻﾊﾞ", "ﾆｷﾞﾜｲ")
			}
			if !sc.GuildCreatedAt.IsZero() {
				gAge := jst.Sub(sc.GuildCreatedAt)
				switch {
				case gAge >= 4*365*24*time.Hour:
					guildCandidates = append(guildCandidates, "ｼﾆｾｻﾊﾞ", "ｲﾆｼｴｻﾊﾞ")
				case gAge >= 0 && gAge < 90*24*time.Hour:
					guildCandidates = append(guildCandidates, "ｼﾝｾﾂｻﾊﾞ", "ﾋﾟｶﾋﾟｶ")
				}
			}
			if sc.GuildIsCommunity {
				guildCandidates = append(guildCandidates, "ﾋﾛﾊﾞ", "ｺﾐｭﾆﾃｨ")
			}
			combined := slices.Concat(guildCandidates, userCandidates)
			if len(combined) > 0 {
				base = pickSlipWord(attrSum, 3, combined)
			}
		}
	}

	if base == "" {
		fallbackSum := g.hmacSum(fmt.Sprintf("slip-fallback:%d", userID))
		base = pickSlipWord(fallbackSum, 0, []string{"ﾜｯﾁｮｲ", "ｻﾊﾞﾌﾗｲ", "ﾄﾞﾝｸﾞﾘ", "ｱｳｱｳ", "ｽｯﾌﾟ"})
	}

	if evLabel, evMode, ok := matchSlipEvent(g.slipEvents, jst); ok {
		if evMode == SlipEventModeOverride {
			return evLabel
		}
		return evLabel + " " + base
	}
	return base
}

// GenerateSlip formats the complete SLIP badge for level (SLIPVVV or SLIPVVVVV).
func (g *IdentityGenerator) GenerateSlip(
	level SLIPLevel,
	boardID uuid.UUID,
	scope Scope,
	guildID snowflake.ID,
	userID snowflake.ID,
	sc SlipContext,
	at time.Time,
) string {
	switch level {
	case SLIPVVV:
		return g.GenerateSlipNickname(boardID, scope, guildID, userID, sc, at)
	case SLIPVVVVV:
		nick := g.GenerateSlipNickname(boardID, scope, guildID, userID, sc, at)
		code := g.GenerateSlipCode(scope, userID, sc)
		return nick + " " + code
	default:
		return ""
	}
}
