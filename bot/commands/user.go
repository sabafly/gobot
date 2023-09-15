package commands

import (
	"time"

	"github.com/disgoorg/json"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
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
					Name:                     "set-birthday",
					Description:              "set your own birthday",
					DescriptionLocalizations: translate.MessageMap("user_set_birthday_command_description", false),
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionInt{
							Name:                     "month",
							Description:              "month",
							DescriptionLocalizations: translate.MessageMap("user_set_birthday_command_month_description", false),
							Required:                 true,
							Choices: []discord.ApplicationCommandOptionChoiceInt{
								{
									Name:              "January",
									NameLocalizations: translate.MessageMap("january", false),
									Value:             1,
								},
								{
									Name:              "February",
									NameLocalizations: translate.MessageMap("february", false),
									Value:             2,
								},
								{
									Name:              "March",
									NameLocalizations: translate.MessageMap("march", false),
									Value:             3,
								},
								{
									Name:              "April",
									NameLocalizations: translate.MessageMap("april", false),
									Value:             4,
								},
								{
									Name:              "May",
									NameLocalizations: translate.MessageMap("may", false),
									Value:             5,
								},
								{
									Name:              "June",
									NameLocalizations: translate.MessageMap("june", false),
									Value:             6,
								},
								{
									Name:              "July",
									NameLocalizations: translate.MessageMap("july", false),
									Value:             7,
								},
								{
									Name:              "August",
									NameLocalizations: translate.MessageMap("august", false),
									Value:             8,
								},
								{
									Name:              "September",
									NameLocalizations: translate.MessageMap("september", false),
									Value:             9,
								},
								{
									Name:              "October",
									NameLocalizations: translate.MessageMap("october", false),
									Value:             10,
								},
								{
									Name:              "November",
									NameLocalizations: translate.MessageMap("november", false),
									Value:             11,
								},
								{
									Name:              "December",
									NameLocalizations: translate.MessageMap("december", false),
									Value:             12,
								},
							},
						},
						discord.ApplicationCommandOptionInt{
							Name:                     "date",
							Description:              "date of number",
							DescriptionLocalizations: translate.MessageMap("user_set_birthday_command_date_description", false),
							Required:                 true,
							MinValue:                 json.Ptr(1),
							MaxValue:                 json.Ptr(31),
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
		b.Self.DB.GuildData().Mu(event.User().ID).Lock()
		defer b.Self.DB.GuildData().Mu(event.User().ID).Unlock()
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
