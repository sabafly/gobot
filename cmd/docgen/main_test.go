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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sabafly/gobot/bot/commands"
)

func TestGenerate_PublicCommandsBilingual(t *testing.T) {
	outDir := t.TempDir()
	// Place a stale markdown file to verify cleanup.
	stale := filepath.Join(outDir, "stale.ja.md")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatalf("write stale file failed: %v", err)
	}

	defs := commands.PublicDefinitions()
	if err := generate("../../locales", outDir, defs); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("expected stale markdown file to be removed, err=%v", err)
	}

	for _, lang := range []string{"ja", "en"} {
		idxPath := filepath.Join(outDir, "_index."+lang+".md")
		idxBytes, err := os.ReadFile(idxPath)
		if err != nil {
			t.Fatalf("read %s failed: %v", idxPath, err)
		}
		idxStr := string(idxBytes)
		if strings.Contains(idxStr, "debug") || strings.Contains(idxStr, "bbsadmin") {
			t.Fatalf("private commands must not appear in %s", idxPath)
		}
		for _, def := range defs {
			pagePath := filepath.Join(outDir, sanitizeSlug(def.Name)+"."+lang+".md")
			raw, err := os.ReadFile(pagePath)
			if err != nil {
				t.Fatalf("expected generated page %s: %v", pagePath, err)
			}
			if !strings.HasPrefix(string(raw), "---\n") {
				t.Fatalf("expected frontmatter in %s", pagePath)
			}
		}
	}

	// Verify subcommand groups (e.g. /bbs board mount) are rendered properly.
	bbsJA, err := os.ReadFile(filepath.Join(outDir, "bbs.ja.md"))
	if err != nil {
		t.Fatalf("read bbs.ja.md failed: %v", err)
	}
	if !strings.Contains(string(bbsJA), "/bbs") {
		t.Fatalf("expected /bbs content in bbs.ja.md")
	}
}
