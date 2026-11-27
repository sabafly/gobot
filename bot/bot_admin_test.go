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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/admin"
)

func TestInitAdminServer_ConfigAndFailClosed(t *testing.T) {
	t.Setenv("ADMIN_DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_CLIENT_SECRET", "")

	cfgPath := filepath.Join(t.TempDir(), "gobot.yml")
	yamlBody := `admin:
  enabled: true
  listen_addr: "127.0.0.1:18080"
  session_ttl_hours: 48
  log_buffer_size: 500
  oauth:
    redirect_url: "http://127.0.0.1:18080/api/v1/auth/discord/callback"
  roles:
    custom_mod:
      permissions:
        - dashboard.read
        - bbs.moderate
  users:
    - user_id: 111111111111111111
      role: super_admin
    - user_id: 222222222222222222
      role: custom_mod
`
	if err := os.WriteFile(cfgPath, []byte(yamlBody), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfg, err := components.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !cfg.Admin.Enabled || cfg.Admin.ListenAddr != "127.0.0.1:18080" || cfg.Admin.SessionTTLHours != 48 || cfg.Admin.LogBufferSize != 500 {
		t.Fatalf("unexpected decoded admin config: %+v", cfg.Admin)
	}
	if len(cfg.Admin.Users) != 2 || cfg.Admin.Users[0].Role != "super_admin" {
		t.Fatalf("unexpected decoded admin users: %+v", cfg.Admin.Users)
	}

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := gdb.AutoMigrate(&models.AdminSession{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	db := &database.DB{DB: gdb}
	comp := components.New(context.Background(), *cfg, db)
	client := &bot.Client{ApplicationID: snowflake.ID(999000)}
	hub := admin.NewLogHub(cfg.Admin.LogBufferSize)

	// Missing secret fails closed.
	if srv, err := initAdminServer(comp, cfg, db, client, nil, hub, "vtest", time.Now()); err == nil || srv != nil {
		t.Fatalf("expected fail-closed error when secret missing, got srv=%v err=%v", srv, err)
	}

	// Setting ADMIN_DISCORD_CLIENT_SECRET enables healthy creation.
	t.Setenv("ADMIN_DISCORD_CLIENT_SECRET", "env-secret-123")
	srv, err := initAdminServer(comp, cfg, db, client, nil, hub, "vtest", time.Now())
	if err != nil || srv == nil {
		t.Fatalf("expected initAdminServer success, got srv=%v err=%v", srv, err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthenticated overview, got %d", rec.Code)
	}
}
