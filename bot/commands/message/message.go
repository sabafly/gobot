package message

import (
	"context"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent/wordsuffix"
	"github.com/sabafly/gobot/internal/builtin"
)

func Command(c *components.Components) *generic.GenericCommand {
	return (&generic.GenericCommand{
		Namespace: "message",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "message",
				Description:  "message",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "suffix",
						Description: "suffix",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "set",
								Description: "set member's suffix",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:        "target",
										Description: "target",
										Required:    true,
									},
									discord.ApplicationCommandOptionString{
										Name:        "suffix",
										Description: "suffix",
										Required:    true,
										MaxLength:   builtin.Ptr(512),
									},
									discord.ApplicationCommandOptionString{
										Name:        "rule",
										Description: "rule",
										Required:    true,
										Choices: []discord.ApplicationCommandOptionChoiceString{
											{
												Name:  "webhook",
												Value: wordsuffix.RuleWebhook.String(),
											},
											{
												Name:  "warn",
												Value: wordsuffix.RuleWarn.String(),
											},
											{
												Name:  "delete",
												Value: wordsuffix.RuleDelete.String(),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/message/suffix/set": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) error {
				g, err := c.GuildCreateID(context.Background(), *event.GuildID())
				if err != nil {
					slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
					return err
				}
				u, err := c.UserCreate(context.Background(), event.SlashCommandInteractionData().User("target"))
				if err != nil {
					slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
					return err
				}
				if u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).ExistX(context.Background()) {
					w := u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).OnlyX(context.Background())
					w.Update().
						SetSuffix(event.SlashCommandInteractionData().String("suffix")).
						SetOwner(u).
						SetRule(wordsuffix.Rule(event.SlashCommandInteractionData().String("rule"))).
						SaveX(context.Background())
				} else {
					w := c.DB().WordSuffix.
						Create().
						SetGuild(g).
						SetSuffix(event.SlashCommandInteractionData().String("suffix")).
						SetOwner(u).
						SetRule(wordsuffix.Rule(event.SlashCommandInteractionData().String("rule"))).
						SaveX(context.Background())
					u.Update().AddWordSuffixIDs(w.ID).SaveX(context.Background())
				}
				event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").Build())

				return nil
			}),
		},

		EventHandler: func(c *components.Components, e bot.Event) error {
			switch e := e.(type) {
			case *events.GuildMessageCreate:
				slog.Debug("メッセージ作成")
				if e.Message.Type.System() || e.Message.Author.System || e.Message.Author.Bot {
					return nil
				}
				if e.Message.Type != discord.MessageTypeDefault && e.Message.Type != discord.MessageTypeReply {
					return nil
				}

				u, err := c.UserCreate(context.Background(), e.Message.Author)
				if err != nil {
					slog.Error("メッセージ著者取得に失敗", "err", err, "uid", e.Message.Author.ID)
					return err
				}
				if !u.QueryWordSuffix().Where(wordsuffix.GuildID(e.GuildID)).ExistX(context.Background()) {
					slog.Debug("語尾が存在しません")
					return nil
				}
				w := u.QueryWordSuffix().Where(wordsuffix.GuildID(e.GuildID)).OnlyX(context.Background())
				switch w.Rule {
				case wordsuffix.RuleDelete:
					if strings.HasSuffix(e.Message.Content, w.Suffix) {
						return nil
					}
					if err := e.Client().Rest().DeleteMessage(e.ChannelID, e.MessageID); err != nil {
						slog.Error("メッセージを削除できません", "err", err)
						return err
					}
				case wordsuffix.RuleWarn:
					if strings.HasSuffix(e.Message.Content, w.Suffix) {
						return nil
					}
				}
			}
			return nil
		},
	}).SetDB(c)
}
