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
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	DefaultTermsURL         = "https://gobot.sabafly.net/bbs/terms"
	DefaultGuidelinesURL    = "https://gobot.sabafly.net/bbs/guidelines"
	DefaultPrivacyPolicyURL = "https://gobot.sabafly.net/bbs/privacy"
)

// PolicyConfig configures policy versioning and external document links.
type PolicyConfig struct {
	Version          string
	TermsURL         string
	GuidelinesURL    string
	PrivacyPolicyURL string
}

func isValidPolicyURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Hostname() != ""
}

// ValidatePolicyConfig ensures configured policy metadata satisfies persistence length limits and URL format.
func ValidatePolicyConfig(cfg PolicyConfig) error {
	ver := strings.TrimSpace(cfg.Version)
	if len(ver) > MaxPolicyVersionLen || utf8.RuneCountInString(ver) > MaxPolicyVersionLen {
		return ErrInvalidPolicyVersion
	}
	for _, entry := range []struct {
		name string
		val  string
	}{
		{"terms_url", cfg.TermsURL},
		{"guidelines_url", cfg.GuidelinesURL},
		{"privacy_policy_url", cfg.PrivacyPolicyURL},
	} {
		if strings.TrimSpace(entry.val) != "" && !isValidPolicyURL(entry.val) {
			return fmt.Errorf("%w: invalid %s", ErrInvalidInput, entry.name)
		}
	}
	return nil
}

// PolicyManager manages policy version compliance and external legal document URLs.
type PolicyManager struct {
	CurrentVersion   string
	TermsURL         string
	GuidelinesURL    string
	PrivacyPolicyURL string
}

// NewPolicyManager initializes a PolicyManager with fallback version and URL defaults.
func NewPolicyManager(cfg PolicyConfig) *PolicyManager {
	ver := strings.TrimSpace(cfg.Version)
	if ver == "" {
		ver = DefaultPolicyVersion
	}
	termsURL := strings.TrimSpace(cfg.TermsURL)
	if termsURL == "" {
		termsURL = DefaultTermsURL
	}
	guidelinesURL := strings.TrimSpace(cfg.GuidelinesURL)
	if guidelinesURL == "" {
		guidelinesURL = DefaultGuidelinesURL
	}
	privacyURL := strings.TrimSpace(cfg.PrivacyPolicyURL)
	if privacyURL == "" {
		privacyURL = DefaultPrivacyPolicyURL
	}
	return &PolicyManager{
		CurrentVersion:   ver,
		TermsURL:         termsURL,
		GuidelinesURL:    guidelinesURL,
		PrivacyPolicyURL: privacyURL,
	}
}

// NeedsConsent returns true if the user has no consent record or agreed to an older policy version.
func (m *PolicyManager) NeedsConsent(consent *UserConsent) bool {
	if consent == nil {
		return true
	}
	return consent.PolicyVersion != m.CurrentVersion
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
	return "v_" + hex.EncodeToString(sum[:8])
}
