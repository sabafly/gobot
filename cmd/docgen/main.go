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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/sabafly/gobot/bot/commands"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/internal/i18n"
)

type docLang struct {
	Code   string
	Locale discord.Locale
}

var targetLangs = []docLang{
	{Code: "ja", Locale: discord.LocaleJapanese},
	{Code: "en", Locale: discord.LocaleEnglishUS},
}

var categoryOrder = []components.CommandCategory{
	components.CommandCategoryGeneral,
	components.CommandCategoryModeration,
	components.CommandCategoryEconomy,
	components.CommandCategoryGames,
	components.CommandCategoryBBS,
}

func categoryLabel(cat components.CommandCategory, lang string) string {
	switch cat {
	case components.CommandCategoryGeneral:
		if lang == "ja" {
			return "一般・ユーティリティ"
		}
		return "General & Utility"
	case components.CommandCategoryModeration:
		if lang == "ja" {
			return "サーバー管理・設定"
		}
		return "Server Management & Settings"
	case components.CommandCategoryEconomy:
		if lang == "ja" {
			return "通貨・経済"
		}
		return "Economy & Currency"
	case components.CommandCategoryGames:
		if lang == "ja" {
			return "ゲーム"
		}
		return "Games"
	case components.CommandCategoryBBS:
		if lang == "ja" {
			return "匿名掲示板 (BBS)"
		}
		return "Anonymous BBS"
	default:
		return string(cat)
	}
}

func main() {
	outDir := flag.String("out", "website/content/docs/commands", "output directory for generated Hugo command markdown files")
	localesDir := flag.String("locales", "locales", "directory containing gobot locale YAML files")
	flag.Parse()

	if err := generate(*localesDir, *outDir, commands.PublicDefinitions()); err != nil {
		fmt.Fprintf(os.Stderr, "docgen failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated command reference markdown into %s\n", *outDir)
}

func generate(localesDir, outDir string, defs []components.CommandDefinition) error {
	if err := i18n.LoadLocales(localesDir); err != nil {
		return fmt.Errorf("load locales: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// Clean previously generated markdown files to avoid stale pages.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", outDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			if err := os.Remove(filepath.Join(outDir, entry.Name())); err != nil {
				return fmt.Errorf("remove stale file %s: %w", entry.Name(), err)
			}
		}
	}

	sortedDefs := slices.Clone(defs)
	slices.SortFunc(sortedDefs, func(a, b components.CommandDefinition) int {
		ca := categoryRank(a.Category)
		cb := categoryRank(b.Category)
		if ca != cb {
			return ca - cb
		}
		return strings.Compare(a.Name, b.Name)
	})

	for _, lang := range targetLangs {
		indexContent := renderIndex(sortedDefs, lang)
		if err := os.WriteFile(filepath.Join(outDir, fmt.Sprintf("_index.%s.md", lang.Code)), []byte(indexContent), 0o644); err != nil {
			return err
		}
		for i, def := range sortedDefs {
			pageContent := renderCommandPage(def, lang, (i+1)*10)
			filename := fmt.Sprintf("%s.%s.md", sanitizeSlug(def.Name), lang.Code)
			if err := os.WriteFile(filepath.Join(outDir, filename), []byte(pageContent), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func categoryRank(cat components.CommandCategory) int {
	for i, c := range categoryOrder {
		if c == cat {
			return i
		}
	}
	return len(categoryOrder)
}

func sanitizeSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

func renderIndex(defs []components.CommandDefinition, lang docLang) string {
	var b strings.Builder
	if lang.Code == "ja" {
		b.WriteString("---\ntitle: コマンド一覧\nweight: 20\n---\n\n")
		b.WriteString("gobot で利用できる公開コマンドのリファレンスです。ソースコードのコマンド定義から自動生成されています。\n\n")
	} else {
		b.WriteString("---\ntitle: Commands\nweight: 20\n---\n\n")
		b.WriteString("Reference for all public commands available in gobot, automatically generated from source definitions.\n\n")
	}

	byCat := make(map[components.CommandCategory][]components.CommandDefinition)
	for _, def := range defs {
		byCat[def.Category] = append(byCat[def.Category], def)
	}

	for _, cat := range categoryOrder {
		items := byCat[cat]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", categoryLabel(cat, lang.Code))
		if lang.Code == "ja" {
			b.WriteString("| コマンド | 概要 |\n| --- | --- |\n")
		} else {
			b.WriteString("| Command | Description |\n| --- | --- |\n")
		}
		for _, def := range items {
			creates := def.Commands()
			summary := primarySummary(creates, lang.Locale)
			slug := sanitizeSlug(def.Name)
			link := fmt.Sprintf("./%s/", slug)
			displayName := displayCommandTitle(def, creates)
			fmt.Fprintf(&b, "| [%s](%s) | %s |\n", escapeTable(displayName), link, escapeTable(summary))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderCommandPage(def components.CommandDefinition, lang docLang, weight int) string {
	creates := def.Commands()
	title := displayCommandTitle(def, creates)
	summary := primarySummary(creates, lang.Locale)

	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %q\nweight: %d\ndescription: %q\n---\n\n", title, weight, summary)
	if summary != "" {
		b.WriteString(summary + "\n\n")
	}
	if lang.Code == "ja" {
		fmt.Fprintf(&b, "- **カテゴリ**: %s\n", categoryLabel(def.Category, lang.Code))
	} else {
		fmt.Fprintf(&b, "- **Category**: %s\n", categoryLabel(def.Category, lang.Code))
	}
	b.WriteString("\n")

	for _, cmd := range creates {
		switch c := cmd.(type) {
		case discord.SlashCommandCreate:
			renderSlashCommand(&b, c, def.Permissions, lang)
		case discord.MessageCommandCreate:
			if lang.Code == "ja" {
				fmt.Fprintf(&b, "## メッセージコマンド: %s\n\nメッセージを右クリック →「アプリ」→「**%s**」から実行できます。\n\n", c.Name, localizeName(c.Name, c.NameLocalizations, lang.Locale))
			} else {
				fmt.Fprintf(&b, "## Message Command: %s\n\nRight-click a message → **Apps** → **%s**.\n\n", c.Name, localizeName(c.Name, c.NameLocalizations, lang.Locale))
			}
		case discord.UserCommandCreate:
			if lang.Code == "ja" {
				fmt.Fprintf(&b, "## ユーザーコマンド: %s\n\nユーザーを右クリック →「アプリ」→「**%s**」から実行できます。\n\n", c.Name, localizeName(c.Name, c.NameLocalizations, lang.Locale))
			} else {
				fmt.Fprintf(&b, "## User Command: %s\n\nRight-click a user → **Apps** → **%s**.\n\n", c.Name, localizeName(c.Name, c.NameLocalizations, lang.Locale))
			}
		}
	}
	return b.String()
}

func renderSlashCommand(b *strings.Builder, cmd discord.SlashCommandCreate, perms map[string]components.CommandPermissionMeta, lang docLang) {
	desc := localizeDesc(cmd.Description, cmd.DescriptionLocalizations, lang.Locale)
	fmt.Fprintf(b, "## `/%s`\n\n", cmd.Name)
	if desc != "" {
		b.WriteString(desc + "\n\n")
	}

	var defaultPerms discord.Permissions
	if cmd.DefaultMemberPermissions.OK && cmd.DefaultMemberPermissions.Value != nil {
		defaultPerms = *cmd.DefaultMemberPermissions.Value
	}

	var subCommands []discord.ApplicationCommandOptionSubCommand
	var subGroups []discord.ApplicationCommandOptionSubCommandGroup
	var directOpts []discord.ApplicationCommandOption

	for _, opt := range cmd.Options {
		switch o := opt.(type) {
		case discord.ApplicationCommandOptionSubCommand:
			subCommands = append(subCommands, o)
		case discord.ApplicationCommandOptionSubCommandGroup:
			subGroups = append(subGroups, o)
		default:
			directOpts = append(directOpts, opt)
		}
	}

	if len(subCommands) == 0 && len(subGroups) == 0 {
		route := "/" + cmd.Name
		renderPermissionsLine(b, defaultPerms, perms[route], lang)
		if len(directOpts) > 0 {
			renderOptionsTable(b, directOpts, lang)
		}
		return
	}

	for _, sc := range subCommands {
		route := fmt.Sprintf("/%s/%s", cmd.Name, sc.Name)
		fmt.Fprintf(b, "### `/%s %s`\n\n", cmd.Name, sc.Name)
		if d := localizeDesc(sc.Description, sc.DescriptionLocalizations, lang.Locale); d != "" {
			b.WriteString(d + "\n\n")
		}
		renderPermissionsLine(b, defaultPerms, perms[route], lang)
		if len(sc.Options) > 0 {
			renderOptionsTable(b, sc.Options, lang)
		}
	}

	for _, grp := range subGroups {
		grpDesc := localizeDesc(grp.Description, grp.DescriptionLocalizations, lang.Locale)
		fmt.Fprintf(b, "### `/%s %s`\n\n", cmd.Name, grp.Name)
		if grpDesc != "" {
			b.WriteString(grpDesc + "\n\n")
		}
		for _, sc := range grp.Options {
			route := fmt.Sprintf("/%s/%s/%s", cmd.Name, grp.Name, sc.Name)
			fmt.Fprintf(b, "#### `/%s %s %s`\n\n", cmd.Name, grp.Name, sc.Name)
			if d := localizeDesc(sc.Description, sc.DescriptionLocalizations, lang.Locale); d != "" {
				b.WriteString(d + "\n\n")
			}
			renderPermissionsLine(b, defaultPerms, perms[route], lang)
			if len(sc.Options) > 0 {
				renderOptionsTable(b, sc.Options, lang)
			}
		}
	}
}

func renderPermissionsLine(b *strings.Builder, cmdDefault discord.Permissions, meta components.CommandPermissionMeta, lang docLang) {
	discordPerm := cmdDefault | meta.DiscordPermission
	var parts []string
	if discordPerm != 0 {
		parts = append(parts, formatDiscordPermissions(discordPerm)...)
	}
	for _, node := range meta.Permissions {
		parts = append(parts, fmt.Sprintf("`%s`", node))
	}
	if len(parts) == 0 {
		return
	}
	if lang.Code == "ja" {
		fmt.Fprintf(b, "**必要な権限**: %s\n\n", strings.Join(parts, ", "))
	} else {
		fmt.Fprintf(b, "**Required Permissions**: %s\n\n", strings.Join(parts, ", "))
	}
}

func renderOptionsTable(b *strings.Builder, opts []discord.ApplicationCommandOption, lang docLang) {
	if lang.Code == "ja" {
		b.WriteString("| オプション | 型 | 必須 | 説明 |\n| --- | --- | --- | --- |\n")
	} else {
		b.WriteString("| Option | Type | Required | Description |\n| --- | --- | --- | --- |\n")
	}
	for _, opt := range opts {
		name, typ, req, desc, choices := extractOptionInfo(opt, lang.Locale)
		reqStr := "No"
		if lang.Code == "ja" {
			reqStr = "任意"
			if req {
				reqStr = "必須"
			}
		} else if req {
			reqStr = "Yes"
		}
		if len(choices) > 0 {
			if lang.Code == "ja" {
				desc += "（選択肢: " + strings.Join(choices, ", ") + "）"
			} else {
				desc += " (Choices: " + strings.Join(choices, ", ") + ")"
			}
		}
		fmt.Fprintf(b, "| `%s` | %s | %s | %s |\n", escapeTable(name), typ, reqStr, escapeTable(desc))
	}
	b.WriteString("\n")
}

func extractOptionInfo(opt discord.ApplicationCommandOption, loc discord.Locale) (name, typ string, required bool, desc string, choices []string) {
	switch o := opt.(type) {
	case discord.ApplicationCommandOptionString:
		for _, ch := range o.Choices {
			choices = append(choices, fmt.Sprintf("`%s`", ch.Value))
		}
		return o.Name, "String", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), choices
	case discord.ApplicationCommandOptionInt:
		for _, ch := range o.Choices {
			choices = append(choices, fmt.Sprintf("`%d`", ch.Value))
		}
		return o.Name, "Integer", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), choices
	case discord.ApplicationCommandOptionFloat:
		for _, ch := range o.Choices {
			choices = append(choices, fmt.Sprintf("`%g`", ch.Value))
		}
		return o.Name, "Number", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), choices
	case discord.ApplicationCommandOptionBool:
		return o.Name, "Boolean", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	case discord.ApplicationCommandOptionUser:
		return o.Name, "User", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	case discord.ApplicationCommandOptionChannel:
		return o.Name, "Channel", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	case discord.ApplicationCommandOptionRole:
		return o.Name, "Role", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	case discord.ApplicationCommandOptionMentionable:
		return o.Name, "Mentionable", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	case discord.ApplicationCommandOptionAttachment:
		return o.Name, "Attachment", o.Required, localizeDesc(o.Description, o.DescriptionLocalizations, loc), nil
	default:
		return opt.OptionName(), fmt.Sprintf("Type(%d)", opt.Type()), false, "", nil
	}
}

func formatDiscordPermissions(p discord.Permissions) []string {
	known := []struct {
		flag discord.Permissions
		name string
	}{
		{discord.PermissionAdministrator, "Administrator"},
		{discord.PermissionManageGuild, "Manage Server"},
		{discord.PermissionManageRoles, "Manage Roles"},
		{discord.PermissionManageChannels, "Manage Channels"},
		{discord.PermissionManageMessages, "Manage Messages"},
		{discord.PermissionManageWebhooks, "Manage Webhooks"},
		{discord.PermissionModerateMembers, "Moderate Members"},
		{discord.PermissionBanMembers, "Ban Members"},
		{discord.PermissionKickMembers, "Kick Members"},
	}
	var out []string
	for _, k := range known {
		if p.Has(k.flag) {
			out = append(out, fmt.Sprintf("Discord `%s`", k.name))
		}
	}
	return out
}

func displayCommandTitle(def components.CommandDefinition, creates []discord.ApplicationCommandCreate) string {
	for _, c := range creates {
		if sc, ok := c.(discord.SlashCommandCreate); ok {
			return "/" + sc.Name
		}
	}
	return def.Name
}

func primarySummary(creates []discord.ApplicationCommandCreate, loc discord.Locale) string {
	for _, c := range creates {
		if sc, ok := c.(discord.SlashCommandCreate); ok {
			return localizeDesc(sc.Description, sc.DescriptionLocalizations, loc)
		}
	}
	return ""
}

func localizeDesc(fallback string, locs map[discord.Locale]string, target discord.Locale) string {
	if locs != nil {
		if v := strings.TrimSpace(locs[target]); v != "" {
			return v
		}
	}
	return strings.TrimSpace(fallback)
}

func localizeName(fallback string, locs map[discord.Locale]string, target discord.Locale) string {
	if locs != nil {
		if v := strings.TrimSpace(locs[target]); v != "" {
			return v
		}
	}
	return strings.TrimSpace(fallback)
}

func escapeTable(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}
