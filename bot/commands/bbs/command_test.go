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

package bbs_test

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	bbsCmd "github.com/sabafly/gobot/bot/commands/bbs"
	"github.com/sabafly/gobot/bot/components"
	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/i18n"
)

func hasPictographEmoji(s string) bool {
	for _, r := range s {
		if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
			return true
		}
	}
	return false
}

func assertNoEmojiInOptions(t *testing.T, opts []discord.ApplicationCommandOption) {
	t.Helper()
	for _, o := range opts {
		if hasPictographEmoji(o.OptionName()) || hasPictographEmoji(o.OptionDescription()) {
			t.Fatalf("unexpected pictograph emoji in option %q / %q", o.OptionName(), o.OptionDescription())
		}
		switch typed := o.(type) {
		case discord.ApplicationCommandOptionSubCommand:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in subcommand localization %q", v)
				}
			}
			assertNoEmojiInOptions(t, typed.Options)
		case discord.ApplicationCommandOptionSubCommandGroup:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in group localization %q", v)
				}
			}
			subOpts := make([]discord.ApplicationCommandOption, len(typed.Options))
			for i, so := range typed.Options {
				subOpts[i] = so
			}
			assertNoEmojiInOptions(t, subOpts)
		case discord.ApplicationCommandOptionString:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in string option localization %q", v)
				}
			}
		case discord.ApplicationCommandOptionBool:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in bool option localization %q", v)
				}
			}
		case discord.ApplicationCommandOptionInt:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in int option localization %q", v)
				}
			}
		case discord.ApplicationCommandOptionChannel:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in channel option localization %q", v)
				}
			}
		case discord.ApplicationCommandOptionRole:
			for _, v := range typed.DescriptionLocalizations {
				if hasPictographEmoji(v) {
					t.Fatalf("unexpected pictograph emoji in role option localization %q", v)
				}
			}
		}
	}
}

func newTestSystem(t *testing.T, operators ...snowflake.ID) *bbsCmd.System {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	store := gormstore.New(db)
	reg := bbscore.NewProviderRegistry(bbscore.DefaultProviderKey)
	reg.Register(store)
	idGen := bbscore.NewIdentityGenerator([]byte("test-secret-key-32bytes-minimum!!"))
	pm := bbscore.NewPolicyManager(bbscore.PolicyConfig{})
	svc := bbscore.NewService(reg, store, idGen, pm, operators)
	syncer := discordsync.NewSyncer(svc)
	return bbsCmd.NewSystem(&components.Components{}, svc, syncer, nil)
}

func TestBBSCommands_ValidateAndZeroEmojiAndRouting(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	opID := snowflake.ID(777001)
	sys := newTestSystem(t, opID)

	cmd := bbsCmd.Command(sys)
	adminCmd := bbsCmd.AdminCommand(sys)
	if cmd.IsPrivate() {
		t.Fatal("/bbs must not be private")
	}
	if !adminCmd.IsPrivate() {
		t.Fatal("/bbs-admin must be private")
	}

	allCreates := append(cmd.Create(), adminCmd.Create()...)
	var v discordsync.PayloadValidator
	if err := v.ValidateApplicationCommands(allCreates); err != nil {
		t.Fatalf("ValidateApplicationCommands failed: %v", err)
	}

	for _, c := range allCreates {
		switch typed := c.(type) {
		case discord.SlashCommandCreate:
			if hasPictographEmoji(typed.Name) || hasPictographEmoji(typed.Description) {
				t.Fatalf("unexpected pictograph emoji in command %q", typed.Name)
			}
			for _, l := range typed.DescriptionLocalizations {
				if hasPictographEmoji(l) {
					t.Fatalf("unexpected pictograph emoji in localization %q", l)
				}
			}
			assertNoEmojiInOptions(t, typed.Options)
		case discord.MessageCommandCreate:
			if hasPictographEmoji(typed.Name) {
				t.Fatalf("unexpected pictograph emoji in message command %q", typed.Name)
			}
			for _, l := range typed.NameLocalizations {
				if hasPictographEmoji(l) {
					t.Fatalf("unexpected pictograph emoji in message command localization %q", l)
				}
			}
		}
	}

	r := handler.New()
	cmd.RegisterRoutes(r)
	adminCmd.RegisterRoutes(r)

	tid := uuid.New().String()
	// Static /bbs/reader/jump/{thread_id} must match specifically even before /bbs/reader/{thread_id}/{page}
	if !r.Match("/bbs/reader/jump/"+tid, discord.InteractionTypeComponent, int(discord.ComponentTypeButton)) {
		t.Fatal("expected /bbs/reader/jump/{thread_id} route to match")
	}
	if !r.Match("/bbs/reader/"+tid+"/1", discord.InteractionTypeComponent, int(discord.ComponentTypeButton)) {
		t.Fatal("expected /bbs/reader/{thread_id}/{page} route to match")
	}
	if !r.Match("/bbs/consent/revoke", discord.InteractionTypeComponent, int(discord.ComponentTypeButton)) {
		t.Fatal("expected /bbs/consent/revoke route to match")
	}
	if !r.Match("/BBS - レスを通報", discord.InteractionTypeApplicationCommand, int(discord.ApplicationCommandTypeMessage)) {
		t.Fatal("expected report message command route to match")
	}
	if !r.Match("/bbs-admin/report/inspect", discord.InteractionTypeApplicationCommand, int(discord.ApplicationCommandTypeSlash)) {
		t.Fatal("expected /bbs-admin/report/inspect route to match")
	}

	if sys.Service.IsOperator(snowflake.ID(999999)) {
		t.Fatal("non-operator must not satisfy IsOperator")
	}
	if !sys.Service.IsOperator(opID) {
		t.Fatal("configured operator must satisfy IsOperator")
	}
}
