package setting

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

// TODO: 君たちはどう設定を作るか

func Command(c *components.Components) components.Command {
	return (&generic.GenericCommand{
		Namespace: "setting",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "setting",
				Description:  "setting",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "bump",
						Description: "bump",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "toggle",
								Description: "toggle",
							},
							{
								Name:        "message",
								Description: "set message",
							},
							{
								Name:        "mention",
								Description: "set mention target",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "up",
						Description: "up",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "toggle",
								Description: "toggle",
							},
							{
								Name:        "message",
								Description: "set message",
							},
							{
								Name:        "mention",
								Description: "set mention target",
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/setting/bump/toggle": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.bump.toggle"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g = g.Update().
						SetBumpEnabled(!g.BumpEnabled).
						SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.bump.toggle."+builtin.Or(g.BumpEnabled, "enabled", "disabled"))).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/up/toggle": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.up.toggle"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g = g.Update().
						SetUpEnabled(!g.UpEnabled).
						SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.up.toggle."+builtin.Or(g.UpEnabled, "enabled", "disabled"))).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},

		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			switch event := event.(type) {
			case *events.GuildMessageCreate:
				if event.Message.Interaction == nil || event.Message.ApplicationID == nil {
					return nil
				}
				if event.Message.Author.ID != c.Config().BumpUserID && event.Message.Author.ID != c.Config().UpUserID {
					return nil
				}
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				if g.BumpEnabled {
					if err := bumpHandler(c, g, event); err != nil {
						return errors.NewError(err)
					}
				}
				if g.UpEnabled {
					if err := upHandler(c, g, event); err != nil {
						return errors.NewError(err)
					}
				}
			}
			return nil
		},

		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker: func(c *components.Components, client bot.Client) error {
					bumpLock.Lock()
					defer bumpLock.Unlock()
					for k, n := range bumpNotice {
						g, err := c.GuildCreateID(context.Background(), n.guildID)
						if err != nil {
							continue
						}
						if !g.BumpEnabled {
							continue
						}

						if time.Now().After(n.t.Add(-time.Minute * 2)) {
							go func() {
								time.Sleep(time.Until(n.t))
								createNotice(g.BumpRemindMessageTitle, g.BumpRemindMessage, n, client)
							}()
							delete(bumpNotice, k)
						}
					}
					upLock.Lock()
					defer upLock.Unlock()
					for k, n := range upNotice {
						g, err := c.GuildCreateID(context.Background(), n.guildID)
						if err != nil {
							continue
						}
						if !g.UpEnabled {
							continue
						}

						if time.Now().After(n.t.Add(-time.Minute * 2)) {
							go func() {
								time.Sleep(time.Until(n.t))
								createNotice(g.UpRemindMessageTitle, g.UpRemindMessage, n, client)
							}()
							delete(bumpNotice, k)
						}
					}

					return nil
				},
			},
		},
	}).SetComponent(c)
}

type notice struct {
	channelID snowflake.ID
	guildID   snowflake.ID
	t         time.Time
}

var bumpNotice = map[snowflake.ID]notice{}
var bumpLock sync.Mutex

func bumpHandler(c *components.Components, g *ent.Guild, event *events.GuildMessageCreate) error {
	bumpLock.Lock()
	defer bumpLock.Unlock()
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "bump" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Image == nil || event.Message.Embeds[0].Image.URL != c.Config().BumpImage {
		return nil
	}
	if !g.BumpEnabled {
		return nil
	}
	n :=
		notice{
			channelID: event.ChannelID,
			guildID:   event.GuildID,
			t:         event.Message.CreatedAt.Add(time.Hour * 2),
		}
	bumpNotice[event.GuildID] = n
	createNotice(g.BumpMessageTitle, g.BumpMessage, n, event.Client())
	return nil
}

var upNotice = map[snowflake.ID]notice{}
var upLock sync.Mutex

func upHandler(c *components.Components, g *ent.Guild, event *events.GuildMessageCreate) error {
	upLock.Lock()
	defer upLock.Unlock()
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "dissoku up" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Image == nil || event.Message.Embeds[0].Image.URL != c.Config().BumpImage {
		return nil
	}
	if !g.UpEnabled {
		return nil
	}
	n :=
		notice{
			channelID: event.ChannelID,
			guildID:   event.GuildID,
			t:         event.Message.CreatedAt.Add(time.Hour * 1),
		}
	upNotice[event.GuildID] = n
	createNotice(g.UpMessageTitle, g.UpMessage, n, event.Client())
	return nil
}

func createNotice(title, message string, n notice, client bot.Client) {
	if _, err := client.Rest().CreateMessage(n.channelID,
		discord.NewMessageBuilder().
			SetEmbeds(
				embeds.SetEmbedProperties(
					discord.NewEmbedBuilder().
						SetTitle(title).
						SetDescription(message).
						Build(),
				),
			).
			Create(),
	); err != nil {
		slog.Error("通知作成に失敗", slog.Any("err", err))
		return
	}
}
