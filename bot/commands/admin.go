package commands

import (
	"bytes"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/handler"
)

func Admin(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:        "admin",
			Description: "admin only",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "message",
					Description: "about message",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "get",
							Description: "get channel message",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "channel-id",
									Description: "channel id",
									Required:    true,
								},
								discord.ApplicationCommandOptionString{
									Name:        "message-id",
									Description: "message id",
									Required:    false,
								},
								discord.ApplicationCommandOptionString{
									Name:        "after-id",
									Description: "message id",
									Required:    false,
								},
								discord.ApplicationCommandOptionString{
									Name:        "before-id",
									Description: "message id",
									Required:    false,
								},
								discord.ApplicationCommandOptionString{
									Name:        "around-id",
									Description: "message id",
									Required:    false,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "channel",
					Description: "about channel",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "get",
							Description: "get guild channel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "guild-id",
									Description: "guild id",
									Required:    true,
								},
								discord.ApplicationCommandOptionString{
									Name:        "channel-id",
									Description: "channel id",
									Required:    false,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "guild",
					Description: "about guild",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "get",
							Description: "get guild",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "guild-id",
									Description: "guild id",
									Required:    false,
								},
							},
						},
						{
							Name:        "leave",
							Description: "leave guild",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "guild-id",
									Description: "guild id",
									Required:    true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "application",
					Description: "about application",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "command-get",
							Description: "get application commands",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "guild-id",
									Description: "guild id",
									Required:    false,
								},
								discord.ApplicationCommandOptionString{
									Name:        "command-id",
									Description: "command id",
									Required:    false,
								},
							},
						},
						{
							Name:        "command-delete",
							Description: "delete application commands",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "command-id",
									Description: "command id",
									Required:    true,
								},
								discord.ApplicationCommandOptionString{
									Name:        "guild-id",
									Description: "guild id",
									Required:    false,
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"message/get":                adminCommandMessageGetHandler(b),
			"channel/get":                adminCommandChannelGetHandler(b),
			"guild/get":                  adminCommandGuildGetHandler(b),
			"guild/leave":                adminCommandGuildLeaveHandler(b),
			"application/command-get":    adminCommandApplicationCommandGet(b),
			"application/command-delete": adminCommandApplicationCommandDelete(b),
		},
		Check: func(ctx *events.ApplicationCommandInteractionCreate) bool {
			if b.CheckDev(ctx.User().ID) {
				return true
			}
			if ctx.GuildID() != nil && b.CheckDev(*ctx.GuildID()) && ctx.Member().Permissions.Has(discord.PermissionAdministrator) {
				return true
			}
			return false
		},
		DevOnly: false,
	}
}

func adminCommandGuildLeaveHandler(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		guildID := snowflake.MustParse(event.SlashCommandInteractionData().String("guild-id"))
		if err := event.Client().Rest().LeaveGuild(guildID); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func adminCommandApplicationCommandDelete(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		commandID := snowflake.MustParse(event.SlashCommandInteractionData().String("command-id"))
		var err error
		if id, ok := event.SlashCommandInteractionData().OptString("guild-id"); ok {
			err = event.Client().Rest().DeleteGuildCommand(event.ApplicationID(), snowflake.MustParse(id), commandID)
		} else {
			err = event.Client().Rest().DeleteGlobalCommand(event.ApplicationID(), commandID)
		}
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = event.CreateMessage(discord.MessageCreate{
			Content: "OK",
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func adminCommandApplicationCommandGet(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if id, ok := event.SlashCommandInteractionData().OptString("command-id"); ok {
			var err error
			commandID := snowflake.MustParse(id)
			var command discord.ApplicationCommand
			if id, ok := event.SlashCommandInteractionData().OptString("guild-id"); ok {
				command, err = event.Client().Rest().GetGuildCommand(event.ApplicationID(), snowflake.MustParse(id), commandID)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}
			} else {
				command, err = event.Client().Rest().GetGlobalCommand(event.ApplicationID(), commandID)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}
			}
			raw, err := json.MarshalIndent(command, "", "  ")
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			err = event.CreateMessage(discord.MessageCreate{
				Files: []*discord.File{
					{
						Name:   fmt.Sprintf("command-%d.json", command.ID()),
						Reader: bytes.NewReader(raw),
					},
				},
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
		} else {
			channel, _ := event.Channel()
			var commands []discord.ApplicationCommand
			var err error
			if id, ok := event.SlashCommandInteractionData().OptString("guild-id"); ok {
				commands, err = event.Client().Rest().GetGuildCommands(event.ApplicationID(), snowflake.MustParse(id), false)
			} else {
				commands, err = event.Client().Rest().GetGlobalCommands(event.ApplicationID(), false)
			}
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			var description []string
			var temp string
			for _, ac := range commands {
				s := fmt.Sprintf("id:%d name:%s type:%d\r", ac.ID(), ac.Name(), ac.Type())
				if len(s)+len(temp) >= 4096 {
					description = append(description, temp)
					temp = ""
				}
				temp += s
			}
			embeds := []discord.Embed{}
			for _, v := range description {
				embeds = append(embeds, discord.Embed{
					Description: v,
				})
			}
			embeds = append(embeds, discord.Embed{
				Description: temp,
			})
			mEmbeds := [][]discord.Embed{}
			for len(embeds) > 0 {
				mEmbeds = append(mEmbeds, embeds[:func() int {
					if len(embeds) < 1 {
						return len(embeds)
					} else {
						return 1
					}
				}()])
				if len(embeds) > 1 {
					embeds = embeds[1:]
				} else {
					embeds = []discord.Embed{}
				}
			}
			embeds = botlib.SetEmbedProperties(mEmbeds[0])
			err = event.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			if len(mEmbeds) > 1 {
				for i, v := range mEmbeds {
					if i == 0 {
						continue
					}
					v = botlib.SetEmbedProperties(v)
					_, err := botlib.SendWebhook(event.Client(), channel.ID(), discord.WebhookMessageCreate{Embeds: v})
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func adminCommandGuildGetHandler(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if id, ok := event.SlashCommandInteractionData().OptString("guild-id"); ok {
			guildID := snowflake.MustParse(id)
			guild, err := event.Client().Rest().GetGuild(guildID, true)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			raw, err := json.MarshalIndent(guild, "", "  ")
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			err = event.CreateMessage(discord.MessageCreate{
				Files: []*discord.File{
					{
						Name:   fmt.Sprintf("guild-%d.json", guild.ID),
						Reader: bytes.NewReader(raw),
					},
				},
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
		} else {
			channel, _ := event.Channel()
			var description []string
			var temp string
			event.Client().Caches().GuildsForEach(func(guild discord.Guild) {
				s := fmt.Sprintf("id:%d members:%d name:%s join:%s\r", guild.ID, guild.MemberCount, guild.Name, guild.JoinedAt.Format(time.DateTime))
				if len(s)+len(temp) >= 4096 {
					description = append(description, temp)
					temp = ""
				}
				temp += s
			})
			embeds := []discord.Embed{}
			for _, v := range description {
				embeds = append(embeds, discord.Embed{
					Description: v,
				})
			}
			embeds = append(embeds, discord.Embed{
				Description: temp,
			})
			mEmbeds := [][]discord.Embed{}
			for len(embeds) > 0 {
				mEmbeds = append(mEmbeds, embeds[:func() int {
					if len(embeds) < 1 {
						return len(embeds)
					} else {
						return 1
					}
				}()])
				if len(embeds) > 1 {
					embeds = embeds[1:]
				} else {
					embeds = []discord.Embed{}
				}
			}
			embeds = botlib.SetEmbedProperties(mEmbeds[0])
			err := event.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			if len(mEmbeds) > 1 {
				for i, v := range mEmbeds {
					if i == 0 {
						continue
					}
					v = botlib.SetEmbedProperties(v)
					_, err := botlib.SendWebhook(event.Client(), channel.ID(), discord.WebhookMessageCreate{Embeds: v})
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func adminCommandChannelGetHandler(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if id, ok := event.SlashCommandInteractionData().OptString("channel-id"); ok {
			channelID := snowflake.MustParse(id)
			channel, err := event.Client().Rest().GetChannel(channelID)
			if err != nil {
				return nil
			}
			raw, err := json.MarshalIndent(channel, "", "  ")
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			err = event.CreateMessage(discord.MessageCreate{
				Files: []*discord.File{
					{
						Name:   fmt.Sprintf("channel-%d.json", channel.ID()),
						Reader: bytes.NewReader(raw),
					},
				},
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
		} else {
			channel, _ := event.Channel()
			guildID := snowflake.MustParse(event.SlashCommandInteractionData().String("guild-id"))
			channels, err := event.Client().Rest().GetGuildChannels(guildID)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			var description []string
			var temp string
			for _, gc := range channels {
				s := fmt.Sprintf("ch-id:%d type:%d prt%d name:%s\r", gc.ID(), gc.Type(), gc.ParentID(), gc.Name())
				if len(s)+len(temp) >= 4096 {
					description = append(description, temp)
					temp = ""
				}
				temp += s
			}
			embeds := []discord.Embed{}
			for _, v := range description {
				embeds = append(embeds, discord.Embed{
					Description: v,
				})
			}
			embeds = append(embeds, discord.Embed{
				Description: temp,
			})
			mEmbeds := [][]discord.Embed{}
			for len(embeds) > 0 {
				mEmbeds = append(mEmbeds, embeds[:func() int {
					if len(embeds) < 1 {
						return len(embeds)
					} else {
						return 1
					}
				}()])
				if len(embeds) > 1 {
					embeds = embeds[1:]
				} else {
					embeds = []discord.Embed{}
				}
			}
			embeds = botlib.SetEmbedProperties(mEmbeds[0])
			err = event.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			if len(mEmbeds) > 1 {
				for i, v := range mEmbeds {
					if i == 0 {
						continue
					}
					v = botlib.SetEmbedProperties(v)
					_, err := event.Client().Rest().CreateMessage(channel.ID(), discord.MessageCreate{
						Embeds: v,
					})
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func adminCommandMessageGetHandler(b *botlib.Bot) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		channel, _ := event.Channel()
		channelID := snowflake.MustParse(event.SlashCommandInteractionData().String("channel-id"))
		if messageID, ok := event.SlashCommandInteractionData().OptString("message-id"); ok {
			mes, err := event.Client().Rest().GetMessage(channelID, snowflake.MustParse(messageID))
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			raw, err := json.MarshalIndent(mes, "", "  ")
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			embeds := []discord.Embed{
				{
					Author: &discord.EmbedAuthor{
						Name:    fmt.Sprintf("%s#%s", mes.Author.Username, mes.Author.Discriminator),
						IconURL: mes.Author.EffectiveAvatarURL(),
						URL:     fmt.Sprintf("https://discord.com/users/%d", mes.Author.ID),
					},
					Description: mes.Content,
				},
			}
			embeds = botlib.SetEmbedProperties(embeds)
			err = event.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Files: []*discord.File{
					{
						Name:   fmt.Sprintf("message-%d.json", mes.ID),
						Reader: bytes.NewReader(raw),
					},
				},
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
		} else {
			var after, before, around snowflake.ID
			if str, ok := event.SlashCommandInteractionData().OptString("after-id"); ok {
				after = snowflake.MustParse(str)
			}
			if str, ok := event.SlashCommandInteractionData().OptString("before-id"); ok {
				before = snowflake.MustParse(str)
			}
			if str, ok := event.SlashCommandInteractionData().OptString("around-id"); ok {
				around = snowflake.MustParse(str)
			}
			channelMes, err := event.Client().Rest().GetMessages(channelID, around, before, after, 100)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			var description []string
			var temp string
			for _, m := range channelMes {
				s := fmt.Sprintf("%s#%s(%d) mes-id:%d[link](%s)\r", m.Author.Username, m.Author.Discriminator, m.Author.ID, m.ID, m.JumpURL())
				if len(s)+len(temp) >= 4000 {
					description = append(description, temp)
					temp = ""
				}
				temp += s
			}
			embeds := []discord.Embed{}
			for _, v := range description {
				embeds = append(embeds, discord.Embed{
					Description: v,
				})
			}
			embeds = append(embeds, discord.Embed{
				Description: temp,
			})
			mEmbeds := [][]discord.Embed{}
			for len(embeds) > 0 {
				mEmbeds = append(mEmbeds, embeds[:func() int {
					if len(embeds) < 1 {
						return len(embeds)
					} else {
						return 1
					}
				}()])
				if len(embeds) > 1 {
					embeds = embeds[1:]
				} else {
					embeds = []discord.Embed{}
				}
			}
			embeds = mEmbeds[0]
			embeds = botlib.SetEmbedProperties(embeds)
			err = event.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			if len(mEmbeds) > 1 {
				for i, v := range mEmbeds {
					if i == 0 {
						continue
					}
					v = botlib.SetEmbedProperties(v)
					_, err := event.Client().Rest().CreateMessage(channel.ID(), discord.MessageCreate{
						Embeds: v,
					})
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}
