package bet

import (
	"fmt"
	"time"

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
						Name:                     "allow_vote_change",
						NameLocalizations:        i18n.TranslateCommandOptionMap("command.bet.option.allow_vote_change.name"),
						Description:              "Allow users to change their vote destination after voting (optional, default: false)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.bet.option.allow_vote_change.description"),
						Required:                 false,
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
			"bet:config_poll":          handlePollConfig,
			"bet:config_race":          handleRaceConfig,
			"bet:config_battle_royale": handleBattleRoyaleConfig,
			"bet:vote":                 handleVote,
			"bet:decide":               handleDecideResult,
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"bet:vote_btn": generic.PComponentHandler{
				ComponentHandler: handleVoteButton,
			},
			"bet:decide_btn": generic.PComponentHandler{
				ComponentHandler: handleDecideButton,
			},
			"bet:close_vote_btn": generic.PComponentHandler{
				ComponentHandler: handleCloseVoteButton,
			},
			"bet:entry_btn": generic.PComponentHandler{
				ComponentHandler: handleEntryButton,
			},
			"bet:start_vote_btn": generic.PComponentHandler{
				ComponentHandler: handleStartVoteButton,
			},
			"bet:br_entry_btn": generic.PComponentHandler{
				ComponentHandler: handleBattleRoyaleEntryButton,
			},
			"bet:br_close_entry_btn": generic.PComponentHandler{
				ComponentHandler: handleBattleRoyaleCloseEntryButton,
			},
			"bet:cancel_entry_btn": generic.PComponentHandler{
				ComponentHandler: handleCancelEntryButton,
			},
		},

		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker:   betSchedulerWorker,
			},
		},
	}).SetComponent(c)
}

// handleBetCommand handles the /bet command with title and mode arguments
func handleBetCommand(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	data := event.SlashCommandInteractionData()
	locale := event.Locale()

	title, _ := data.OptString("title")
	mode, _ := data.OptString("mode")
	allowVoteChange, ok := data.OptBool("allow_vote_change")
	if !ok {
		allowVoteChange = false // default value
	}
	voteType := models.BetVoteType(mode)

	// For poll mode, show modal for options directly
	if voteType == models.BetVoteTypeGuess {
		// Encode allow_vote_change in custom ID
		customID := fmt.Sprintf("bet:config_poll:%t", allowVoteChange)
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(customID).
			SetTitle(i18n.TranslateText(locale, "command.bet.modal.create_poll.title")).
			SetComponents(
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_poll.input.title.label"),
					discord.TextInputComponent{
						CustomID:    "title",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_poll.input.title.placeholder"),
						Required:    true,
						MinLength:   ptr(1),
						MaxLength:   100,
						Value:       title,
					},
				),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_poll.input.options.label"),
					discord.TextInputComponent{
						CustomID:    "options",
						Style:       discord.TextInputStyleParagraph,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_poll.input.options.placeholder"),
						Required:    true,
						MinLength:   ptr(3),
						MaxLength:   4000,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_poll.input.prize_pool.label"),
					discord.TextInputComponent{
						CustomID:    "prize_pool",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_poll.input.prize_pool.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_poll.input.deadline.label"),
					discord.TextInputComponent{
						CustomID:    "vote_deadline",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_poll.input.deadline.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
			).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	// For race mode, show modal for race configuration
	if voteType == models.BetVoteTypeRace {
		// Encode allow_vote_change in custom ID
		customID := fmt.Sprintf("bet:config_race:%t", allowVoteChange)
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(customID).
			SetTitle(i18n.TranslateText(locale, "command.bet.modal.create_race.title")).
			SetComponents(
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_race.input.title.label"),
					discord.TextInputComponent{
						CustomID:    "title",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_race.input.title.placeholder"),
						Required:    true,
						MinLength:   ptr(1),
						MaxLength:   100,
						Value:       title,
					},
				),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_race.input.prize_pool.label"),
					discord.TextInputComponent{
						CustomID:    "prize_pool",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_race.input.prize_pool.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_race.input.entry_fee.label"),
					discord.TextInputComponent{
						CustomID:    "entry_fee",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_race.input.entry_fee.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_race.input.entry_deadline.label"),
					discord.TextInputComponent{
						CustomID:    "entry_deadline",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_race.input.entry_deadline.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_race.input.vote_deadline.label"),
					discord.TextInputComponent{
						CustomID:    "vote_deadline",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_race.input.vote_deadline.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
			).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	// For battle royale mode, show modal for battle royale configuration
	if voteType == models.BetVoteTypeBattleRoyale {
		customID := "bet:config_battle_royale"
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(customID).
			SetTitle(i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.title")).
			SetComponents(
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.title.label"),
					discord.TextInputComponent{
						CustomID:    "title",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.title.placeholder"),
						Required:    true,
						MinLength:   ptr(1),
						MaxLength:   100,
						Value:       title,
					},
				),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.entry_fee.label"),
					discord.TextInputComponent{
						CustomID:    "entry_fee",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.entry_fee.placeholder"),
						Required:    true,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.prize_pool.label"),
					discord.TextInputComponent{
						CustomID:    "prize_pool",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.prize_pool.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.entry_deadline.label"),
					discord.TextInputComponent{
						CustomID:    "entry_deadline",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.create_battle_royale.input.entry_deadline.placeholder"),
						Required:    false,
						MinLength:   ptr(1),
						MaxLength:   10,
					}),
			).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	// For other modes, show not implemented message
	if err := event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(i18n.TranslateText(locale, "command.bet.error.mode_not_implemented")).
		SetFlags(discord.MessageFlagEphemeral).
		Build()); err != nil {
		return errors.NewError(err)
	}
	return nil
}
