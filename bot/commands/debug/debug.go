package debug

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sabafly/gobot/bot/commands/debug/db"
	"github.com/sabafly/gobot/bot/commands/role"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) *generic.GenericCommand {
	return (&generic.GenericCommand{
		Namespace: "debug",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "debug",
				Description:              "debug",
				DMPermission:             builtin.Ptr(false),
				DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "translate",
						Description: "translate",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "get",
								Description: "get translate",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "key",
										Description: "translate key",
										Required:    true,
									},
									discord.ApplicationCommandOptionString{
										Name:        "locale",
										Description: "locale",
										Required:    true,
									},
								},
							},
							{
								Name:        "reload",
								Description: "reload translate",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "redis",
						Description: "redis",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "import",
								Description: "import",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "addr",
										Description: "address",
										Required:    true,
									},
									discord.ApplicationCommandOptionInt{
										Name:        "db",
										Description: "db",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/debug/translate/get": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				key := event.SlashCommandInteractionData().String("key")
				locale := discord.Locale(event.SlashCommandInteractionData().String("locale"))
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent(translate.Message(locale, key)).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/translate/reload": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if _, err := translate.LoadDir(c.Config().TranslateDir); err != nil {
					slog.Error("翻訳ファイルを読み込めません", "err", err)
					return errors.NewError(err)
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent("OK").
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/redis/import": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				client := redis.NewClient(&redis.Options{
					Addr: event.SlashCommandInteractionData().String("addr"),
					DB:   event.SlashCommandInteractionData().Int("db"),
				})
				// GuildData
				guildCmd := client.HGetAll(event, "guild-data")
				if err := guildCmd.Err(); err != nil {
					return errors.NewError(err)
				}

				rpv2Cmd := client.HGetAll(event, "role-panel-v2")
				if err := rpv2Cmd.Err(); err != nil {
					return errors.NewError(err)
				}

				rpv2List := map[uuid.UUID]db.RolePanelV2{}

				for _, v := range rpv2Cmd.Val() {
					var rpv2 db.RolePanelV2
					if err := json.Unmarshal([]byte(v), &rpv2); err != nil {
						slog.Error("unmarshalに失敗", "err", err)
						continue
					}
					rpv2List[rpv2.ID] = rpv2
				}

				for _, v := range guildCmd.Val() {
					var guildData db.GuildData
					if err := json.Unmarshal([]byte(v), &guildData); err != nil {
						slog.Error("unmarshalに失敗", "err", err)
						continue
					}
					if guildData.DataVersion == nil || *guildData.DataVersion != 11 {
						continue
					}

					g, err := c.GuildCreateID(event, guildData.ID)
					if err != nil {
						slog.Error("guild取得に失敗", slog.Any("err", err))
						continue
					}

					createRolePanelBulk := []*ent.RolePanelCreate{}

					for u := range guildData.RolePanelV2 {
						rpv2, ok := rpv2List[u]
						if !ok {
							continue
						}

						roles := make([]schema.Role, len(rpv2.Roles))

						for i, role := range rpv2.Roles {
							roles[i] = schema.Role{
								ID:    role.RoleID,
								Name:  role.RoleName,
								Emoji: role.Emoji,
							}
						}

						createRolePanelBulk = append(createRolePanelBulk,
							c.DB().RolePanel.Create().
								SetID(rpv2.ID).
								SetRoles(roles).
								SetName(rpv2.Name).
								SetGuild(g).
								SetDescription(rpv2.Description),
						)
					}

					rolePanels, err := c.DB().RolePanel.CreateBulk(createRolePanelBulk...).Save(event)
					if err != nil {
						return errors.NewError(err)
					}

					placedIDMap := map[[2]snowflake.ID]uuid.UUID{}
					for k, u := range guildData.RolePanelV2Placed {
						ks := strings.Split(k, "/")
						channelID, messageID := snowflake.MustParse(ks[0]), snowflake.MustParse(ks[1])
						placedIDMap[[2]snowflake.ID{channelID, messageID}] = u
					}

					for k, u := range placedIDMap {
						index := slices.IndexFunc(rolePanels, func(rp *ent.RolePanel) bool { return rp.ID == u })
						if index == -1 {
							continue
						}

						keyString := fmt.Sprintf("%d/%d", k[0], k[1])

						placed := c.DB().RolePanelPlaced.Create().
							SetChannelID(k[0]).
							SetMessageID(k[1]).
							SetType(rolepanelplaced.Type(guildData.RolePanelV2PlacedConfig[keyString].PanelType)).
							SetButtonType(guildData.RolePanelV2PlacedConfig[keyString].ButtonStyle).
							SetFoldingSelectMenu(guildData.RolePanelV2PlacedConfig[keyString].SimpleSelectMenu).
							SetUseDisplayName(guildData.RolePanelV2PlacedConfig[keyString].UseDisplayName).
							SetShowName(guildData.RolePanelV2PlacedConfig[keyString].ButtonShowName).
							SetHideNotice(guildData.RolePanelV2PlacedConfig[keyString].HideNotice).
							SetRolePanel(rolePanels[index]).
							SetGuild(g).
							SaveX(event)

						role.UpdateRolePanel(event, rolePanels[index], placed, event.Locale(), event.Client())
					}

				}

				if err := event.RespondMessage(discord.NewMessageBuilder().SetContent("OK")); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetComponent(c)
}
