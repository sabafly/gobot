package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/json"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Minecraft(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "minecraft",
			Description:  "minecraft",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "status-panel",
					Description: "status-panel",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "create status panel",
							DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_create_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "server-name",
									Description:              "name of server",
									DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_create_command_server_name_option_description", false),
									Required:                 true,
									MaxLength:                json.Ptr(100),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "address",
									Description:              "address of server",
									DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_create_command_address_option_description", false),
									Required:                 true,
									MaxLength:                json.Ptr(32),
								},
								discord.ApplicationCommandOptionString{
									Name:                     "edition",
									Description:              "edition of server",
									DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_create_command_hide_address_option_description", false),
									Required:                 true,
									Choices: []discord.ApplicationCommandOptionChoiceString{
										{
											Name:  "java",
											Value: "java",
										},
										{
											Name:  "bedrock",
											Value: "bedrock",
										},
									},
								},
								discord.ApplicationCommandOptionBool{
									Name:                     "hide-address",
									Description:              "hide address",
									DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_create_command_hide_address_option_description", false),
								},
							},
						},
						{
							Name:                     "delete",
							Description:              "delete panel",
							DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_delete_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "panel",
									Description:              "target panel",
									DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_delete_command_panel_option", false),
									Autocomplete:             true,
									Required:                 true,
								},
							},
						},
						{
							Name:                     "list",
							Description:              "show list of panels",
							DescriptionLocalizations: translate.MessageMap("minecraft_status_panel_list_command_description", false),
						},
					},
				},
			},
		},
		AutocompleteHandlers: map[string]handler.AutocompleteHandler{
			"status-panel/delete": minecraftStatusPanelAutocomplete(b),
		},
		AutocompleteCheck: func(ctx *events.AutocompleteInteractionCreate) bool {
			if b.CheckDev(ctx.User().ID) {
				return true
			}
			if ctx.Member() != nil && ctx.Member().Permissions.Has(discord.PermissionManageGuild) {
				return true
			}
			gd, err := b.Self.DB.GuildData().Get(*ctx.GuildID())
			if err == nil {
				if gd.UserPermissions[ctx.User().ID].Has("mc.panel.manage") {
					return true
				}
				for _, id := range ctx.Member().RoleIDs {
					if gd.RolePermissions[id].Has("mc.panel.manage") {
						return true
					}
				}
			}
			_ = ctx.Result(nil)
			return false
		},
		Checks: map[string]handler.Check[*events.ApplicationCommandInteractionCreate]{
			"status-panel/create": b.Self.CheckCommandPermission(b, "mc.panel.manage", discord.PermissionManageGuild),
			"status-panel/delete": b.Self.CheckCommandPermission(b, "mc.panel.manage", discord.PermissionManageGuild),
			"status-panel/list":   b.Self.CheckCommandPermission(b, "mc.panel.manage", discord.PermissionManageGuild),
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"status-panel/create": minecraftStatusPanelCreateCommandHandler(b),
			"status-panel/delete": minecraftStatusPanelDeleteCommandHandler(b),
			"status-panel/list":   minecraftStatusPanelListCommandHandler(b),
		},
	}
}

var numIp = regexp.MustCompile(`[\d\D]{1,3}\.[\d\D]{1,3}\.[\d\D]{1,3}\.[\d\D]{1,3}`)
var invalidAddress = regexp.MustCompile(`[^\w\d\.\-_]`)

func minecraftStatusPanelCreateCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if len(gd.MCStatusPanel) >= gd.MCStatusPanelMax {
			return botlib.ReturnErrMessage(event, "error_guild_max_count_limit_has_reached")
		}
		var port uint16
		var s_type db.MinecraftServerType
		switch event.SlashCommandInteractionData().String("edition") {
		case "java":
			port = 25565
			s_type = db.MinecraftServerTypeJava
		case "bedrock":
			port = 19132
			s_type = db.MinecraftServerTypeBedrock
		}
		address := event.SlashCommandInteractionData().String("address")
		if numIp.MatchString(address) || invalidAddress.MatchString(address) {
			return botlib.ReturnErrMessage(event, "error_invalid_command_argument")
		}
		hash, err := db.Address2Hash(address, int(port))
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		server, err := b.Self.DB.MinecraftServer().Get(hash)
		if err != nil {
			if err != redis.Nil {
				return botlib.ReturnErr(event, err)
			}
			server = db.NewMinecraftServer(hash, address, port, s_type)
			err := b.Self.DB.MinecraftServer().Set(server.Hash, server)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
		}
		resp := server.LastResponse
		if resp == nil || server.LastResponseTime.Before(time.Now().Add(-5*time.Minute)) {
			resp, err = server.Fetch()
			if err != nil {
				return botlib.ReturnErrMessage(event, "error_failed_to_connect_server", botlib.WithTranslateData(map[string]any{"Err": err}))
			}
		}
		if err := b.Self.DB.MinecraftServer().Set(server.Hash, server); err != nil {
			return botlib.ReturnErr(event, err)
		}

		name := event.SlashCommandInteractionData().String("server-name")
		show_address := !event.SlashCommandInteractionData().Bool("hide-address")
		mcp := db.NewMinecraftStatusPanel(name, *event.GuildID(), event.Channel().ID(), 0, hash, show_address)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(mcp.Embed(address, resp))
		thumb := strings.ReplaceAll(resp.Favicon, "data:image/png;base64,", "")
		res, _ := base64.RawStdEncoding.DecodeString(thumb)
		if resp.Favicon != "" {
			message.AddFiles(discord.NewFile("favicon.png", "", bytes.NewBuffer(res)))
		}
		message.AddContainerComponents(mcp.Components()...)
		mes, err := event.Client().Rest().CreateMessage(mcp.ChannelID, message.Build())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		mcp.MessageID = mes.ID
		if err := b.Self.DB.MinecraftStatusPanel().Set(mcp.ID, mcp); err != nil {
			return botlib.ReturnErr(event, err)
		}

		gd.MCStatusPanelName[mcp.Name]++
		gd.MCStatusPanel[mcp.ID] = mcp.Name

		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetDescription(translate.Message(event.Locale(), "minecraft_status_panel_created"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		result_message := discord.NewMessageCreateBuilder()
		result_message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(result_message.SetFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func minecraftStatusPanelDeleteCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		panel_id, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		panel, err := b.Self.DB.MinecraftStatusPanel().Get(panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		server, err := b.Self.DB.MinecraftServer().Get(panel.Hash)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.MCStatusPanelName[panel.Name]--
		delete(gd.MCStatusPanel, panel.ID)
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		_ = event.Client().Rest().DeleteMessage(panel.ChannelID, panel.MessageID)
		if err := b.Self.DB.MinecraftStatusPanel().Del(panel.ID); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		message.SetFlags(discord.MessageFlagEphemeral)
		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("Successfully deleted")
		embed.SetDescriptionf("```\rName: %s\rAddress: %s:%d ```", panel.Name, server.Address, server.Port)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func minecraftStatusPanelListCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return err
		}
		var res string
		event.Client().Logger().Debug(gd.MCStatusPanel)
		for k := range gd.MCStatusPanel {
			panel, err := b.Self.DB.MinecraftStatusPanel().Get(k)
			if err != nil {
				continue
			}
			server, err := b.Self.DB.MinecraftServer().Get(panel.Hash)
			if err != nil {
				continue
			}
			res += fmt.Sprintf("- `%s` `%s:%d` %s\r", panel.Name, server.Address, server.Port, discord.MessageURL(panel.GuildID, panel.ChannelID, panel.MessageID))
		}
		res = fmt.Sprintf("## There are (%d/%d) panels\r%s", len(gd.MCStatusPanel), gd.MCStatusPanelMax, res)
		message := discord.NewMessageCreateBuilder().SetContent(res)
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func minecraftStatusPanelAutocomplete(b *botlib.Bot[*client.Client]) handler.AutocompleteHandler {
	return func(event *events.AutocompleteInteractionCreate) error {
		if !event.AutocompleteInteraction.Data.Options["panel"].Focused {
			_ = event.Result(nil)
			return nil
		}
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return err
		}
		var choices []discord.AutocompleteChoice
		for u, v := range gd.MCStatusPanel {
			if !strings.HasPrefix(v, event.AutocompleteInteraction.Data.String("panel")) {
				continue
			}
			panel, err := b.Self.DB.MinecraftStatusPanel().Get(u)
			if err != nil {
				continue
			}
			server, err := b.Self.DB.MinecraftServer().Get(panel.Hash)
			if err != nil {
				continue
			}
			name := panel.Name
			if gd.MCStatusPanelName[v] > 1 {
				name += fmt.Sprintf(" (%s:%d)", server.Address, server.Port)
			}
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  name,
				Value: panel.ID.String(),
			})
		}
		if err := event.Result(choices); err != nil {
			return err
		}
		return nil
	}
}

func MinecraftComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "minecraft",
		Handler: map[string]handler.ComponentHandler{
			"status-refresh": minecraftComponentStatusRefreshHandler(b),
		},
	}
}

func minecraftComponentStatusRefreshHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.ButtonInteractionData().CustomID(), ":")
		panel_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.MinecraftStatusPanel().Get(panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		server, err := b.Self.DB.MinecraftServer().Get(panel.Hash)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		resp := server.LastResponse
		if resp == nil || server.LastResponseTime.Before(time.Now().Add(-5*time.Minute)) {
			resp, err = server.Fetch()
			if err != nil {
				return botlib.ReturnErrMessage(event, "error_failed_to_connect_server", botlib.WithTranslateData(map[string]any{"Err": err}), botlib.WithEphemeral(true))
			}
		}
		if err := b.Self.DB.MinecraftServer().Set(server.Hash, server); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageUpdateBuilder()
		message.AddEmbeds(panel.Embed(server.Address, resp))
		thumb := strings.ReplaceAll(resp.Favicon, "data:image/png;base64,", "")
		res, _ := base64.RawStdEncoding.DecodeString(thumb)
		if resp.Favicon != "" {
			message.AddFiles(discord.NewFile("favicon.png", "", bytes.NewBuffer(res)))
		}
		message.AddContainerComponents(panel.Components()...)
		if _, err := event.Client().Rest().UpdateMessage(panel.ChannelID, panel.MessageID, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if err := event.DeferUpdateMessage(); err != nil {
			return err
		}
		return nil
	}
}
