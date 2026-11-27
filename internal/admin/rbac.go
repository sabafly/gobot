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

package admin

import (
	"fmt"
	"slices"
	"strings"

	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/bot/components"
)

// RBAC resolves Discord operator identities to configured roles and permission sets.
type RBAC struct {
	roles map[string]map[Permission]struct{}
	users map[snowflake.ID]string
}

// NewRBAC builds an RBAC policy by merging built-in roles with custom role definitions and user bindings.
// It fails closed on unknown permission keys, empty role names, or references to undefined roles.
func NewRBAC(cfg components.ConfigAdmin) (*RBAC, error) {
	validPerms := make(map[Permission]struct{}, len(AllPermissions()))
	for _, p := range AllPermissions() {
		validPerms[p] = struct{}{}
	}

	roles := map[string]map[Permission]struct{}{
		RoleSuperAdmin: toPermSet(AllPermissions()...),
		RoleBBSModerator: toPermSet(
			PermDashboardRead,
			PermLogsRead,
			PermGuildsRead,
			PermAuditRead,
			PermBBSRead,
			PermBBSModerate,
		),
		RoleViewer: toPermSet(
			PermDashboardRead,
			PermLogsRead,
			PermGuildsRead,
			PermAuditRead,
			PermBBSRead,
		),
	}

	for rawName, def := range cfg.Roles {
		roleName := strings.TrimSpace(rawName)
		if roleName == "" {
			return nil, fmt.Errorf("admin rbac: role name must not be empty")
		}
		set := make(map[Permission]struct{}, len(def.Permissions))
		for _, rawPerm := range def.Permissions {
			p := Permission(strings.TrimSpace(rawPerm))
			if _, ok := validPerms[p]; !ok {
				return nil, fmt.Errorf("admin rbac: unknown permission %q in role %q", rawPerm, roleName)
			}
			set[p] = struct{}{}
		}
		roles[roleName] = set
	}

	users := make(map[snowflake.ID]string, len(cfg.Users))
	for _, u := range cfg.Users {
		if u.UserID == 0 {
			return nil, fmt.Errorf("admin rbac: user_id must not be 0")
		}
		roleName := strings.TrimSpace(u.Role)
		if _, ok := roles[roleName]; !ok {
			return nil, fmt.Errorf("admin rbac: user %d references unknown role %q", u.UserID, u.Role)
		}
		users[u.UserID] = roleName
	}

	return &RBAC{
		roles: roles,
		users: users,
	}, nil
}

func toPermSet(perms ...Permission) map[Permission]struct{} {
	set := make(map[Permission]struct{}, len(perms))
	for _, p := range perms {
		set[p] = struct{}{}
	}
	return set
}

// ResolveUser returns the role and deterministically sorted permissions for userID.
func (r *RBAC) ResolveUser(userID snowflake.ID) (string, []Permission, bool) {
	if r == nil || userID == 0 {
		return "", nil, false
	}
	roleName, ok := r.users[userID]
	if !ok {
		return "", nil, false
	}
	permSet, ok := r.roles[roleName]
	if !ok {
		return "", nil, false
	}
	perms := make([]Permission, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	slices.SortFunc(perms, func(a, b Permission) int {
		return strings.Compare(string(a), string(b))
	})
	return roleName, perms, true
}

// HasPermission reports whether perms contains required.
func HasPermission(perms []Permission, required Permission) bool {
	if required == "" {
		return true
	}
	return slices.Contains(perms, required)
}
