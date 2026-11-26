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
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

//go:embed legal/*.md
var legalFS embed.FS

type PolicyDocKind string

const (
	PolicyDocTerms      PolicyDocKind = "terms"
	PolicyDocGuidelines PolicyDocKind = "guidelines"
	PolicyDocPrivacy    PolicyDocKind = "privacy"
)

// PolicyConfig configures policy versioning and optional external document links.
type PolicyConfig struct {
	Version          string
	TermsURL         string
	GuidelinesURL    string
	PrivacyPolicyURL string
}

// ValidatePolicyConfig ensures configured policy metadata satisfies persistence length limits.
func ValidatePolicyConfig(cfg PolicyConfig) error {
	ver := strings.TrimSpace(cfg.Version)
	if len(ver) > MaxPolicyVersionLen || utf8.RuneCountInString(ver) > MaxPolicyVersionLen {
		return ErrInvalidPolicyVersion
	}
	return nil
}

// PolicyManager serves embedded policy documents and checks consent version compliance.
type PolicyManager struct {
	CurrentVersion   string
	TermsURL         string
	GuidelinesURL    string
	PrivacyPolicyURL string
}

// NewPolicyManager initializes a PolicyManager with fallback version defaults.
func NewPolicyManager(cfg PolicyConfig) *PolicyManager {
	ver := strings.TrimSpace(cfg.Version)
	if ver == "" {
		ver = DefaultPolicyVersion
	}
	return &PolicyManager{
		CurrentVersion:   ver,
		TermsURL:         strings.TrimSpace(cfg.TermsURL),
		GuidelinesURL:    strings.TrimSpace(cfg.GuidelinesURL),
		PrivacyPolicyURL: strings.TrimSpace(cfg.PrivacyPolicyURL),
	}
}

// NeedsConsent returns true if the user has no consent record or agreed to an older policy version.
func (m *PolicyManager) NeedsConsent(consent *UserConsent) bool {
	if consent == nil {
		return true
	}
	return consent.PolicyVersion != m.CurrentVersion
}

// GetDocument returns the embedded markdown text for the requested document and language.
func (m *PolicyManager) GetDocument(kind PolicyDocKind, locale string) string {
	lang := normalizePolicyLocale(locale)
	filename := "legal/" + string(kind) + "_" + lang + ".md"
	data, err := legalFS.ReadFile(filename)
	if err != nil {
		data, err = legalFS.ReadFile("legal/" + string(kind) + "_ja.md")
		if err != nil {
			return ""
		}
	}
	return string(data)
}

func normalizePolicyLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(l, "en") {
		return "en"
	}
	return "ja"
}

// PolicyVersionToken encodes a policy version string into a route-safe token for component CustomIDs.
func PolicyVersionToken(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		v = DefaultPolicyVersion
	}
	if len(v) <= 32 {
		safe := true
		for i := 0; i < len(v); i++ {
			b := v[i]
			if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '.' && b != '_' && b != '-' {
				safe = false
				break
			}
		}
		if safe {
			return v
		}
	}
	sum := sha256.Sum256([]byte(v))
	return "h_" + hex.EncodeToString(sum[:8])
}
