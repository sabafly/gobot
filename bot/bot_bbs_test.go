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

package bot

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func TestInitBBSSystem_FailSafeOnSecretDBError(t *testing.T) {
	t.Setenv("BBS_SECRET_KEY", "")
	ctx := context.Background()

	// Closed/unmigrated DB with empty secret key must fail closed (no ephemeral random fallback).
	dbNoSchema, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	comp := components.New(ctx, components.Config{}, &database.DB{DB: dbNoSchema})
	sys, initErr := initBBSSystem(comp, &components.Config{}, &database.DB{DB: dbNoSchema})
	if initErr == nil || sys != nil {
		t.Fatalf("expected initBBSSystem error on missing secret table, got sys=%v err=%v", sys, initErr)
	}

	// Nil DB must fail closed as well.
	if sysNil, errNil := initBBSSystem(comp, &components.Config{}, nil); errNil == nil || sysNil != nil {
		t.Fatalf("expected initBBSSystem error on nil DB, got sys=%v err=%v", sysNil, errNil)
	}

	// Overlong PolicyVersion must fail closed in initBBSSystem so Bot startup isolates BBS cleanly.
	dbHealthy, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open healthy sqlite failed: %v", err)
	}
	if err := dbHealthy.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	gdb := &database.DB{DB: dbHealthy}
	compHealthy := components.New(ctx, components.Config{}, gdb)
	tooLongCfg := &components.Config{BBS: components.ConfigBBS{PolicyVersion: strings.Repeat("v", bbs.MaxPolicyVersionLen+1)}}
	if sysBad, errBad := initBBSSystem(compHealthy, tooLongCfg, gdb); errBad == nil || sysBad != nil {
		t.Fatalf("expected initBBSSystem to reject >%d PolicyVersion, got sys=%v err=%v", bbs.MaxPolicyVersionLen, sysBad, errBad)
	}

	// Migrated DB with valid PolicyVersion succeeds and returns a usable System.
	validCfg := &components.Config{BBS: components.ConfigBBS{PolicyVersion: strings.Repeat("v", bbs.MaxPolicyVersionLen)}}
	sysOK, errOK := initBBSSystem(compHealthy, validCfg, gdb)
	if errOK != nil || sysOK == nil {
		t.Fatalf("expected healthy initBBSSystem to succeed, got sys=%v err=%v", sysOK, errOK)
	}
}
