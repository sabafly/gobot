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
 *
 */

// Deprecated: This file is deprecated and will be removed in a future version.
package permissions

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (p *Permission) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &p.m)
}

func (p Permission) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.m)
}

type Permission struct{ m map[string]bool }

func (p Permission) String() string {
	b := strings.Builder{}
	for k, v := range p.m {
		b.WriteString(fmt.Sprintf("%s: %t\n", k, v))
	}
	return b.String()
}

func (p *Permission) Set(perm string, enabled bool) {
	if p.m == nil {
		p.m = make(map[string]bool)
	}
	perm = strings.TrimPrefix(perm, ".")
	perm = strings.TrimSuffix(perm, ".*")
	p.m[perm] = enabled
}

func (p Permission) UnSet(perm string) {
	if p.m == nil {
		p.m = make(map[string]bool)
	}
	perm = strings.TrimPrefix(perm, ".")
	perm = strings.TrimSuffix(perm, ".*")
	delete(p.m, perm)
}

func (p Permission) Enabled(perm string) bool {
	if p.m == nil {
		return false
	}
	perms := strings.Split(perm, ".")
	var pc string
	r := p.m["*"]
	for _, v := range perms {
		pc += v
		if a, ok := p.m[pc]; ok {
			r = a
		}
		pc += "."
	}
	return r
}

func (p Permission) Disabled(perm string) bool {
	if p.m == nil {
		return false
	}
	perms := strings.Split(perm, ".")
	var pc string
	a, ok := p.m["*"]
	r := ok && !a
	for _, v := range perms {
		pc += v
		if a, ok := p.m[pc]; ok {
			r = !a
		}
		pc += "."
	}
	return r
}
