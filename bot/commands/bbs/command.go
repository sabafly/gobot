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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	pendingTTL           = 10 * time.Minute
	maxPendingEntries    = 2048
	msgCmdReportName     = "BBS - レスを通報"
	msgCmdModerateName   = "BBS - モデレート"
	datCheckInterval     = 15 * time.Minute
	errTitlePermission   = "権限不足"
	errTitleInvalidInput = "入力エラー"
)

type pendingKind string

const (
	pendingNewThread  pendingKind = "new_thread"
	pendingNewPost    pendingKind = "new_post"
	pendingOpenReader pendingKind = "open_reader"
)

type pendingAction struct {
	Kind      pendingKind
	UserID    snowflake.ID
	TargetID  uuid.UUID
	Page      string
	ExpiresAt time.Time
}

// System coordinates Discord BBS command/component handlers, services, and syncer.
type System struct {
	Components *components.Components
	Service    *bbscore.Service
	Syncer     *discordsync.Syncer
	LogChannel *snowflake.ID

	pendingMu sync.Mutex
	pending   map[string]pendingAction
}

// NewSystem constructs a System orchestrating BBS commands and interactions.
func NewSystem(c *components.Components, svc *bbscore.Service, syncer *discordsync.Syncer, logCh *snowflake.ID) *System {
	return &System{
		Components: c,
		Service:    svc,
		Syncer:     syncer,
		LogChannel: logCh,
		pending:    make(map[string]pendingAction),
	}
}

func (s *System) savePending(userID snowflake.ID, kind pendingKind, targetID uuid.UUID, page string) string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	token := hex.EncodeToString(buf[:])

	now := time.Now()
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for k, v := range s.pending {
		if now.After(v.ExpiresAt) || len(s.pending) >= maxPendingEntries {
			delete(s.pending, k)
		}
	}
	s.pending[token] = pendingAction{
		Kind:      kind,
		UserID:    userID,
		TargetID:  targetID,
		Page:      page,
		ExpiresAt: now.Add(pendingTTL),
	}
	return token
}

func (s *System) popPending(token string, userID snowflake.ID) (pendingAction, bool) {
	if token == "" || token == "none" {
		return pendingAction{}, false
	}
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	act, ok := s.pending[token]
	if !ok {
		return pendingAction{}, false
	}
	if time.Now().After(act.ExpiresAt) || act.UserID != userID {
		if time.Now().After(act.ExpiresAt) {
			delete(s.pending, token)
		}
		return pendingAction{}, false
	}
	delete(s.pending, token)
	return act, true
}

type permEventAdapter struct {
	member  *discord.ResolvedMember
	guildID *snowflake.ID
	user    discord.User
	client  *bot.Client
}

func (p permEventAdapter) Member() *discord.ResolvedMember { return p.member }
func (p permEventAdapter) GuildID() *snowflake.ID          { return p.guildID }
func (p permEventAdapter) User() discord.User              { return p.user }
func (p permEventAdapter) Client() *bot.Client             { return p.client }

func (s *System) hasGuildAdmin(ctx context.Context, member *discord.ResolvedMember, guildID *snowflake.ID, user discord.User, client *bot.Client) bool {
	if guildID == nil {
		return false
	}
	return generic.CheckEventPermission(
		ctx,
		permEventAdapter{member: member, guildID: guildID, user: user, client: client},
		s.Components,
		[]generic.Permission{generic.PermissionString("bbs.admin")},
		discord.PermissionManageGuild,
	)
}

func (s *System) requireCmdGuildAdmin(e *handler.CommandEvent) bool {
	if s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		return true
	}
	s.replyCommandError(e, errTitlePermission, "この操作にはサーバー管理権限が必要です。")
	return false
}

func (s *System) replyCommandError(e *handler.CommandEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(title, body))
}

func (s *System) replyCommandNotice(e *handler.CommandEvent, title, body string, color int) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(title, body, color))
}

func (s *System) replyComponentError(e *handler.ComponentEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(title, body))
}

func (s *System) replyModalError(e *handler.ModalEvent, title, body string) {
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderError(title, body))
}

// Command returns the public /bbs command, message context menus, routes, event hooks, and dat-ochi scheduler.
func Command(sys *System) components.Command {
	return &bbsCommand{sys: sys}
}

type bbsCommand struct {
	sys *System
}

func (c *bbsCommand) Name() string { return "bbs" }

func (c *bbsCommand) IsPrivate() bool { return false }

func intPtr(v int) *int { return &v }

func (c *bbsCommand) Create() []discord.ApplicationCommandCreate {
	guildCtx := []discord.InteractionContextType{discord.InteractionContextTypeGuild}
	slipChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "none (非表示)", Value: "none"},
		{Name: "vvvvv (端末種別+OS)", Value: "vvvvv"},
	}

	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:                     "bbs",
			Description:              "Configure the anonymous BBS, manage boards, and review policies",
			DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.description"),
			Contexts:                 guildCtx,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:                     "setup",
					Description:              "Enable or disable BBS features and set an optional viewer role",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.description"),
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{
							Name:                     "enabled",
							Description:              "Enable BBS features in this server",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.options.enabled.description"),
							Required:                 true,
						},
						discord.ApplicationCommandOptionRole{
							Name:                     "viewer_role",
							Description:              "Role granted after accepting BBS policies",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.options.viewer_role.description"),
						},
						discord.ApplicationCommandOptionChannel{
							Name:                     "gate_channel",
							Description:              "Channel to post the consent role panel",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.setup.options.gate_channel.description"),
							ChannelTypes:             []discord.ChannelType{discord.ChannelTypeGuildText},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "category",
					Description:              "Manage server-local categories",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a server-local category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Category name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(60),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Category description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.create.options.description.description"),
									MaxLength:                intPtr(200),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List server-local categories",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.list.description"),
						},
						{
							Name:                     "delete",
							Description:              "Delete a server-local category",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Local category to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.category.delete.options.category.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "board",
					Description:              "Manage server-local boards",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "Create a local board and Discord channel",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "name",
									Description:              "Board name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.name.description"),
									Required:                 true,
									MaxLength:                intPtr(80),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "category",
									Description:              "Local category for the board",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.category.description"),
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "Board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "Default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "forum",
									Description:              "Create as a forum channel (falls back to text if non-Community)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.forum.description"),
								},
								discord.ApplicationCommandOptionChannel{
									Name:                     "discord_category",
									Description:              "Parent Discord category channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.discord_category.description"),
									ChannelTypes:             []discord.ChannelType{discord.ChannelTypeGuildCategory},
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP (Wachoi) display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.slip.description"),
									Choices:                  slipChoices,
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "max_res",
									Description:              "Max replies per thread (50-1000)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.max_res.description"),
									MinValue:                 intPtr(50),
									MaxValue:                 intPtr(1000),
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "archive_days",
									Description:              "Days of inactivity before auto-archive (0 disables)",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.create.options.archive_days.description"),
									MinValue:                 intPtr(0),
									MaxValue:                 intPtr(365),
								},
							},
						},
						{
							Name:                     "list",
							Description:              "List boards mounted in this server",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.list.description"),
						},
						{
							Name:                     "edit",
							Description:              "Edit a server-local board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Local board to edit",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "default_name",
									Description:              "New default anonymous name",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.default_name.description"),
									MaxLength:                intPtr(bbscore.MaxNameRunes),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "description",
									Description:              "New board description",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.description.description"),
									MaxLength:                intPtr(300),
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide_id",
									Description:              "Hide daily poster IDs",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.hide_id.description"),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "slip",
									Description:              "SLIP (Wachoi) display level",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.edit.options.slip.description"),
									Choices:                  slipChoices,
								},
							},
						},
						{
							Name:                     "delete",
							Description:              "Delete a local board and remove its mount",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.delete.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Local board to delete",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.board.delete.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:                     "global",
					Description:              "Browse and subscribe to global BBS boards",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.description"),
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "list",
							Description:              "List available global BBS boards",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.list.description"),
						},
						{
							Name:                     "subscribe",
							Description:              "Subscribe this server to a global BBS board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to subscribe",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "forum",
									Description:              "Create as a forum channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.forum.description"),
								},
								discord.ApplicationCommandOptionChannel{
									Name:                     "discord_category",
									Description:              "Parent Discord category channel",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.subscribe.options.discord_category.description"),
									ChannelTypes:             []discord.ChannelType{discord.ChannelTypeGuildCategory},
								},
							},
						},
						{
							Name:                     "unsubscribe",
							Description:              "Unsubscribe this server from a global board",
							DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.unsubscribe.description"),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "board",
									Description:              "Global board to unsubscribe",
									DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.global.unsubscribe.options.board.description"),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:                     "policy",
					Description:              "View BBS Terms, Guidelines, Privacy Policy, and consent status",
					DescriptionLocalizations: i18n.TranslateTextMap("command.bbs.policy.description"),
				},
			},
		},
		discord.MessageCommandCreate{
			Name:              msgCmdReportName,
			NameLocalizations: i18n.TranslateTextMap("command.bbs.msg.report.name"),
			Contexts:          guildCtx,
		},
		discord.MessageCommandCreate{
			Name:                     msgCmdModerateName,
			NameLocalizations:        i18n.TranslateTextMap("command.bbs.msg.moderate.name"),
			DefaultMemberPermissions: omit.NewPtr(discord.PermissionManageGuild),
			Contexts:                 guildCtx,
		},
	}
}

func (c *bbsCommand) RegisterRoutes(r handler.Router) {
	// Slash commands
	r.SlashCommand("/bbs/setup", c.handleSetup)
	r.SlashCommand("/bbs/category/create", c.handleCategoryCreate)
	r.SlashCommand("/bbs/category/list", c.handleCategoryList)
	r.SlashCommand("/bbs/category/delete", c.handleCategoryDelete)
	r.SlashCommand("/bbs/board/create", c.handleBoardCreate)
	r.SlashCommand("/bbs/board/list", c.handleBoardList)
	r.SlashCommand("/bbs/board/edit", c.handleBoardEdit)
	r.SlashCommand("/bbs/board/delete", c.handleBoardDelete)
	r.SlashCommand("/bbs/global/list", c.handleGlobalList)
	r.SlashCommand("/bbs/global/subscribe", c.handleGlobalSubscribe)
	r.SlashCommand("/bbs/global/unsubscribe", c.handleGlobalUnsubscribe)
	r.SlashCommand("/bbs/policy", c.handlePolicy)

	// Autocompletes
	r.Autocomplete("/bbs/category/delete", c.autocompleteLocalCategories)
	r.Autocomplete("/bbs/board/create", c.autocompleteLocalCategories)
	r.Autocomplete("/bbs/board/edit", c.autocompleteLocalBoards)
	r.Autocomplete("/bbs/board/delete", c.autocompleteLocalBoards)
	r.Autocomplete("/bbs/global/subscribe", c.autocompleteGlobalBoards)
	r.Autocomplete("/bbs/global/unsubscribe", c.autocompleteSubscribedGlobalBoards)

	// Message context menus
	r.MessageCommand("/"+msgCmdReportName, c.sys.handleReportMessageCommand)
	r.MessageCommand("/"+msgCmdModerateName, c.sys.handleModerateMessageCommand)

	// Consent & policy routes (static paths BEFORE parameterized routes)
	r.Component("/bbs/consent/revoke", c.sys.handleConsentRevoke)
	r.Component("/bbs/consent/view/{doc}", c.sys.handleConsentViewDoc)
	r.Component("/bbs/consent/accept/{token}", c.sys.handleConsentAccept)
	r.Component("/bbs/consent/role/{guild_id}", c.sys.handleConsentRole)

	// Posting & board/thread exploration routes
	r.Component("/bbs/board/threads/{mount_id}", c.sys.handleBoardThreads)
	r.Component("/bbs/board/jump/{mount_id}", c.sys.handleBoardJumpSelect)
	r.Component("/bbs/thread/new/{mount_id}", c.sys.handleNewThreadButton)
	r.Modal("/bbs/thread/modal/{mount_id}", c.sys.handleNewThreadModal)
	r.Component("/bbs/post/new/{mirror_id}", c.sys.handleNewPostButton)
	r.Modal("/bbs/post/modal/{mirror_id}", c.sys.handleNewPostModal)
	r.Component("/bbs/watch/toggle/{thread_id}", c.sys.handleWatchToggle)

	// Ephemeral read.cgi viewer (CRITICAL ORDER: static /jump BEFORE /{thread_id}/{page})
	r.Component("/bbs/reader/jump/{thread_id}", c.sys.handleReaderJumpButton)
	r.Modal("/bbs/reader/jumpmodal/{thread_id}", c.sys.handleReaderJumpModal)
	r.Component("/bbs/reader/{thread_id}/{page}", c.sys.handleReaderPage)

	// Reporting & local anonymous moderation routes
	r.Component("/bbs/report/thread/{thread_id}", c.sys.handleReportThreadButton)
	r.Modal("/bbs/report/threadmodal/{thread_id}", c.sys.handleReportThreadModal)
	r.Modal("/bbs/report/postmodal/{post_id}", c.sys.handleReportPostModal)
	r.Component("/bbs/mod/abone/{post_id}", c.sys.handleModAboneButton)
	r.Modal("/bbs/mod/abonemodal/{post_id}", c.sys.handleModAboneModal)
	r.Component("/bbs/mod/archive/{thread_id}", c.sys.handleModArchiveButton)
	r.Component("/bbs/mod/ban/{post_id}", c.sys.handleModBanButton)
	r.Modal("/bbs/mod/banmodal/{post_id}", c.sys.handleModBanModal)
}

func (c *bbsCommand) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return nil
}
func (c *bbsCommand) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return nil
}
func (c *bbsCommand) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return nil
}
func (c *bbsCommand) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
	return nil
}
func (c *bbsCommand) HasCommand(_ *events.ApplicationCommandInteractionCreate) bool { return false }

func (c *bbsCommand) OnEvent() func(event bot.Event) error {
	return func(event bot.Event) error {
		ctx := c.sys.Components.Ctx()
		switch e := event.(type) {
		case *events.GuildMessageUpdate:
			c.sys.Syncer.HandleMessageUpdate(ctx, e.Client(), e)
		case *events.GuildChannelDelete:
			c.sys.Syncer.HandleChannelDelete(ctx, e.ChannelID)
		case *events.ThreadDelete:
			c.sys.Syncer.HandleThreadDelete(ctx, e.ThreadID)
		}
		return nil
	}
}

func (c *bbsCommand) Scheduler() []components.Scheduler {
	return []components.Scheduler{
		{
			Duration: datCheckInterval,
			Worker: func(comp *components.Components, client *bot.Client) error {
				return c.sys.Syncer.RunDatOchiCheck(comp.Ctx(), client)
			},
		},
	}
}

func (c *bbsCommand) handleSetup(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}

	enabled := data.Bool("enabled")
	input := bbscore.GuildSetupInput{
		GuildID:     *guildID,
		Enabled:     enabled,
		ActorUserID: e.User().ID,
	}
	var viewerRoleID *snowflake.ID
	if role, ok := data.OptRole("viewer_role"); ok {
		rid := role.ID
		viewerRoleID = &rid
		input.ViewerRoleID = &rid
	}

	checker := discordsync.NewPreflightChecker(discordsync.NewClientGuildStateReader(e.Client()))
	if viewerRoleID != nil {
		if err := checker.CheckRoleAssign(*guildID, *viewerRoleID); err != nil {
			c.sys.replyCommandError(e, "閲覧ロール設定エラー", err.Error())
			return nil
		}
	}

	cfg, err := c.sys.Service.SetupGuild(e.Ctx, input)
	if err != nil {
		c.sys.replyCommandError(e, "セットアップ失敗", describeBBSError(err))
		return nil
	}

	summary := fmt.Sprintf("BBS機能: %v\n同意ポリシーバージョン: %s", cfg.Enabled, cfg.AgreedVersion)
	if cfg.ViewerRoleID != nil {
		summary += fmt.Sprintf("\n閲覧ロール: <@&%d>", *cfg.ViewerRoleID)
	}

	if gateCh, ok := data.OptChannel("gate_channel"); ok {
		if cfg.ViewerRoleID == nil || *cfg.ViewerRoleID == 0 {
			summary += "\n警告: gate_channel にパネルを設置するには viewer_role も指定してください。"
		} else {
			panel := discordsync.RenderRoleGatePanel(*guildID, *cfg.ViewerRoleID, c.sys.Service.Policy.CurrentVersion)
			validated, vErr := c.sys.Syncer.Validator.ValidateAndClampMessageCreate(panel)
			if vErr != nil {
				summary += fmt.Sprintf("\nロールゲートパネル検証失敗: %v", vErr)
			} else if _, sendErr := e.Client().Rest.CreateMessage(gateCh.ID, validated); sendErr != nil {
				summary += fmt.Sprintf("\nロールゲートパネル投稿失敗 (<#%d>): %v", gateCh.ID, sendErr)
			} else {
				summary += fmt.Sprintf("\nロールゲートパネルを設置しました: <#%d>", gateCh.ID)
			}
		}
	}

	c.sys.replyCommandNotice(e, "BBS設定を更新しました", summary, discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleCategoryCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	name := data.String("name")
	desc := data.String("description")
	cat, err := c.sys.Service.CreateCategory(e.Ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeLocal,
		GuildID:     guildID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		c.sys.replyCommandError(e, "カテゴリ作成失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "ローカルカテゴリを作成しました", fmt.Sprintf("名前: %s\nID: %s", cat.Name, cat.ID), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleCategoryList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: guildID})
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(cats) == 0 {
		c.sys.replyCommandNotice(e, "ローカルカテゴリ一覧", "ローカルカテゴリはまだありません。/bbs category create で作成してください。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, cat := range cats {
		line := fmt.Sprintf("- %s (%s)", cat.Name, cat.ID)
		if cat.Description != "" {
			line += " -- " + discordsync.TruncateRunes(cat.Description, 60)
		}
		lines = append(lines, line)
	}
	c.sys.replyCommandNotice(e, "ローカルカテゴリ一覧", strings.Join(lines, "\n"), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleCategoryDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	catID, err := uuid.Parse(strings.TrimSpace(data.String("category")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "カテゴリの選択が不正です。補完候補から選択してください。")
		return nil
	}
	cat, err := c.sys.Service.GetCategory(e.Ctx, catID)
	if err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	if cat.Scope != bbscore.ScopeLocal || cat.GuildID == nil || *cat.GuildID != *guildID {
		c.sys.replyCommandError(e, errTitlePermission, "このサーバーのローカルカテゴリのみ削除できます。")
		return nil
	}
	if err := c.sys.Service.DeleteCategory(e.Ctx, catID); err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "ローカルカテゴリを削除しました", fmt.Sprintf("カテゴリ: %s", cat.Name), discordsync.ColorNotice)
	return nil
}

func (c *bbsCommand) handleBoardCreate(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs board create defer failed", "err", err)
	}

	cfg, err := c.sys.Service.EnsureGuildEnabled(e.Ctx, *guildID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("板作成失敗", describeBBSError(err)))
		return nil
	}

	catID, err := c.resolveOrCreateDefaultLocalCategory(e.Ctx, *guildID, data.String("category"))
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("カテゴリ解決失敗", describeBBSError(err)))
		return nil
	}

	rules := parseBoardRulesFromSlash(data)
	wantForum := true
	if f, ok := data.OptBool("forum"); ok {
		wantForum = f
	}
	var discordCatID *snowflake.ID
	if ch, ok := data.OptChannel("discord_category"); ok {
		id := ch.ID
		discordCatID = &id
	}

	board, err := c.sys.Service.CreateBoard(e.Ctx, bbscore.CreateBoardInput{
		CategoryID:  catID,
		Scope:       bbscore.ScopeLocal,
		GuildID:     guildID,
		Name:        data.String("name"),
		Description: data.String("description"),
		Rules:       rules,
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("板作成失敗", describeBBSError(err)))
		return nil
	}

	mount, plan, err := c.sys.Syncer.ProvisionBoardChannel(e.Ctx, e.Client(), *guildID, *board, discordCatID, wantForum, cfg.ViewerRoleID, e.Locale())
	if err != nil {
		_ = c.sys.Service.DeleteBoard(e.Ctx, board.ID)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("チャンネル作成失敗", fmt.Sprintf("Discordチャンネルの作成に失敗したため板作成を中止しました: %v", err)))
		return nil
	}

	msg := fmt.Sprintf("板: %s\nチャンネル: <#%d>\n最大レス数: %d", board.Name, mount.ChannelID, board.Rules.MaxRes)
	if plan.FallbackReason == "community_required" {
		msg += "\n注意: このサーバーはCommunity機能が無効のため、テキストチャンネルとして作成しました。"
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("ローカル板を作成しました", msg, discordsync.ColorLocalBoard))
	return nil
}

func (c *bbsCommand) resolveOrCreateDefaultLocalCategory(ctx context.Context, guildID snowflake.ID, rawCat string) (uuid.UUID, error) {
	if strings.TrimSpace(rawCat) != "" {
		id, err := uuid.Parse(strings.TrimSpace(rawCat))
		if err != nil {
			return uuid.Nil, fmt.Errorf("%w: invalid category", bbscore.ErrInvalidInput)
		}
		cat, err := c.sys.Service.GetCategory(ctx, id)
		if err != nil {
			return uuid.Nil, err
		}
		if cat.Scope != bbscore.ScopeLocal || cat.GuildID == nil || *cat.GuildID != guildID {
			return uuid.Nil, bbscore.ErrUnauthorized
		}
		return id, nil
	}
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: &guildID})
	if err != nil {
		return uuid.Nil, err
	}
	if len(cats) > 0 {
		return cats[0].ID, nil
	}
	created, err := c.sys.Service.CreateCategory(ctx, bbscore.CreateCategoryInput{
		Scope:       bbscore.ScopeLocal,
		GuildID:     &guildID,
		Name:        "一般",
		Description: "デフォルトカテゴリ",
	})
	if err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

func (c *bbsCommand) handleBoardList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	mounts, err := c.sys.Service.Store.ListBoardMountsByGuild(e.Ctx, *guildID)
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(mounts) == 0 {
		c.sys.replyCommandNotice(e, "マウント済み板一覧", "このサーバーにマウントされている板はありません。", discordsync.ColorNotice)
		return nil
	}
	var lines []string
	for _, m := range mounts {
		b, err := c.sys.Service.GetBoard(e.Ctx, m.BoardID)
		if err != nil {
			continue
		}
		scopeLabel := "ローカル"
		if b.Scope == bbscore.ScopeGlobal {
			scopeLabel = "グローバル"
		}
		modeLabel := "Text"
		if m.IsForum {
			modeLabel = "Forum"
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s -> <#%d> (%s / 上限%dレス)", scopeLabel, b.Name, m.ChannelID, modeLabel, b.Rules.Normalize().MaxRes))
	}
	if len(lines) == 0 {
		c.sys.replyCommandNotice(e, "マウント済み板一覧", "表示可能な板がありません。", discordsync.ColorNotice)
		return nil
	}
	c.sys.replyCommandNotice(e, "マウント済み板一覧", strings.Join(lines, "\n"), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) handleBoardEdit(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "板の指定が不正です。")
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		c.sys.replyCommandError(e, "編集失敗", describeBBSError(err))
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		c.sys.replyCommandError(e, errTitlePermission, "このサーバーのローカル板のみ編集できます。")
		return nil
	}
	input := parseUpdateBoardInput(data)
	updated, err := c.sys.Service.UpdateBoard(e.Ctx, boardID, input)
	if err != nil {
		c.sys.replyCommandError(e, "編集失敗", describeBBSError(err))
		return nil
	}
	c.refreshLocalMountGuide(e.Ctx, e.Client(), *guildID, *updated, e.Locale())
	c.sys.replyCommandNotice(e, "ローカル板を更新しました", fmt.Sprintf("板: %s\n名無し: %s\nID非表示: %v", updated.Name, updated.Rules.Normalize().DefaultName, updated.Rules.HideID), discordsync.ColorLocalBoard)
	return nil
}

func (c *bbsCommand) refreshLocalMountGuide(ctx context.Context, client *bot.Client, guildID snowflake.ID, board bbscore.Board, locale discord.Locale) {
	mount, err := c.sys.Service.Store.GetBoardMountByGuildAndBoard(ctx, guildID, board.ID)
	if err != nil || !mount.Active || mount.ControlMessageID == nil {
		return
	}
	guide := discordsync.RenderBoardGuide(board, mount.ID, locale)
	mu := discord.MessageUpdate{
		Components:      &guide.Components,
		Flags:           &guide.Flags,
		AllowedMentions: guide.AllowedMentions,
	}
	validated, err := c.sys.Syncer.Validator.ValidateAndClampMessageUpdate(mu, true)
	if err != nil {
		return
	}
	_, _ = client.Rest.UpdateMessage(mount.ChannelID, *mount.ControlMessageID, validated)
}

func (c *bbsCommand) handleBoardDelete(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "板の指定が不正です。")
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		c.sys.replyCommandError(e, errTitlePermission, "このサーバーのローカル板のみ削除できます。")
		return nil
	}
	if mount, mErr := c.sys.Service.Store.GetBoardMountByGuildAndBoard(e.Ctx, *guildID, boardID); mErr == nil {
		_ = c.sys.Service.Store.DeactivateBoardMount(e.Ctx, mount.ID)
	}
	if err := c.sys.Service.DeleteBoard(e.Ctx, boardID); err != nil {
		c.sys.replyCommandError(e, "削除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "ローカル板を削除しました", fmt.Sprintf("板: %s (Discordチャンネル自体は安全のため残されています)", board.Name), discordsync.ColorNotice)
	return nil
}

func (c *bbsCommand) handleGlobalList(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		c.sys.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if len(boards) == 0 {
		c.sys.replyCommandNotice(e, "グローバル板一覧", "公開されているグローバル板はまだありません。", discordsync.ColorGlobalBoard)
		return nil
	}
	var lines []string
	for _, b := range boards {
		line := fmt.Sprintf("- %s (ID: %s)", b.Name, b.ID)
		if b.Description != "" {
			line += " -- " + discordsync.TruncateRunes(b.Description, 60)
		}
		lines = append(lines, line)
	}
	c.sys.replyCommandNotice(e, "グローバル板一覧", strings.Join(lines, "\n"), discordsync.ColorGlobalBoard)
	return nil
}

func (c *bbsCommand) handleGlobalSubscribe(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs global subscribe defer failed", "err", err)
	}

	cfg, err := c.sys.Service.EnsureGuildEnabled(e.Ctx, *guildID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("購読失敗", describeBBSError(err)))
		return nil
	}

	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(errTitleInvalidInput, "グローバル板の選択が不正です。"))
		return nil
	}
	board, err := c.sys.Service.GetBoard(e.Ctx, boardID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("購読失敗", describeBBSError(err)))
		return nil
	}
	if board.Scope != bbscore.ScopeGlobal {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(errTitleInvalidInput, "指定された板はグローバル板ではありません。"))
		return nil
	}

	wantForum := true
	if f, ok := data.OptBool("forum"); ok {
		wantForum = f
	}
	var discordCatID *snowflake.ID
	if ch, ok := data.OptChannel("discord_category"); ok {
		id := ch.ID
		discordCatID = &id
	}

	mount, plan, err := c.sys.Syncer.ProvisionBoardChannel(e.Ctx, e.Client(), *guildID, *board, discordCatID, wantForum, cfg.ViewerRoleID, e.Locale())
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("購読失敗", describeBBSError(err)))
		return nil
	}

	msg := fmt.Sprintf("グローバル板「%s」を購読し、<#%d> にマウントしました。\n既存スレッドは新着書き込みまたはスレッド一覧から選択時に遅延展開されます。", board.Name, mount.ChannelID)
	if plan.FallbackReason == "community_required" {
		msg += "\n注意: Community機能が無効なためテキストチャンネルとして作成しました。"
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("グローバル板を購読しました", msg, discordsync.ColorGlobalBoard))
	return nil
}

func (c *bbsCommand) handleGlobalUnsubscribe(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.sys.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	boardID, err := uuid.Parse(strings.TrimSpace(data.String("board")))
	if err != nil {
		c.sys.replyCommandError(e, errTitleInvalidInput, "グローバル板の選択が不正です。")
		return nil
	}
	mount, err := c.sys.Service.Store.GetBoardMountByGuildAndBoard(e.Ctx, *guildID, boardID)
	if err != nil {
		c.sys.replyCommandError(e, "解除失敗", describeBBSError(err))
		return nil
	}
	if err := c.sys.Service.Store.DeactivateBoardMount(e.Ctx, mount.ID); err != nil {
		c.sys.replyCommandError(e, "解除失敗", describeBBSError(err))
		return nil
	}
	c.sys.replyCommandNotice(e, "グローバル板の購読を解除しました", fmt.Sprintf("マウント <#%d> の同期を停止しました。", mount.ChannelID), discordsync.ColorNotice)
	return nil
}

func (c *bbsCommand) handlePolicy(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	consent, err := c.sys.Service.CheckConsent(e.Ctx, e.User().ID)
	pm := c.sys.Service.Policy

	if err != nil {
		msg := discordsync.RenderConsentGate(pm, e.Locale(), "none")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, msg)
		return nil
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ BBS ポリシー同意状況"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplayf("現在のステータス: 同意済み\n同意バージョン: %s\n最新バージョン: %s\n同意日時: %s",
				consent.PolicyVersion, pm.CurrentVersion, consent.AgreedAt.In(bbscore.JSTLocation).Format("2006/01/02 15:04:05")),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewSecondaryButton("利用規約", "/bbs/consent/view/terms"),
				discord.NewSecondaryButton("ガイドライン", "/bbs/consent/view/guidelines"),
				discord.NewSecondaryButton("プライバシーポリシー", "/bbs/consent/view/privacy"),
			),
			discord.NewActionRow(
				discord.NewDangerButton("同意を撤回する", "/bbs/consent/revoke"),
			),
		).WithAccentColor(discordsync.ColorNotice),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (c *bbsCommand) autocompleteLocalCategories(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeLocal
	cats, err := c.sys.Service.ListCategories(e.Ctx, bbscore.CategoryFilter{Scope: &scope, GuildID: guildID})
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

func (c *bbsCommand) autocompleteLocalBoards(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	scope := bbscore.ScopeLocal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope, GuildID: guildID})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) autocompleteGlobalBoards(e *handler.AutocompleteEvent) error {
	scope := bbscore.ScopeGlobal
	boards, err := c.sys.Service.ListBoards(e.Ctx, bbscore.BoardFilter{Scope: &scope})
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) autocompleteSubscribedGlobalBoards(e *handler.AutocompleteEvent) error {
	guildID := e.GuildID()
	if guildID == nil {
		return e.AutocompleteResult(nil)
	}
	mounts, err := c.sys.Service.Store.ListBoardMountsByGuild(e.Ctx, *guildID)
	if err != nil {
		return e.AutocompleteResult(nil)
	}
	var boards []bbscore.Board
	for _, m := range mounts {
		b, bErr := c.sys.Service.GetBoard(e.Ctx, m.BoardID)
		if bErr == nil && b.Scope == bbscore.ScopeGlobal {
			boards = append(boards, *b)
		}
	}
	return c.respondBoardChoices(e, boards)
}

func (c *bbsCommand) respondBoardChoices(e *handler.AutocompleteEvent, boards []bbscore.Board) error {
	q := strings.ToLower(strings.TrimSpace(e.Data.Focused().String()))
	var choices []discord.AutocompleteChoice
	for _, b := range boards {
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) || strings.Contains(strings.ToLower(b.Slug), q) {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  discordsync.TruncateRunes(b.Name, 90),
				Value: b.ID.String(),
			})
		}
	}
	return e.AutocompleteResult(discordsync.ValidateAutocompleteChoices(choices))
}

func parseBoardRulesFromSlash(data discord.SlashCommandInteractionData) bbscore.BoardRules {
	rules := bbscore.BoardRules{}
	if v, ok := data.OptString("default_name"); ok && strings.TrimSpace(v) != "" {
		rules.DefaultName = strings.TrimSpace(v)
	}
	if v, ok := data.OptBool("hide_id"); ok {
		rules.HideID = v
	}
	if v, ok := data.OptString("slip"); ok && v == "vvvvv" {
		rules.SLIP = bbscore.SLIPVVVVV
	}
	if v, ok := data.OptInt("max_res"); ok && v >= 50 && v <= 1000 {
		rules.MaxRes = uint32(v)
	}
	if v, ok := data.OptInt("archive_days"); ok {
		if v <= 0 {
			rules.DatOchiIdleHours = -1
			rules.DatOchiLowResHours = -1
		} else {
			rules.DatOchiIdleHours = v * 24
			rules.DatOchiLowResHours = v * 24
		}
	}
	return rules.Normalize()
}

func parseUpdateBoardInput(data discord.SlashCommandInteractionData) bbscore.UpdateBoardInput {
	var input bbscore.UpdateBoardInput
	if v, ok := data.OptString("default_name"); ok {
		input.DefaultName = &v
	}
	if v, ok := data.OptString("description"); ok {
		input.Description = &v
	}
	if v, ok := data.OptBool("hide_id"); ok {
		input.HideID = &v
	}
	if v, ok := data.OptString("slip"); ok {
		lvl := bbscore.SLIPNone
		if v == "vvvvv" {
			lvl = bbscore.SLIPVVVVV
		}
		input.SLIP = &lvl
	}
	if v, ok := data.OptBool("locked"); ok {
		input.Locked = &v
	}
	return input
}

func describeBBSError(err error) string {
	switch {
	case errors.Is(err, bbscore.ErrGuildNotEnabled):
		return "このサーバーではBBS機能が有効化されていません。サーバー管理者が /bbs setup enabled:true を実行してください。"
	case errors.Is(err, bbscore.ErrConsentRequired):
		return "利用規約・ガイドライン・プライバシーポリシーへの同意が必要です。"
	case errors.Is(err, bbscore.ErrBanned):
		return "規制(BAN)中のため書き込みできません。"
	case errors.Is(err, bbscore.ErrRateLimited):
		return "連続投稿制限中です。少し時間を置いてから投稿してください。"
	case errors.Is(err, bbscore.ErrThreadArchived):
		return "このスレッドは過去ログ倉庫に格納されているため書き込みできません。"
	case errors.Is(err, bbscore.ErrMaxResReached):
		return "このスレッドは上限レス数に達したため書き込みできません。"
	case errors.Is(err, bbscore.ErrBoardLocked):
		return "この板は現在ロックされています。"
	case errors.Is(err, bbscore.ErrUnauthorized):
		return "この操作を実行する権限がありません。"
	case errors.Is(err, bbscore.ErrAlreadyExists):
		return "すでに同じ対象が登録されています。"
	case errors.Is(err, bbscore.ErrNotFound):
		return "対象のデータが見つかりませんでした。"
	case errors.Is(err, bbscore.ErrInvalidInput):
		msg := strings.TrimPrefix(err.Error(), bbscore.ErrInvalidInput.Error())
		msg = strings.TrimPrefix(msg, ": ")
		if utf8.RuneCountInString(msg) == 0 {
			return "入力内容が不正です。"
		}
		return msg
	default:
		return err.Error()
	}
}
