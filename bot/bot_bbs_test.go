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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
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

	// Config YAML decoding & wiring of bbs.caps + fail-closed on invalid cap config.
	opUser := snowflake.ID(111111111111111111)
	capUser := snowflake.ID(222222222222222222)
	otherUser := snowflake.ID(333333333333333333)
	cfgPath := filepath.Join(t.TempDir(), "gobot.yml")
	yamlBody := `bbs:
  secret_key: yaml-test-secret
  operators:
    - 111111111111111111
  caps:
    - id: admin
      name: 管理人
      description: 運営公式用
      grant_operators: true
      users:
        - 222222222222222222
    - id: notice
      name: お知らせ
      grant_operators: false
      users:
        - 222222222222222222
`
	if err := os.WriteFile(cfgPath, []byte(yamlBody), 0o600); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	loadedCfg, err := components.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	sysWithCaps, err := initBBSSystem(compHealthy, loadedCfg, gdb)
	if err != nil || sysWithCaps == nil {
		t.Fatalf("initBBSSystem with caps failed: sys=%v err=%v", sysWithCaps, err)
	}
	if opCaps := sysWithCaps.Service.ListAvailableCaps(ctx, opUser, uuid.Nil); len(opCaps) != 1 || opCaps[0].ID != "admin" {
		t.Fatalf("expected operator to hold [admin], got %+v", opCaps)
	}
	if userCaps := sysWithCaps.Service.ListAvailableCaps(ctx, capUser, uuid.Nil); len(userCaps) != 2 || userCaps[0].ID != "admin" || userCaps[1].ID != "notice" {
		t.Fatalf("expected capUser to hold [admin, notice], got %+v", userCaps)
	}
	if noneCaps := sysWithCaps.Service.ListAvailableCaps(ctx, otherUser, uuid.Nil); len(noneCaps) != 0 {
		t.Fatalf("expected non-holder to have no caps, got %+v", noneCaps)
	}

	invalidCapsCfg := &components.Config{BBS: components.ConfigBBS{
		Caps: []components.ConfigBBSCap{
			{ID: bbs.CapSelectNoneValue, Name: "Reserved"},
		},
	}}
	if sysBadCap, errBadCap := initBBSSystem(compHealthy, invalidCapsCfg, gdb); errBadCap == nil || sysBadCap != nil {
		t.Fatalf("expected initBBSSystem to reject reserved cap id %q, got sys=%v err=%v", bbs.CapSelectNoneValue, sysBadCap, errBadCap)
	}
}
