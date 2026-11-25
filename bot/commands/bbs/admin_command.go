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
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/omit"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

// AdminCommand returns the private operator-only /bbs-admin command.
func AdminCommand(sys *System) components.Command {
	return &adminCommand{sys: sys}
}

type adminCommand struct {
	sys *System
}

func (c *adminCommand) Name() string { return "bbs-admin" }

func (c *adminCommand) IsPrivate() bool { return true }

func (c *adminCommand) Create() []discord.ApplicationCommandCreate {
	slipChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "none (非表示)", Value: "none"},
		{Name: "vvvvv (端末種別+OS)", Value: "vvvvv"},
	}
	statusChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "open (未処理)", Value: string(bbscore.ReportStatusOpen)},
		{Name: "resolved (対処済)", Value: string(bbscore.ReportStatusResolved)},
		{Name: "dismissed (却下)", Value: string(bbscore.ReportStatusDismissed)},
	}
	scopeChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "global", Value: string(bbscore.ScopeGlobal)},
		{Name: "local", Value: string(bbscore.ScopeLocal)},
	}

	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:                     "bbs-admin",
			Description:              "gobot operators: manage global BBS boards, reports, and bans",
			DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.description"),
			DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
			Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "category",
					Description:              "Manage global categories",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a global category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Category name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(60),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Category description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.create.options.description.description"),
									MaxLength:                intPtr(200),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List global categories",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.list.description"),
						},
						{
							Name:                     "delete",
							Description:              "Delete a global category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Global category to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.category.delete.options.category.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "board",
					Description:              "Manage global boards",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a global board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Board name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(80),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Global category for the board",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.category.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "Default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.slip.description"),
									Choices:                  slipChoices,
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "max_res",
									Description:              "Max replies per thread (50-1000)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.max_res.description"),
									MinValue:                 intPtr(50),
									MaxValue:                 intPtr(1000),
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "archive_days",
									Description:              "Days of inactivity before auto-archive (0 disables)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.create.options.archive_days.description"),
									MinValue:                 intPtr(0),
									MaxValue:                 intPtr(365),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List global boards",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.list.description"),
						},
						{
							Name:                     "edit",
							Description:              "Edit a global board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to edit",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "New default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "New board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.edit.options.slip.description"),
									Choices:                  slipChoices,
								},
							},
						},
						{
							Name:                     "delete",
							Description:              "Delete a global board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.board.delete.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "report",
					Description:              "Inspect and handle BBS reports",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.report.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "list",
							Description:              "List BBS reports",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.report.list.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "status",
									Description:              "Filter by status",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.report.list.options.status.description"),
									Choices:                  statusChoices,
								},
							},
						},
						{
							Name:                     "inspect",
							Description:              "Inspect a report and moderate",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.report.inspect.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "report",
									Description:              "Report to inspect",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.report.inspect.options.report.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "ban",
					Description:              "Manage BBS bans",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.ban.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "list",
							Description:              "List active BBS bans",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.ban.list.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "scope",
									Description:              "Filter by scope",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.ban.list.options.scope.description"),
									Choices:                  scopeChoices,
								},
							},
						},
						{
							Name:                     "unban",
							Description:              "Lift an active BBS ban",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.ban.unban.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "ban",
									Description:              "Ban to lift",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs_admin.ban.unban.options.ban.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *adminCommand) RegisterRoutes(r handler.Router) {
	r.SlashCommand("/bbs-admin/category/create", c.handleGlobalCategoryCreate)
	r.SlashCommand("/bbs-admin/category/list", c.handleGlobalCategoryList)
	r.SlashCommand("/bbs-admin/category/delete", c.handleGlobalCategoryDelete)
	r.SlashCommand("/bbs-admin/board/create", c.handleGlobalBoardCreate)
	r.SlashCommand("/bbs-admin/board/list", c.handleGlobalBoardList)
	r.SlashCommand("/bbs-admin/board/edit", c.handleGlobalBoardEdit)
	r.SlashCommand("/bbs-admin/board/delete", c.handleGlobalBoardDelete)
	r.SlashCommand("/bbs-admin/report/list", c.handleReportList)
	r.SlashCommand("/bbs-admin/report/inspect", c.handleReportInspect)
	r.SlashCommand("/bbs-admin/ban/list", c.handleBanList)
	r.SlashCommand("/bbs-admin/ban/unban", c.handleBanUnban)

	r.Autocomplete("/bbs-admin/category/delete", c.autocompleteGlobalCategories)
	r.Autocomplete("/bbs-admin/board/create", c.autocompleteGlobalCategories)
	r.Autocomplete("/bbs-admin/board/edit", c.autocompleteGlobalBoards)
	r.Autocomplete("/bbs-admin/board/delete", c.autocompleteGlobalBoards)
	r.Autocomplete("/bbs-admin/report/inspect", c.autocompleteReports)
	r.Autocomplete("/bbs-admin/ban/unban", c.autocompleteBans)

	r.Component("/bbs/admin/abone/{report_id}", c.handleAdminAboneButton)
	r.Component("/bbs/admin/ban/{report_id}", c.handleAdminBanButton)
	r.Modal("/bbs/admin/banmodal/{report_id}", c.handleAdminBanModal)
	r.Component("/bbs/admin/dismiss/{report_id}", c.handleAdminDismissButton)
}

func (c *adminCommand) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return nil
}
func (c *adminCommand) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return nil
}
func (c *adminCommand) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return nil
}
func (c *adminCommand) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
	return nil
}
func (c *adminCommand) OnEvent() func(event bot.Event) error { return nil }
func (c *adminCommand) Scheduler() []components.Scheduler    { return nil }
func (c *adminCommand) HasCommand(_ *events.ApplicationCommandInteractionCreate) bool {
	return false
}

func (c *adminCommand) requireOperatorCmd(e *handler.CommandEvent) bool {
	if c.sys.Service.IsOperator(e.User().ID) {
		return true
	}
	c.sys.replyCommandError(e, errTitlePermission, "この操作は gobot 運営メンバーのみ実行できます。")
	return false
}

func (c *adminCommand) handleGlobalCategoryCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	cat, err := c.sys.Service.CreateCategory(e.Ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeGlobal,
		Name:        data.String("name"),
		Description: data.String("description"),
	})
	if err != nil {
		c.sys.replyCommandError(e, "作成失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバルカテゴリを作成しました", fmt.Sprintf("カテゴリ: %s\nID: %s", cat.Name, cat.ID), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalCategoryList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	scope := bbscore.ScopeGlobal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(cats) == 0 {
		c.sys.replyCommandNotice(e, "グローバルカテゴリ一覧", "グローバルカテゴリはまだありません。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, cat := range cats {
		lines = append(lines, fmt.Sprintf("- %s (%s)", cat.Name, cat.ID))
	}
	c.sys.replyCommandNotice(e, "グローバルカテゴリ一覧", strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalCategoryDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "カテゴリIDが不正です。")
		return nil
	}
	cat, err := c.sys.Service.GetCategory(e.Ctx, catID)
	if err != nil || cat.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, "削除失敗", "対象のグローバルカテゴリが見つかりません。")
		return nil
	}
	if err := c.sys.Service.DeleteCategory(e.Ctx, catID); err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバルカテゴリを削除しました", cat.Name, discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleGlobalBoardCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "カテゴリIDが不正です。")
		return nil
	}
	board, err := c.sys.Service.CreateBoard(e.Ctx, bbscore.CreateBoardInput{
		CategoryID:  catID,
		Scope:       bbscore.ScopeGlobal,
		Name:        data.String("name"),
		Description: data.String("description"),
		Rules:       parseBoardRulesFromSlash(data),
	})
	if err != nil {
		c.sys.replyCommandError(e, "グローバル板作成失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバル板を作成しました", fmt.Sprintf("板: %s\nID: %s\n各サーバーから /bbs global subscribe で購読可能になりました。", board.Name, board.ID), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalBoardList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(boards) == 0 {
		c.sys.replyCommandNotice(e, "グローバル板一覧", "グローバル板はまだありません。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, b := range boards {
		lines = append(lines, fmt.Sprintf("- %s (%s) 名無し:%s", b.Name, b.ID, b.Rules.Normalize().DefaultName))
	}
	c.sys.replyCommandNotice(e, "グローバル板一覧", strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalBoardEdit(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "板IDが不正です。")
		return nil
	}
	b, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil || b.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, "編集失敗", "対象のグローバル板が見つかりません。")
		return nil
	}
	updated, err := c.sys.Service.UpdateBoard(e.Ctx, boardID, parseUpdateBoardInput(data))
	if err != nil {
		c.sys.replyCommandError(e, "編集失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバル板を更新しました", fmt.Sprintf("板: %s\n名無し: %s", updated.Name, updated.Rules.Normalize().DefaultName), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalBoardDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "板IDが不正です。")
		return nil
	}
	b, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil || b.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, "削除失敗", "対象のグローバル板が見つかりません。")
		return nil
	}
	if mounts, mErr := c.sys.Service.Store.ListActiveBoardMountsByBoard(e.Ctx, boardID); mErr == nil {
		for _, m := range mounts {
			_ = c.sys.Service.Store.DeactivateBoardMount(e.Ctx, m.ID)
		}
	}
	if err := c.sys.Service.DeleteBoard(e.Ctx, boardID); err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバル板を削除しました", b.Name, discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleReportList(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	filter := bbscore.ReportFilter{Limit: 25}
	if v, ok := data.OptString("status"); ok && v != "" {
		st := bbscore.ReportStatus(v)
		filter.Status = &st
	}
	reports, err := c.sys.Service.Store.ListReports(e.Ctx, filter)
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(reports) == 0 {
		c.sys.replyCommandNotice(e, "通報一覧", "該当する通報はありません。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, r := range reports {
		lines = append(lines, fmt.Sprintf("- [%s] %s | Guild:%d | 理由:%s", r.Status, r.ID, r.GuildID, discordsync.TruncateRunes(r.Reason, 40)))
	}
	c.sys.replyCommandNotice(e, "BBS通報一覧", strings.Join(lines, "\n"), discordsync.ColorError)
	return nil
}

func (c *adminCommand) handleReportInspect(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	reportID, err := uuid.Parse(strings.TrimSpace(data.String("report")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "通報IDが不正です。")
		return nil
	}
	report, audit, err := c.sys.Service.InspectReportForOperator(e.Ctx, e.User().ID, reportID)
	if err != nil {
		c.sys.replyCommandError(e, "監査情報取得失敗", describeBBSError(err))
		return nil
	}

	preview := "あぼーん済み"
	if !audit.Post.Abone {
		preview = discordsync.TruncateRunes(audit.Post.Body, 400)
	}
	detail := fmt.Sprintf(
		"通報ID: %s\nステータス: %s / スコープ: %s\n通報元Guild: %d / 通報者: <@%d> (%d)\n対象PostID: %s (>>%d)\n投稿者(運営監査情報): <@%d> (%d)\nPosterHash: %s / ID:%s\n通報理由: %s\n--- 本文抜粋 ---\n%s",
		report.ID, report.Status, report.Scope,
		report.GuildID, report.ReporterUserID, report.ReporterUserID,
		report.PostID, audit.Post.ResNumber,
		audit.AuthorUserID, audit.AuthorUserID,
		audit.Post.PosterHash, audit.Post.AnonID,
		report.Reason,
		preview,
	)

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ BBS 通報詳細・運営監査パネル"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(discordsync.TruncateRunes(detail, 1800)),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewDangerButton("グローバルあぼーん", fmt.Sprintf("/bbs/admin/abone/%s", report.ID)),
				discord.NewDangerButton("グローバルBAN+あぼーん", fmt.Sprintf("/bbs/admin/ban/%s", report.ID)),
				discord.NewSecondaryButton("却下", fmt.Sprintf("/bbs/admin/dismiss/%s", report.ID)),
			),
		).WithAccentColor(discordsync.ColorError),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (c *adminCommand) handleBanList(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	filter := bbscore.BanFilter{Limit: 25}
	if v, ok := data.OptString("scope"); ok && v != "" {
		sc := bbscore.Scope(v)
		filter.Scope = &sc
	}
	bans, err := c.sys.Service.Store.ListBans(e.Ctx, filter)
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(bans) == 0 {
		c.sys.replyCommandNotice(e, "BAN一覧", "BANエントリはありません。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, b := range bans {
		exp := "無期限"
		if b.ExpiresAt != nil {
			exp = b.ExpiresAt.In(bbscore.JSTLocation).Format("2006/01/02 15:04")
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s | Hash:%s | 期限:%s | 理由:%s", b.Scope, b.ID, b.PosterHash, exp, discordsync.TruncateRunes(b.Reason, 30)))
	}
	c.sys.replyCommandNotice(e, "BBS BAN一覧", strings.Join(lines, "\n"), discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleBanUnban(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	banID, err := uuid.Parse(strings.TrimSpace(data.String("ban")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "BAN IDが不正です。")
		return nil
	}
	if err := c.sys.Service.Store.DeleteBan(e.Ctx, banID); err != nil {
		c.sys.replyCommandError(e, "解除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "BANを解除しました", fmt.Sprintf("BanID: %s", banID), discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleAdminAboneButton(e *handler.ComponentEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e, errTitlePermission, "運営メンバーのみ実行できます。")
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyComponentError(e, errTitleInvalidInput, "通報IDが不正です。")
		return nil
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		c.sys.replyComponentError(e, "処理失敗", describeBBSError(err))
		return nil
	}
	post, err := c.sys.Service.AbonePostGlobal(e.Ctx, e.User().ID, report.PostID, "運営による通報対応")
	if err != nil {
		c.sys.replyComponentError(e, "あぼーん失敗", describeBBSError(err))
		return nil
	}
	if thread, tErr := c.sys.Service.GetThread(e.Ctx, post.ThreadID); tErr == nil {
		go c.sys.Syncer.SyncAbone(c.sys.Components.Ctx(), e.Client(), *post, *thread)
	}
	c.markReportStatus(e, report, bbscore.ReportStatusResolved, "global_abone")
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("グローバルあぼーんを実行しました", fmt.Sprintf("PostID: %s\nReportID: %s", post.ID, report.ID), discordsync.ColorError))
	return nil
}

func (c *adminCommand) handleAdminBanButton(e *handler.ComponentEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e, errTitlePermission, "運営メンバーのみ実行できます。")
		return nil
	}
	reportID := e.Vars["report_id"]
	if _, err := uuid.Parse(reportID); err != nil {
		c.sys.replyComponentError(e, errTitleInvalidInput, "通報IDが不正です。")
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/admin/banmodal/%s", reportID),
		"グローバルBAN + あぼーん",
		discord.NewLabel(
			"BAN期間 (時間数・空欄または0で無期限)",
			discord.NewShortTextInput("hours").WithRequired(false).WithPlaceholder("例: 168 (7日間) / 0 (無期限)").WithMaxLength(10),
		),
		discord.NewLabel(
			"理由 (監査ログに保存)",
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(200),
		),
	)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		c.sys.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (c *adminCommand) handleAdminBanModal(e *handler.ModalEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyModalError(e, errTitlePermission, "運営メンバーのみ実行できます。")
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyModalError(e, errTitleInvalidInput, "通報IDが不正です。")
		return nil
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		c.sys.replyModalError(e, "処理失敗", describeBBSError(err))
		return nil
	}
	hoursRaw := strings.TrimSpace(e.Data.Text("hours"))
	var dur *time.Duration
	if hoursRaw != "" && hoursRaw != "0" {
		hrs, pErr := strconv.Atoi(hoursRaw)
		if pErr != nil || hrs < 0 {
			c.sys.replyModalError(e, errTitleInvalidInput, "BAN期間は0以上の整数(時間数)で入力してください。")
			return nil
		}
		if hrs > 0 {
			d := time.Duration(hrs) * time.Hour
			dur = &d
		}
	}
	reason := e.Data.Text("reason")
	ban, err := c.sys.Service.BanGlobal(e.Ctx, e.User().ID, report.PostID, reason, dur)
	if err != nil {
		c.sys.replyModalError(e, "BAN失敗", describeBBSError(err))
		return nil
	}
	if post, aErr := c.sys.Service.AbonePostGlobal(e.Ctx, e.User().ID, report.PostID, reason); aErr == nil {
		if thread, tErr := c.sys.Service.GetThread(e.Ctx, post.ThreadID); tErr == nil {
			go c.sys.Syncer.SyncAbone(c.sys.Components.Ctx(), e.Client(), *post, *thread)
		}
	}
	c.markReportStatus(e, report, bbscore.ReportStatusResolved, "global_ban:"+reason)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("グローバルBANを実行しました", fmt.Sprintf("BanID: %s\nHash: %s", ban.ID, ban.PosterHash), discordsync.ColorError))
	return nil
}

func (c *adminCommand) handleAdminDismissButton(e *handler.ComponentEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e, errTitlePermission, "運営メンバーのみ実行できます。")
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyComponentError(e, errTitleInvalidInput, "通報IDが不正です。")
		return nil
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		c.sys.replyComponentError(e, "処理失敗", describeBBSError(err))
		return nil
	}
	c.markReportStatus(e, report, bbscore.ReportStatusDismissed, "dismissed")
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("通報を却下しました", fmt.Sprintf("ReportID: %s", report.ID), discordsync.ColorNotice))
	return nil
}

func (c *adminCommand) markReportStatus(e interface {
	User() discord.User
}, report *bbscore.Report, status bbscore.ReportStatus, note string) {
	actor := e.User().ID
	report.Status = status
	report.HandledBy = &actor
	report.ResolutionNote = note
	report.UpdatedAt = time.Now()
	_ = c.sys.Service.Store.UpdateReport(c.sys.Components.Ctx(), report)
}

func (c *adminCommand) autocompleteGlobalCategories(e *handler.AutocompleteEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeGlobal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, cat := range cats {
		if q == "" || strings.Contains(strings.ToLower(cat.Name), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(cat.Name, 90),
				Value: cat.ID.String(),
			})
		}
	}
	return e.AutocompleteResult(discordsync.ValidateAutocompleteChoices(choices))
}

func (c *adminCommand) autocompleteGlobalBoards(e *handler.AutocompleteEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, b := range boards {
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(b.Name, 90),
				Value: b.ID.String(),
			})
		}
	}
	return e.AutocompleteResult(discordsync.ValidateAutocompleteChoices(choices))
}

func (c *adminCommand) autocompleteReports(e *handler.AutocompleteEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		return e.AutocompleteResult(nil)
	}
	reports, err := c.sys.Service.Store.ListReports(e.Ctx, bbscore.ReportFilter{Limit: 25})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, r := range reports {
		label := fmt.Sprintf("[%s] %s - %s", r.Status, r.ID.String()[:8], discordsync.TruncateRunes(r.Reason, 45))
		if q == "" || strings.Contains(strings.ToLower(label), q) || strings.Contains(strings.ToLower(r.ID.String()), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(label, 95),
				Value: r.ID.String(),
			})
		}
	}
	return e.AutocompleteResult(discordsync.ValidateAutocompleteChoices(choices))
}

func (c *adminCommand) autocompleteBans(e *handler.AutocompleteEvent) error {
	if !c.sys.Service.IsOperator(e.User().ID) {
		return e.AutocompleteResult(nil)
	}
	bans, err := c.sys.Service.Store.ListBans(e.Ctx, bbscore.BanFilter{Limit: 25})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, b := range bans {
		label := fmt.Sprintf("[%s] %s (%s)", b.Scope, b.PosterHash, discordsync.TruncateRunes(b.Reason, 45))
		if q == "" || strings.Contains(strings.ToLower(label), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(label, 95),
				Value: b.ID.String(),
			})
		}
	}
	return e.AutocompleteResult(discordsync.ValidateAutocompleteChoices(choices))
}
