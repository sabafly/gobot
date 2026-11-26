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
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		{
			Name:              "none (Hidden)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs.choices.slip.none"),
			Value:             "none",
		},
		{
			Name:              "vvv (Nickname only)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs.choices.slip.vvv"),
			Value:             "vvv",
		},
		{
			Name:              "vvvvv (Nickname + Fixed ID / Korokoro)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs.choices.slip.vvvvv"),
			Value:             "vvvvv",
		},
	}
	statusChoices := []discord.ApplicationCommandOptionChoiceString{
		{
			Name:              "open (Open)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs_admin.choices.status.open"),
			Value:             string(bbscore.ReportStatusOpen),
		},
		{
			Name:              "resolved (Resolved)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs_admin.choices.status.resolved"),
			Value:             string(bbscore.ReportStatusResolved),
		},
		{
			Name:              "dismissed (Dismissed)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs_admin.choices.status.dismissed"),
			Value:             string(bbscore.ReportStatusDismissed),
		},
	}
	scopeChoices := []discord.ApplicationCommandOptionChoiceString{
		{
			Name:              "global (Global)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs_admin.choices.scope.global"),
			Value:             string(bbscore.ScopeGlobal),
		},
		{
			Name:              "local (Local)",
			NameLocalizations: i18n.TranslateTextMap("command.bbs_admin.choices.scope.local"),
			Value:             string(bbscore.ScopeLocal),
		},
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
	loc := e.Locale()
	c.sys.replyCommandError(e,
		i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
		i18n.TranslateText(loc, "components.bbs.errors.operator_required"),
	)
	return false
}

func (c *adminCommand) handleGlobalCategoryCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	cat, err := c.sys.Service.CreateCategory(e.Ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeGlobal,
		Name:        data.String("name"),
		Description: data.String("description"),
	})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.create_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.admin.cat.created_title"),
		i18n.TranslateText(loc, "components.bbs.cmd.category.created_body", map[string]any{
			"name": cat.Name,
			"id":   cat.ID.String(),
		}),
		discordsync.ColorGlobalBoard,
	)
	return nil
}

func (c *adminCommand) handleGlobalCategoryList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	scope := bbscore.ScopeGlobal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.admin.cat.list_title")
	if len(cats) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.admin.cat.list_empty"), discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, cat := range cats {
		lines = append(lines, "- "+cat.Name+" ("+cat.ID.String()+")")
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalCategoryDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.cat.invalid_id"),
		)
		return nil
	}
	delFailTitle := i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed")
	cat, err := c.sys.Service.GetCategory(e.Ctx, catID)
	if err != nil || cat.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, delFailTitle, i18n.TranslateText(loc, "components.bbs.admin.cat.not_found"))
		return nil
	}
	if err := c.sys.Service.DeleteCategory(e.Ctx, catID); err != nil {
		c.sys.replyCommandError(e, delFailTitle, describeBBSError(loc, err))
		return nil
	}
	c.sys.replyCommandNotice(e, i18n.TranslateText(loc, "components.bbs.admin.cat.deleted_title"), cat.Name, discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleGlobalBoardCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.cat.invalid_id"),
		)
		return nil
	}
	rules, rErr := parseBoardRulesFromSlash(data, loc)
	if rErr != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.admin.board.create_failed"),
			describeBBSError(loc, rErr),
		)
		return nil
	}
	board, err := c.sys.Service.CreateBoard(e.Ctx, bbscore.CreateBoardInput{
		CategoryID:  catID,
		Scope:       bbscore.ScopeGlobal,
		Name:        data.String("name"),
		Description: data.String("description"),
		Rules:       rules,
	})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.admin.board.create_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.admin.board.created_title"),
		i18n.TranslateText(loc, "components.bbs.admin.board.created_body", map[string]any{
			"name": board.Name,
			"id":   board.ID.String(),
		}),
		discordsync.ColorGlobalBoard,
	)
	return nil
}

func (c *adminCommand) handleGlobalBoardList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.admin.board.list_title")
	if len(boards) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.admin.board.list_empty"), discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, b := range boards {
		lines = append(lines, i18n.TranslateText(loc, "components.bbs.admin.board.list_entry", map[string]any{
			"name":         b.Name,
			"id":           b.ID.String(),
			"default_name": b.Rules.Normalize().DefaultName,
		}))
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *adminCommand) handleGlobalBoardEdit(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.board.invalid_id"),
		)
		return nil
	}
	editFailTitle := i18n.TranslateText(loc, "components.bbs.errors.title.edit_failed")
	b, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil || b.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, editFailTitle, i18n.TranslateText(loc, "components.bbs.admin.board.not_found"))
		return nil
	}
	input, pErr := parseUpdateBoardInput(data)
	if pErr != nil {
		c.sys.replyCommandError(e, editFailTitle, describeBBSError(loc, pErr))
		return nil
	}
	updated, err := c.sys.Service.UpdateBoard(e.Ctx, boardID, input)
	if err != nil {
		c.sys.replyCommandError(e, editFailTitle, describeBBSError(loc, err))
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.admin.board.updated_title"),
		i18n.TranslateText(loc, "components.bbs.admin.board.updated_body", map[string]any{
			"name":         updated.Name,
			"default_name": updated.Rules.Normalize().DefaultName,
		}),
		discordsync.ColorGlobalBoard,
	)
	return nil
}

func (c *adminCommand) handleGlobalBoardDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.board.invalid_id"),
		)
		return nil
	}
	delFailTitle := i18n.TranslateText(loc, "components.bbs.errors.title.delete_failed")
	b, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil || b.Scope != bbscore.ScopeGlobal {
		c.sys.replyCommandError(e, delFailTitle, i18n.TranslateText(loc, "components.bbs.admin.board.not_found"))
		return nil
	}
	if err := c.sys.Service.DeleteBoard(e.Ctx, boardID); err != nil {
		c.sys.replyCommandError(e, delFailTitle, describeBBSError(loc, err))
		return nil
	}
	c.sys.replyCommandNotice(e, i18n.TranslateText(loc, "components.bbs.admin.board.deleted_title"), b.Name, discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleReportList(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	filter := bbscore.ReportFilter{Limit: 25}
	if v, ok := data.OptString("status"); ok && v != "" {
		st := bbscore.ReportStatus(v)
		filter.Status = &st
	}
	reports, err := c.sys.Service.Store.ListReports(e.Ctx, filter)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.admin.report.list_title")
	if len(reports) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.admin.report.list_empty"), discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, r := range reports {
		lines = append(lines, i18n.TranslateText(loc, "components.bbs.admin.report.list_entry", map[string]any{
			"status":   string(r.Status),
			"id":       r.ID.String(),
			"guild_id": r.GuildID,
			"reason":   discordsync.TruncateRunes(r.Reason, 40),
		}))
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorError)
	return nil
}

func (c *adminCommand) handleReportInspect(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	reportID, err := uuid.Parse(strings.TrimSpace(data.String("report")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.report.invalid_id"),
		)
		return nil
	}
	report, audit, err := c.sys.Service.InspectReportForOperator(e.Ctx, e.User().ID, reportID)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.admin.report.audit_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}

	preview := i18n.TranslateText(loc, "components.bbs.admin.report.aboned_preview")
	if !audit.Post.Abone {
		preview = discordsync.TruncateRunes(audit.Post.Body, 400)
	}
	detail := i18n.TranslateText(loc, "components.bbs.admin.report.inspect_detail", map[string]any{
		"id":          report.ID.String(),
		"status":      string(report.Status),
		"scope":       string(report.Scope),
		"guild_id":    report.GuildID,
		"reporter_id": report.ReporterUserID,
		"post_id":     report.PostID.String(),
		"res":         audit.Post.ResNumber,
		"author_id":   audit.AuthorUserID,
		"hash":        audit.Post.PosterHash,
		"anon_id":     audit.Post.AnonID,
		"reason":      report.Reason,
		"preview":     preview,
	})

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.admin.report.inspect_title")),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(discordsync.TruncateRunes(detail, 1800)),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewDangerButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.report.btn.abone"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/admin/abone/%s", report.ID)),
				discord.NewDangerButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.report.btn.ban"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/admin/ban/%s", report.ID)),
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.report.btn.dismiss"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/admin/dismiss/%s", report.ID)),
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
	loc := e.Locale()
	filter := bbscore.BanFilter{Limit: 25}
	if v, ok := data.OptString("scope"); ok && v != "" {
		sc := bbscore.Scope(v)
		filter.Scope = &sc
	}
	bans, err := c.sys.Service.Store.ListBans(e.Ctx, filter)
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	listTitle := i18n.TranslateText(loc, "components.bbs.admin.ban.list_title")
	if len(bans) == 0 {
		c.sys.replyCommandNotice(e, listTitle, i18n.TranslateText(loc, "components.bbs.admin.ban.list_empty"), discordsync.ColorNotice)
		return nil
	}
	unlimited := i18n.TranslateText(loc, "components.bbs.common.unlimited")
	var lines []string
	for _, b := range bans {
		exp := unlimited
		if b.ExpiresAt != nil {
			exp = b.ExpiresAt.In(bbscore.JSTLocation).Format("2006/01/02 15:04")
		}
		lines = append(lines, i18n.TranslateText(loc, "components.bbs.admin.ban.list_entry", map[string]any{
			"scope":   string(b.Scope),
			"id":      b.ID.String(),
			"hash":    b.PosterHash,
			"expires": exp,
			"reason":  discordsync.TruncateRunes(b.Reason, 30),
		}))
	}
	c.sys.replyCommandNotice(e, listTitle, strings.Join(lines, "\n"), discordsync.ColorNotice)
	return nil
}

func (c *adminCommand) handleBanUnban(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.requireOperatorCmd(e) {
		return nil
	}
	loc := e.Locale()
	banID, err := uuid.Parse(strings.TrimSpace(data.String("ban")))
	if err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.ban.invalid_id"),
		)
		return nil
	}
	if err := c.sys.Service.Store.DeleteBan(e.Ctx, banID); err != nil {
		c.sys.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.admin.process_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	c.sys.replyCommandNotice(e,
		i18n.TranslateText(loc, "components.bbs.admin.ban.unbanned_title"),
		i18n.TranslateText(loc, "components.bbs.admin.ban.unbanned_body", map[string]any{"id": banID.String()}),
		discordsync.ColorNotice,
	)
	return nil
}

func (c *adminCommand) handleAdminAboneButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.operator_required"),
		)
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.report.invalid_id"),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs admin abone defer failed", "err", err)
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.process_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}
	post, err := c.sys.Service.AbonePostGlobal(e.Ctx, e.User().ID, report.PostID, "operator_report_abone")
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.mod.abone_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}
	thread, tErr := c.sys.Service.GetThread(e.Ctx, post.ThreadID)
	syncErr := tErr
	if tErr == nil {
		syncErr = c.sys.Syncer.SyncAbone(e.Ctx, e.Client(), *post, *thread)
	}
	if syncErr != nil {
		slog.Warn("bbs admin abone sync failed; keeping report open", "report_id", report.ID, "post_id", post.ID, "err", syncErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.abone_sync_failed_title"),
			i18n.TranslateText(loc, "components.bbs.admin.abone_sync_failed_body", map[string]any{
				"report_id": report.ID.String(),
				"err":       describeBBSError(loc, syncErr),
			}),
		))
		return nil
	}
	if sErr := c.markReportStatus(e.Ctx, e, report, bbscore.ReportStatusResolved, "global_abone"); sErr != nil {
		slog.Warn("bbs failed to mark report resolved after abone", "report_id", report.ID, "err", sErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_title"),
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_body", map[string]any{
				"id":  report.ID.String(),
				"err": describeBBSError(loc, sErr),
			}),
		))
		return nil
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.admin.aboned_title"),
		i18n.TranslateText(loc, "components.bbs.admin.aboned_body", map[string]any{
			"post_id":   post.ID.String(),
			"report_id": report.ID.String(),
		}),
		discordsync.ColorError,
	).WithEphemeral(true))
	return nil
}

func (c *adminCommand) handleAdminBanButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.operator_required"),
		)
		return nil
	}
	reportID := e.Vars["report_id"]
	if _, err := uuid.Parse(reportID); err != nil {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.report.invalid_id"),
		)
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/admin/banmodal/%s", reportID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.ban_modal_title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.ban_hours_label"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("hours").WithRequired(false).WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.ban_hours_placeholder"), discordsync.MaxTextInputPlaceholder)).WithMaxLength(10),
		),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.admin.ban_reason_label"), discordsync.MaxLabelTextRunes),
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(200),
		),
	)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open admin ban modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (c *adminCommand) handleAdminBanModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.operator_required"),
		)
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.report.invalid_id"),
		)
		return nil
	}
	hoursRaw := strings.TrimSpace(e.Data.Text("hours"))
	var dur *time.Duration
	if hoursRaw != "" && hoursRaw != "0" {
		hrs, pErr := strconv.Atoi(hoursRaw)
		if pErr != nil || hrs < 0 || hrs > bbscore.MaxBanHours {
			c.sys.replyModalError(e,
				i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
				i18n.TranslateText(loc, "components.bbs.mod.invalid_hours_range", map[string]any{"max": bbscore.MaxBanHours}),
			)
			return nil
		}
		if hrs > 0 {
			d := time.Duration(hrs) * time.Hour
			dur = &d
		}
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs admin ban modal defer failed", "err", err)
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.process_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}
	reason := e.Data.Text("reason")
	var ban *bbscore.Ban
	if report.BanID != nil {
		existingBan, gErr := c.sys.Service.Store.GetBan(e.Ctx, *report.BanID)
		if gErr == nil && existingBan != nil {
			ban = existingBan
		} else if !errors.Is(gErr, bbscore.ErrNotFound) {
			_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
				i18n.TranslateText(loc, "components.bbs.admin.ban_failed"),
				describeBBSError(loc, gErr),
			))
			return nil
		}
	}
	if ban == nil {
		ban, err = c.sys.Service.BanGlobal(e.Ctx, e.User().ID, report.PostID, reason, dur)
		if err != nil {
			_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
				i18n.TranslateText(loc, "components.bbs.admin.ban_failed"),
				describeBBSError(loc, err),
			))
			return nil
		}
		report.BanID = &ban.ID
		if uErr := c.sys.Service.Store.UpdateReport(e.Ctx, report); uErr != nil {
			slog.Warn("bbs failed to save ban_id on report", "report_id", report.ID, "ban_id", ban.ID, "err", uErr)
		}
	}
	post, aErr := c.sys.Service.AbonePostGlobal(e.Ctx, e.User().ID, report.PostID, reason)
	if aErr != nil {
		slog.Warn("bbs global ban succeeded but abone failed", "ban_id", ban.ID, "report_id", report.ID, "err", aErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.ban_abone_partial_title"),
			i18n.TranslateText(loc, "components.bbs.admin.ban_abone_partial_body", map[string]any{
				"id":        ban.ID.String(),
				"report_id": report.ID.String(),
				"err":       describeBBSError(loc, aErr),
			}),
		))
		return nil
	}
	thread, tErr := c.sys.Service.GetThread(e.Ctx, post.ThreadID)
	syncErr := tErr
	if tErr == nil {
		syncErr = c.sys.Syncer.SyncAbone(e.Ctx, e.Client(), *post, *thread)
	}
	if syncErr != nil {
		slog.Warn("bbs global ban+abone sync failed; keeping report open", "ban_id", ban.ID, "report_id", report.ID, "err", syncErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.abone_sync_failed_title"),
			i18n.TranslateText(loc, "components.bbs.admin.abone_sync_failed_body", map[string]any{
				"report_id": report.ID.String(),
				"err":       describeBBSError(loc, syncErr),
			}),
		))
		return nil
	}
	if sErr := c.markReportStatus(e.Ctx, e, report, bbscore.ReportStatusResolved, "global_ban:"+reason); sErr != nil {
		slog.Warn("bbs failed to mark report resolved after ban+abone", "report_id", report.ID, "err", sErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_title"),
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_body", map[string]any{
				"id":  report.ID.String(),
				"err": describeBBSError(loc, sErr),
			}),
		))
		return nil
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.admin.banned_title"),
		i18n.TranslateText(loc, "components.bbs.admin.banned_body", map[string]any{
			"id":   ban.ID.String(),
			"hash": ban.PosterHash,
		}),
		discordsync.ColorError,
	).WithEphemeral(true))
	return nil
}

func (c *adminCommand) handleAdminDismissButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !c.sys.Service.IsOperator(e.User().ID) {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.operator_required"),
		)
		return nil
	}
	reportID, err := uuid.Parse(e.Vars["report_id"])
	if err != nil {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.admin.report.invalid_id"),
		)
		return nil
	}
	report, err := c.sys.Service.Store.GetReport(e.Ctx, reportID)
	if err != nil {
		c.sys.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.admin.process_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if sErr := c.markReportStatus(e.Ctx, e, report, bbscore.ReportStatusDismissed, "dismissed"); sErr != nil {
		slog.Warn("bbs failed to mark report dismissed", "report_id", report.ID, "err", sErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_title"),
			i18n.TranslateText(loc, "components.bbs.admin.report.status_save_failed_body", map[string]any{
				"id":  report.ID.String(),
				"err": describeBBSError(loc, sErr),
			}),
		))
		return nil
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.admin.dismissed_title"),
		i18n.TranslateText(loc, "components.bbs.admin.dismissed_body", map[string]any{"id": report.ID.String()}),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	return nil
}

func (c *adminCommand) markReportStatus(ctx context.Context, e interface {
	User() discord.User
}, report *bbscore.Report, status bbscore.ReportStatus, note string) error {
	actor := e.User().ID
	report.Status = status
	report.HandledBy = &actor
	report.ResolutionNote = note
	report.UpdatedAt = time.Now()
	return c.sys.Service.Store.UpdateReport(ctx, report)
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
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		if q == "" || strings.Contains(strings.ToLower(cat.Name), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(cat.Name, 90),
				Value: cat.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs global category autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
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
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(b.Name, 90),
				Value: b.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs global board autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
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
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		label := fmt.Sprintf("[%s] %s - %s", r.Status, r.ID.String()[:8], discordsync.TruncateRunes(r.Reason, 45))
		if q == "" || strings.Contains(strings.ToLower(label), q) || strings.Contains(strings.ToLower(r.ID.String()), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(label, 95),
				Value: r.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs reports autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
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
		if len(choices) >= discordsync.MaxAutocompleteChoices {
			break
		}
		label := fmt.Sprintf("[%s] %s (%s)", b.Scope, b.PosterHash, discordsync.TruncateRunes(b.Reason, 45))
		if q == "" || strings.Contains(strings.ToLower(label), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(label, 95),
				Value: b.ID.String(),
			})
		}
	}
	if err := discordsync.ValidateAutocompleteChoices(choices); err != nil {
		slog.Warn("bbs bans autocomplete validation failed", "err", err)
		return e.AutocompleteResult(nil)
	}
	return e.AutocompleteResult(choices)
}
