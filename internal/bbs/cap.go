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
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
)

// ValidateCapDefinitions normalizes and validates static cap definitions loaded from config.
func ValidateCapDefinitions(defs []CapDefinition) ([]CapDefinition, error) {
	if len(defs) == 0 {
		return nil, nil
	}
	seenIDs := make(map[string]struct{}, len(defs))
	out := make([]CapDefinition, 0, len(defs))
	for _, def := range defs {
		id := SanitizeSingleLine(def.ID)
		if id == "" || id == CapSelectNoneValue || utf8.RuneCountInString(id) > MaxCapIDRunes {
			return nil, fmt.Errorf("%w: invalid cap id %q", ErrInvalidCapConfig, def.ID)
		}
		if _, dup := seenIDs[id]; dup {
			return nil, fmt.Errorf("%w: duplicate cap id %q", ErrInvalidCapConfig, id)
		}
		seenIDs[id] = struct{}{}

		name := SanitizeDisplayName(def.Name)
		if name == "" || strings.TrimSpace(ClearFormat(name)) == "" || utf8.RuneCountInString(name) > MaxCapNameRunes {
			return nil, fmt.Errorf("%w: invalid cap name for %q", ErrInvalidCapConfig, id)
		}

		desc := SanitizeSingleLine(def.Description)
		if utf8.RuneCountInString(desc) > MaxCapDescriptionRunes {
			return nil, fmt.Errorf("%w: cap description too long for %q", ErrInvalidCapConfig, id)
		}

		var users []snowflake.ID
		for _, uid := range def.Users {
			if uid == 0 || slices.Contains(users, uid) {
				continue
			}
			users = append(users, uid)
		}

		out = append(out, CapDefinition{
			ID:             id,
			Name:           name,
			Description:    desc,
			Users:          users,
			GrantOperators: def.GrantOperators,
		})
	}
	return out, nil
}
