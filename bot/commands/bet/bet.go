package bet

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func ptr[T any](v T) *T {
	return &v
}

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "bet",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "bet",
				Description:              "Bet gopoints on various games",
				DescriptionLocalizations: i18n.TranslateTextMap("command.bet.description"),
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/bet": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("bet.use"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.Modal(discord.NewModalCreateBuilder().
						SetCustomID("bet:create").
						SetTitle(i18n.TranslateText(event.Locale(), "command.bet.create.title")).
						SetComponents(
							discord.NewLabel(i18n.TranslateText(event.Locale(), "command.bet.create.input.title.label"),
								discord.TextInputComponent{
									CustomID:    "title",
									Style:       discord.TextInputStyleShort,
									Placeholder: i18n.TranslateText(event.Locale(), "command.bet.create.input.title.placeholder"),
									Required:    true,
									MinLength:   ptr(1),
									MaxLength:   100,
								}),
							discord.NewLabel(i18n.TranslateText(event.Locale(), "command.bet.create.select.vote_type.label"),
								discord.StringSelectMenuComponent{
									CustomID:    "vote_type",
									Placeholder: i18n.TranslateText(event.Locale(), "command.bet.create.select.vote_type.placeholder"),
									MinValues:   ptr(1),
									MaxValues:   1,
									Required:    true,
									Options: []discord.StringSelectMenuOption{
										{
											Label:       i18n.TranslateText(event.Locale(), "command.bet.vote_type.guess"),
											Description: i18n.TranslateText(event.Locale(), "command.bet.vote_type.guess.description"),
											Value:       string(models.BetVoteTypeGuess),
										},
										{
											Label:       i18n.TranslateText(event.Locale(), "command.bet.vote_type.race"),
											Description: i18n.TranslateText(event.Locale(), "command.bet.vote_type.race.description"),
											Value:       string(models.BetVoteTypeRace),
										},
										{
											Label:       i18n.TranslateText(event.Locale(), "command.bet.vote_type.battle_royale"),
											Description: i18n.TranslateText(event.Locale(), "command.bet.vote_type.battle_royale.description"),
											Value:       string(models.BetVoteTypeBattleRoyale),
										},
									},
								},
							),
						).
						Build()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"bet:create": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				// Extract form data (variables prefixed with _ to avoid unused warnings until implementation is complete)
				_ = event.Data.Text("title")
				_ = models.BetVoteType(event.Data.StringValues("vote_type")[0])
				// TODO: Implement bet session creation logic
				// - Create BetHost with OwnerID from event.User().ID
				// - Set GuildID, ChannelID from event context
				// - Store in database
				return nil
			},
		},
	}).SetComponent(c)
}
