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

package commands_test

import (
	"testing"

	"github.com/sabafly/gobot/bot/commands"
	"github.com/sabafly/gobot/bot/commands/bbs"
	"github.com/sabafly/gobot/internal/i18n"
)

func TestCatalogMatchesRuntimeCommands(t *testing.T) {
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	defs := commands.AllDefinitions()
	runtimeCmds := commands.BuildRuntimeCommands(nil, &bbs.System{})

	if len(defs) != len(runtimeCmds) {
		t.Fatalf("catalog definitions count (%d) != runtime commands count (%d)", len(defs), len(runtimeCmds))
	}

	seenNames := make(map[string]struct{}, len(defs))
	for i, def := range defs {
		rc := runtimeCmds[i]
		if def.Name == "" {
			t.Fatalf("definition[%d] has empty Name", i)
		}
		if _, dup := seenNames[def.Name]; dup {
			t.Fatalf("duplicate command definition name %q", def.Name)
		}
		seenNames[def.Name] = struct{}{}

		if def.Name != rc.Name() {
			t.Fatalf("index %d: definition name %q != runtime command name %q", i, def.Name, rc.Name())
		}
		if def.Private != rc.IsPrivate() {
			t.Fatalf("command %q: definition Private=%v != runtime IsPrivate=%v", def.Name, def.Private, rc.IsPrivate())
		}
		if def.Category == "" {
			t.Fatalf("command %q has empty Category", def.Name)
		}
		if def.Commands == nil {
			t.Fatalf("command %q has nil Commands func", def.Name)
		}

		defCreates := def.Commands()
		rcCreates := rc.Create()
		if len(defCreates) != len(rcCreates) {
			t.Fatalf("command %q: definition creates (%d) != runtime creates (%d)", def.Name, len(defCreates), len(rcCreates))
		}
		for j := range defCreates {
			if defCreates[j].CommandName() != rcCreates[j].CommandName() {
				t.Fatalf("command %q create[%d]: %q != %q", def.Name, j, defCreates[j].CommandName(), rcCreates[j].CommandName())
			}
		}
	}

	pub := commands.PublicDefinitions()
	if len(pub) == 0 {
		t.Fatal("expected non-empty PublicDefinitions")
	}
	for _, def := range pub {
		if def.Private {
			t.Fatalf("PublicDefinitions must not include private command %q", def.Name)
		}
	}
}
