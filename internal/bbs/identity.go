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
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

const idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz+/"

// JSTLocation is Asia/Tokyo timezone used for 5ch-style daily ID rotation and timestamps.
var JSTLocation = time.FixedZone("JST", 9*60*60)

// IdentityGenerator derives HMAC-SHA256 anonymous IDs, tripcodes, SLIPs, and moderation hashes.
type IdentityGenerator struct {
	secret []byte
	loc    *time.Location
}

// NewIdentityGenerator creates an IdentityGenerator using the provided HMAC secret.
func NewIdentityGenerator(secret []byte) *IdentityGenerator {
	cp := make([]byte, len(secret))
	copy(cp, secret)
	return &IdentityGenerator{
		secret: cp,
		loc:    JSTLocation,
	}
}

// ParsedIdentity holds resolved identity attributes and command effects for a new post or thread.
type ParsedIdentity struct {
	DisplayName string
	Trip        string
	SlipText    string
	AnonID      string
	PosterHash  string
	IsSage      bool
	ThreadRules BoardRules
	ExtDat      string
	CleanBody   string
}

// SanitizeSingleLine trims whitespace and replaces newlines/control characters in single-line inputs.
func SanitizeSingleLine(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// SanitizeBody trims leading/trailing whitespace and removes control characters except newline/tab.
func SanitizeBody(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// GenerateAnonID returns an 8-character daily per-board anonymous ID.
func (g *IdentityGenerator) GenerateAnonID(boardID uuid.UUID, userID snowflake.ID, at time.Time) string {
	if at.IsZero() {
		at = time.Now()
	}
	dateStr := at.In(g.loc).Format("2006/01/02")
	sum := g.hmacSum(fmt.Sprintf("anon:%s:%s:%d", boardID, dateStr, userID))
	var b strings.Builder
	b.Grow(8)
	for i := range 8 {
		b.WriteByte(idAlphabet[int(sum[i])%len(idAlphabet)])
	}
	return b.String()
}

// GenerateTrip derives a 10-character tripcode string prefixed by '◆' from the user's raw key.
func (g *IdentityGenerator) GenerateTrip(rawKey string) string {
	sum := sha256.Sum256([]byte("trip:" + rawKey))
	var b strings.Builder
	b.Grow(13)
	b.WriteString("◆")
	for i := range 10 {
		b.WriteByte(idAlphabet[int(sum[i])%len(idAlphabet)])
	}
	return b.String()
}

// GenerateSlip derives an abcd-efgh SLIP identifier (Watchoi) from the user's Discord ID.
func (g *IdentityGenerator) GenerateSlip(userID snowflake.ID) string {
	sum := g.hmacSum(fmt.Sprintf("slip:%d", userID))
	var b strings.Builder
	b.Grow(9)
	for i := range 8 {
		if i == 4 {
			b.WriteByte('-')
		}
		b.WriteByte(idAlphabet[int(sum[i])%len(idAlphabet)])
	}
	return b.String()
}

// GeneratePosterHash generates an opaque 16-hex-character moderation identifier.
// Local boards scope the hash per (guildID, userID) while global boards scope per userID.
func (g *IdentityGenerator) GeneratePosterHash(scope Scope, guildID snowflake.ID, userID snowflake.ID) string {
	var input string
	if scope == ScopeGlobal {
		input = fmt.Sprintf("globalmod:%d", userID)
	} else {
		input = fmt.Sprintf("localmod:%d:%d", guildID, userID)
	}
	sum := g.hmacSum(input)
	return fmt.Sprintf("%x", sum[:8])
}

// ParsePostIdentity resolves nickname commands (!id:on, !slip:vvvvv, sage, #trip),
// mail field ("sage" only), thread creation !extend commands, and anonymous identifiers.
func (g *IdentityGenerator) ParsePostIdentity(
	boardID uuid.UUID,
	scope Scope,
	guildID snowflake.ID,
	userID snowflake.ID,
	rawName string,
	rawMail string,
	rawBody string,
	baseRules BoardRules,
	isNewThread bool,
	at time.Time,
) (ParsedIdentity, error) {
	rules := baseRules.Normalize()
	if at.IsZero() {
		at = time.Now()
	}

	var isSage bool
	mailInput := SanitizeSingleLine(rawMail)
	switch {
	case mailInput == "":
	case strings.EqualFold(mailInput, "sage"):
		isSage = true
	default:
		return ParsedIdentity{}, ErrInvalidMail
	}

	nameInput := SanitizeSingleLine(rawName)
	var trip string
	if before, after, hasTrip := strings.Cut(nameInput, "#"); hasTrip {
		nameInput = strings.TrimSpace(before)
		if after != "" {
			trip = g.GenerateTrip(after)
		}
	}

	// Parse special commands from the name field; reject unrecognized '!' commands.
	var remainingTokens []string
	hideID := rules.HideID
	slipLevel := rules.SLIP
	fields := strings.Fields(nameInput)

	for _, tok := range fields {
		lower := strings.ToLower(tok)
		switch lower {
		case "!id:on":
			hideID = false
		case "!slip:vvvvv":
			slipLevel = SLIPVVVVV
		case "!sage":
			isSage = true
		case "sage":
			if len(fields) == 1 && trip == "" {
				isSage = true
			} else {
				remainingTokens = append(remainingTokens, tok)
			}
		default:
			if strings.HasPrefix(lower, "!") {
				return ParsedIdentity{}, ErrInvalidNameCommand
			}
			remainingTokens = append(remainingTokens, tok)
		}
	}

	displayName := strings.Join(remainingTokens, " ")
	if displayName == "" {
		displayName = rules.DefaultName
	}

	bodyNorm := SanitizeBody(rawBody)
	var extDat string
	lowerBody := strings.ToLower(bodyNorm)
	if strings.HasPrefix(lowerBody, "!extend") {
		isCmdPrefix := len(lowerBody) == len("!extend") ||
			lowerBody[len("!extend")] == ':' ||
			lowerBody[len("!extend")] == ' ' ||
			lowerBody[len("!extend")] == '\t' ||
			lowerBody[len("!extend")] == '\n'
		if isCmdPrefix {
			if !isNewThread || !strings.HasPrefix(lowerBody, "!extend:") {
				return ParsedIdentity{}, ErrInvalidExtendCommand
			}
			firstLine, rest, found := strings.Cut(bodyNorm, "\n")
			if !found {
				return ParsedIdentity{}, ErrInvalidExtendCommand
			}
			trimmedFirst := strings.TrimSpace(firstLine)
			spec := trimmedFirst[len("!extend:"):]
			if spec == "" || strings.ContainsAny(spec, " \t") {
				return ParsedIdentity{}, ErrInvalidExtendCommand
			}
			parts := strings.Split(spec, ":")
			if len(parts) < 1 || len(parts) > 2 {
				return ParsedIdentity{}, ErrInvalidExtendCommand
			}
			parsedRules := rules
			switch strings.ToLower(parts[0]) {
			case "default":
				parsedRules.HideID = rules.HideID
			case "none":
				parsedRules.HideID = true
			case "checked":
				parsedRules.HideID = false
			default:
				return ParsedIdentity{}, ErrInvalidExtendCommand
			}
			if len(parts) == 2 {
				switch strings.ToLower(parts[1]) {
				case "default":
					parsedRules.SLIP = rules.SLIP
				case "none":
					parsedRules.SLIP = SLIPNone
				case "vvvvv":
					parsedRules.SLIP = SLIPVVVVV
				default:
					return ParsedIdentity{}, ErrInvalidExtendCommand
				}
			}
			rules = parsedRules
			hideID = parsedRules.HideID
			slipLevel = parsedRules.SLIP
			extDat = fmt.Sprintf("EXTDAT:%s:----: EXT was configured", spec)
			bodyNorm = SanitizeBody(rest)
		}
	}

	var anonID string
	if !hideID {
		anonID = g.GenerateAnonID(boardID, userID, at)
	}
	var slipText string
	if slipLevel == SLIPVVVVV && !hideID {
		slipText = g.GenerateSlip(userID)
	}

	return ParsedIdentity{
		DisplayName: displayName,
		Trip:        trip,
		SlipText:    slipText,
		AnonID:      anonID,
		PosterHash:  g.GeneratePosterHash(scope, guildID, userID),
		IsSage:      isSage,
		ThreadRules: rules,
		ExtDat:      extDat,
		CleanBody:   bodyNorm,
	}, nil
}

func (g *IdentityGenerator) hmacSum(msg string) [32]byte {
	mac := hmac.New(sha256.New, g.secret)
	_, _ = mac.Write([]byte(msg))
	var out [32]byte
	copy(out[:], mac.Sum(nil))
	return out
}
