package commands

import (
	"time"

	"github.com/disgoorg/json"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func User(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "user",
			Description:  "user",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "set-birthday",
					Description: "set your own birthday",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionInt{
							Name:        "month",
							Description: "month",
							Required:    true,
							Choices: []discord.ApplicationCommandOptionChoiceInt{
								{
									Name:  "January",
									Value: 1,
								},
								{
									Name:  "February",
									Value: 2,
								},
								{
									Name:  "March",
									Value: 3,
								},
								{
									Name:  "April",
									Value: 4,
								},
								{
									Name:  "May",
									Value: 5,
								},
								{
									Name:  "June",
									Value: 6,
								},
								{
									Name:  "July",
									Value: 7,
								},
								{
									Name:  "August",
									Value: 8,
								},
								{
									Name:  "September",
									Value: 9,
								},
								{
									Name:  "October",
									Value: 10,
								},
								{
									Name:  "November",
									Value: 11,
								},
								{
									Name:  "December",
									Value: 12,
								},
							},
						},
						discord.ApplicationCommandOptionInt{
							Name:        "date",
							Description: "date of number",
							Required:    true,
							MinValue:    json.Ptr(1),
							MaxValue:    json.Ptr(31),
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"set-birthday": userSetBirthDayCommandHandler(b),
		},
	}
}

func userSetBirthDayCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.UserDataLock(event.User().ID).Lock()
		defer b.Self.UserDataLock(event.User().ID).Unlock()
		ud, err := b.Self.DB.UserData().Get(event.User().ID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		month := event.SlashCommandInteractionData().Int("month")
		day := event.SlashCommandInteractionData().Int("date")
		date := time.Date(time.Now().Year(), time.Month(month), day, 0, 0, 0, 0, ud.Location.Location)
		ud.BirthDay = [2]int{int(date.Month()), date.Day()}
		if err := b.Self.DB.UserData().Set(ud.ID, ud); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "user_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "user_set_birthday", translate.WithTemplate(map[string]any{"Date": date.Format("01/02")})))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}
