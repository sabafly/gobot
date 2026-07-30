package bet

import (
	"fmt"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/commands/currency"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/i18n"
)

var (
	statusEmoji = map[models.BetStatus]string{
		models.BetStatusEntry:     "📝",
		models.BetStatusVoting:    "🗳️",
		models.BetStatusClosed:    "🔒",
		models.BetStatusFinished:  "✅",
		models.BetStatusCancelled: "❌",
	}

	statusColor = map[models.BetStatus]int{
		models.BetStatusEntry:     0x3498DB, // Blue
		models.BetStatusVoting:    0xF1C40F, // Yellow
		models.BetStatusClosed:    0xE67E22, // Orange
		models.BetStatusFinished:  0x2ECC71, // Green
		models.BetStatusCancelled: 0xE74C3C, // Red
	}
)

func statusText(key models.BetStatus, locale discord.Locale) string {
	return map[models.BetStatus]string{
		models.BetStatusEntry:     i18n.TranslateText(locale, "command.bet.status.entry"),
		models.BetStatusVoting:    i18n.TranslateText(locale, "command.bet.status.voting"),
		models.BetStatusClosed:    i18n.TranslateText(locale, "command.bet.status.closed"),
		models.BetStatusFinished:  i18n.TranslateText(locale, "command.bet.status.finished"),
		models.BetStatusCancelled: i18n.TranslateText(locale, "command.bet.status.cancelled"),
	}[key]
}

func createBetLayout(c *components.Components, host *models.BetHost, options []models.BetOption, db *gorm.DB, locale discord.Locale) ([]discord.LayoutComponent, error) {
	var layoutComponents []discord.LayoutComponent
	cName := currency.GetCurrencyName(c, host.GuildID)

	// Header

	headerComponent := discord.NewContainer().WithAccentColor(statusColor[models.BetStatus(host.Status)])

	// Title
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("# %s", host.Title)),
	)

	// Status
	emoji := statusEmoji[models.BetStatus(host.Status)]
	text := statusText(models.BetStatus(host.Status), locale)
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("%s %s %s", i18n.TranslateText(locale, "command.bet.layout.status_label"), emoji, text)),
	)

	// Organizer and mode
	modeText := modeDisplayText(host.Mode, locale)
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("%s <@%d> | %s %s", i18n.TranslateText(locale, "command.bet.layout.organizer_label"), host.OwnerID, i18n.TranslateText(locale, "command.bet.layout.mode_label"), modeText)),
	)

	// Entry Fee (for race mode with entry fee)
	if host.EntryFee != nil && *host.EntryFee > 0 {
		headerComponent = headerComponent.AddComponents(
			discord.NewTextDisplayf("%s %d %s", i18n.TranslateText(locale, "command.bet.layout.entry_fee_label"), *host.EntryFee, cName),
		)
	}

	// Prize Pool (organizer-contributed)
	if host.PrizePool != nil && *host.PrizePool > 0 {
		headerComponent = headerComponent.AddComponents(
			discord.NewTextDisplayf("%s %d %s", i18n.TranslateText(locale, "command.bet.layout.prize_pool_label"), *host.PrizePool, cName),
		)
	}

	// Entry Deadline (for race mode)
	if (host.Mode == string(models.BetVoteTypeRace) || host.Mode == string(models.BetVoteTypeBattleRoyale)) && host.EntryDeadline != nil && host.Status == string(models.BetStatusEntry) {
		headerComponent = headerComponent.AddComponents(
			discord.NewLargeSeparator(),
			discord.NewTextDisplayf("%s %s (%s)", i18n.TranslateText(locale, "command.bet.layout.entry_deadline_label"), discord.FormattedTimestampMention(host.EntryDeadline.Unix(), discord.TimestampStyleShortDateShortTime), discord.FormattedTimestampMention(host.EntryDeadline.Unix(), discord.TimestampStyleRelative)),
		)
	}

	// Vote Deadline
	if host.VoteDeadline != nil && (host.Status == string(models.BetStatusVoting) || (host.Mode == string(models.BetVoteTypeRace) && host.Status == string(models.BetStatusEntry))) {
		headerComponent = headerComponent.AddComponents(
			discord.NewLargeSeparator(),
			discord.NewTextDisplayf("%s %s (%s)", i18n.TranslateText(locale, "command.bet.layout.deadline_label"), discord.FormattedTimestampMention(host.VoteDeadline.Unix(), discord.TimestampStyleShortDateShortTime), discord.FormattedTimestampMention(host.VoteDeadline.Unix(), discord.TimestampStyleRelative)),
		)
	}

	layoutComponents = append(layoutComponents, headerComponent)

	// Options with vote counts (or entrants for race/battle royale mode)
	optionsComponent := discord.NewContainer().WithAccentColor(0x95A5A6) // Gray

	if (host.Mode == string(models.BetVoteTypeRace) || host.Mode == string(models.BetVoteTypeBattleRoyale)) && host.Status == string(models.BetStatusEntry) {
		// Race/Battle Royale mode in entry phase - show entrants
		optionsComponent = optionsComponent.AddComponents(
			discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.entrants_title")),
			discord.NewLargeSeparator(),
		)
		if len(options) > 0 {
			for i, opt := range options {
				if i > 0 {
					optionsComponent = optionsComponent.AddComponents(discord.NewSmallSeparator())
				}

				// Get entrant user info
				var entrant models.BetEntrant
				if err := db.Where("option_id = ?", opt.ID).First(&entrant).Error; err != nil {
					return nil, fmt.Errorf("failed to get entrant: %w", err)
				}

				optionMarker := fmt.Sprintf("%d.", i+1)
				text := discord.NewTextDisplay(fmt.Sprintf("%s <@%d>", optionMarker, entrant.UserID))
				optionsComponent = optionsComponent.AddComponents(text)
			}
			optionsComponent = optionsComponent.AddComponents(
				discord.NewLargeSeparator(),
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.total_entrants_label")+" "+
					fmt.Sprintf("%d", len(options))),
			)
			// Show entry fee pool if applicable
			if host.EntryFee != nil && *host.EntryFee > 0 {
				entryPool := *host.EntryFee * int64(len(options))
				optionsComponent = optionsComponent.AddComponents(
					discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.entry_pool_label") + " " +
						fmt.Sprintf("%dpt", entryPool)),
				)
			}
		} else {
			optionsComponent = optionsComponent.AddComponents(
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.no_entrants")),
			)
		}
	} else if host.Mode == string(models.BetVoteTypeBattleRoyale) && (host.Status == string(models.BetStatusClosed) || host.Status == string(models.BetStatusFinished) || host.Status == string(models.BetStatusCancelled)) {
		// Battle Royale mode after entry phase - show entrants with winners
		optionsComponent = optionsComponent.AddComponents(
			discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.entrants_title")),
			discord.NewLargeSeparator(),
		)
		if len(options) > 0 {
			for i, opt := range options {
				if i > 0 {
					optionsComponent = optionsComponent.AddComponents(discord.NewSmallSeparator())
				}

				// Get entrant user info
				var entrant models.BetEntrant
				if err := db.Where("option_id = ?", opt.ID).First(&entrant).Error; err != nil {
					return nil, fmt.Errorf("failed to get entrant: %w", err)
				}

				optionMarker := fmt.Sprintf("%d.", i+1)
				winners := host.GetWinners()
				if slices.Contains(winners, opt.ID) {
					optionMarker = "🏆"
				}

				text := discord.NewTextDisplay(fmt.Sprintf("%s <@%d>", optionMarker, entrant.UserID))
				optionsComponent = optionsComponent.AddComponents(text)
			}
			optionsComponent = optionsComponent.AddComponents(
				discord.NewLargeSeparator(),
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.total_entrants_label")+" "+
					fmt.Sprintf("%d", len(options))),
			)
			// Show entry fee pool
			if host.EntryFee != nil && *host.EntryFee > 0 {
				entryPool := *host.EntryFee * int64(len(options))
				optionsComponent = optionsComponent.AddComponents(
					discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.entry_pool_label") + " " +
						fmt.Sprintf("%dpt", entryPool)),
				)
			}
		} else {
			optionsComponent = optionsComponent.AddComponents(
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.no_entrants")),
			)
		}
	} else {
		// Normal poll mode or race mode in voting phase - show options with vote counts
		optionsComponent = optionsComponent.AddComponents(
			discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.options_title")),
			discord.NewLargeSeparator(),
		)
		if len(options) > 0 {
			totalVotes := int64(0)
			totalAmount := int64(0)

			for i, opt := range options {
				if i > 0 {
					optionsComponent = optionsComponent.AddComponents(discord.NewSmallSeparator())
				}

				var voteCount int64
				var amount int64
				if err := db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Count(&voteCount).Error; err != nil {
					return nil, fmt.Errorf("failed to count votes: %w", err)
				}
				if err := db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Select("COALESCE(SUM(amount), 0)").Scan(&amount).Error; err != nil {
					return nil, fmt.Errorf("failed to sum amounts: %w", err)
				}

				totalVotes += voteCount
				totalAmount += amount

				optionMarker := fmt.Sprintf("%d.", i+1)
				winners := host.GetWinners()
				if slices.Contains(winners, opt.ID) {
					optionMarker = "🏆"
				}

				// For race mode, show user mention instead of option text
				var displayText string
				if host.Mode == string(models.BetVoteTypeRace) {
					var entrant models.BetEntrant
					if err := db.Where("option_id = ?", opt.ID).First(&entrant).Error; err != nil {
						return nil, fmt.Errorf("failed to get entrant for race mode: %w", err)
					}
					displayText = fmt.Sprintf("<@%d>", entrant.UserID)
				} else {
					displayText = opt.OptionText
				}

				text := discord.NewTextDisplay(fmt.Sprintf("%s %s - ", optionMarker, displayText) +
					i18n.BuildContext().
						WithText("currency_name", cName).
						WithText("currency", cName).
						WithText("votes", fmt.Sprintf("%d", voteCount)).
						WithText("points", fmt.Sprintf("%d", amount)).
						ReplaceText(i18n.TranslateText(locale, "command.bet.layout.option_votes")))
				if host.Status == string(models.BetStatusVoting) {
					optionsComponent = optionsComponent.AddComponents(discord.NewSection(text).
						WithAccessory(discord.NewSecondaryButton(
							i18n.BuildContext().
								WithText("option", opt.OptionText).
								ReplaceText(i18n.TranslateText(locale, "command.bet.button.vote")),
							fmt.Sprintf("bet:vote_btn:%s:%s", host.ID, opt.ID),
						)),
					)
				} else {
					optionsComponent = optionsComponent.AddComponents(text)
				}
			}
			optionsComponent = optionsComponent.AddComponents(
				discord.NewLargeSeparator(),
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.total_label")+" "+
					i18n.BuildContext().
						WithText("currency_name", cName).
						WithText("currency", cName).
						WithText("votes", fmt.Sprintf("%d", totalVotes)).
						WithText("points", fmt.Sprintf("%d", totalAmount)).
						ReplaceText(i18n.TranslateText(locale, "command.bet.layout.votes_points"))),
			)
		} else {
			optionsComponent = optionsComponent.AddComponents(
				discord.NewTextDisplay(i18n.TranslateText(locale, "command.bet.layout.no_options")),
			)
		}
	}
	layoutComponents = append(layoutComponents, optionsComponent)

	actionRow := discord.NewActionRow()
	switch models.BetStatus(host.Status) {
	case models.BetStatusEntry:
		// Add buttons if entry is active (for race mode)
		if host.Mode == string(models.BetVoteTypeRace) {
			actionRow = actionRow.AddComponents(discord.NewPrimaryButton(
				i18n.TranslateText(locale, "command.bet.button.entry"),
				fmt.Sprintf("bet:entry_btn:%s", host.ID),
			))
			actionRow = actionRow.AddComponents(discord.NewSuccessButton(
				i18n.TranslateText(locale, "command.bet.button.start_vote"),
				fmt.Sprintf("bet:start_vote_btn:%s", host.ID),
			))
		}
		// Add buttons if entry is active (for battle royale mode)
		if host.Mode == string(models.BetVoteTypeBattleRoyale) {
			actionRow = actionRow.AddComponents(discord.NewPrimaryButton(
				i18n.TranslateText(locale, "command.bet.button.entry_br"),
				fmt.Sprintf("bet:br_entry_btn:%s", host.ID),
			))
			actionRow = actionRow.AddComponents(discord.NewSuccessButton(
				i18n.TranslateText(locale, "command.bet.button.close_entry"),
				fmt.Sprintf("bet:br_close_entry_btn:%s", host.ID),
			))
		}
	case models.BetStatusVoting:
		// Add buttons if voting is active
		actionRow = actionRow.AddComponents(discord.NewDangerButton(
			i18n.TranslateText(locale, "command.bet.button.close_vote"),
			fmt.Sprintf("bet:close_vote_btn:%s", host.ID),
		))
	case models.BetStatusClosed:
		// Add buttons if voting is active
		actionRow = actionRow.AddComponents(discord.NewSuccessButton(
			i18n.TranslateText(locale, "command.bet.button.decide_result"),
			fmt.Sprintf("bet:decide_btn:%s", host.ID),
		))
	}
	if len(actionRow.Components) > 0 {
		layoutComponents = append(layoutComponents, actionRow)
	}

	return layoutComponents, nil
}

func modeDisplayText(mode string, locale discord.Locale) string {
	switch mode {
	case string(models.BetVoteTypeGuess):
		return i18n.TranslateText(locale, "command.bet.vote_type.guess")
	case string(models.BetVoteTypeRace):
		return i18n.TranslateText(locale, "command.bet.vote_type.race")
	case string(models.BetVoteTypeBattleRoyale):
		return i18n.TranslateText(locale, "command.bet.vote_type.battle_royale")
	default:
		return mode
	}
}
