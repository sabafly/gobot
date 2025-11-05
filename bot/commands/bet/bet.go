package bet

import (
	"fmt"

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
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "title",
						Description:              "Bet title",
						DescriptionLocalizations: i18n.TranslateTextMap("command.bet.option.title.description"),
						Required:                 true,
						MinLength:                ptr(1),
						MaxLength:                ptr(100),
					},
					discord.ApplicationCommandOptionString{
						Name:                     "mode",
						Description:              "Bet mode",
						DescriptionLocalizations: i18n.TranslateTextMap("command.bet.option.mode.description"),
						Required:                 true,
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{
								Name:              i18n.TranslateText(discord.LocaleJapanese, "command.bet.vote_type.guess"),
								NameLocalizations: i18n.TranslateTextMap("command.bet.vote_type.guess"),
								Value:             string(models.BetVoteTypeGuess),
							},
							{
								Name:              i18n.TranslateText(discord.LocaleJapanese, "command.bet.vote_type.race"),
								NameLocalizations: i18n.TranslateTextMap("command.bet.vote_type.race"),
								Value:             string(models.BetVoteTypeRace),
							},
							{
								Name:              i18n.TranslateText(discord.LocaleJapanese, "command.bet.vote_type.battle_royale"),
								NameLocalizations: i18n.TranslateTextMap("command.bet.vote_type.battle_royale"),
								Value:             string(models.BetVoteTypeBattleRoyale),
							},
						},
					},
					discord.ApplicationCommandOptionBool{
						Name:        "allow_vote_change",
						Description: "Allow users to change their vote destination after voting (optional, default: true)",
						Required:    false,
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/bet": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("bet.use"),
				},
				CommandHandler: handleBetCommand,
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"bet:config_poll": handlePollConfig,
			"bet:vote":        handleVote,
			"bet:decide":      handleDecideResult,
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"bet:vote_btn": generic.PComponentHandler{
				ComponentHandler: handleVoteButton,
			},
			"bet:decide_btn": generic.PComponentHandler{
				ComponentHandler: handleDecideButton,
			},
		},
	}).SetComponent(c)
}

// handleBetCommand handles the /bet command with title and mode arguments
func handleBetCommand(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	data := event.SlashCommandInteractionData()

	title, _ := data.OptString("title")
	mode, _ := data.OptString("mode")
	allowVoteChange, ok := data.OptBool("allow_vote_change")
	if !ok {
		allowVoteChange = true // default value
	}
	voteType := models.BetVoteType(mode)

	// For poll mode, show modal for options directly
	if voteType == models.BetVoteTypeGuess {
		// Encode allow_vote_change in custom ID
		customID := fmt.Sprintf("bet:config_poll:%t:%s", allowVoteChange, title)
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(customID).
			SetTitle(title).
			SetComponents(
				discord.NewLabel("選択肢",
					discord.TextInputComponent{
						CustomID:    "options",
						Style:       discord.TextInputStyleParagraph,
						Placeholder: "選択肢を改行で区切って入力してください\n選択肢1\n選択肢2\n選択肢3",
						Required:    true,
						MinLength:   ptr(3),
						MaxLength:   4000,
					}),
			).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	// For other modes, show not implemented message
	if err := event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("このモードはまだ実装されていません。通常モード（予想）を選択してください。").
		SetFlags(discord.MessageFlagEphemeral).
		Build()); err != nil {
		return errors.NewError(err)
	}
	return nil
}
