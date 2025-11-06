package bet

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// handlePollConfig handles poll mode configuration
func handlePollConfig(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	// Defer the response to acknowledge the interaction
	if err := event.DeferCreateMessage(true); err != nil {
		return errors.NewError(err)
	}

	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID, ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	// Parse allow_vote_change
	allowVoteChange, err := strconv.ParseBool(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	title := event.Data.Text("title")

	optionsText := event.Data.Text("options")
	options := strings.Split(optionsText, "\n")
	for i := range options {
		options[i] = strings.TrimSpace(options[i])
	}

	deadlineStr, ok := event.Data.OptText("vote_deadline")
	var voteDeadline *time.Time
	if ok && strings.TrimSpace(deadlineStr) != "" {
		mins, err := strconv.Atoi(deadlineStr)
		if err != nil {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_deadline")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		voteDeadline = ptr(time.Now().Add(time.Duration(mins) * time.Minute))
	}

	// Filter empty options
	validOptions := make([]string, 0)
	for _, opt := range options {
		if utf8.RuneCountInString(opt) > 100 {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.BuildContext().
					WithText("option", opt).
					ReplaceText(i18n.TranslateText(locale, "command.bet.error.option_too_long"))).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		if opt != "" {
			validOptions = append(validOptions, opt)
		}
	}

	if len(validOptions) < 2 {
		if err := event.RespondMessage(discord.NewMessageBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.error.min_options")).
			SetFlags(discord.MessageFlagEphemeral)); err != nil {
			return errors.NewError(err)
		}
		return nil
	}
	if len(validOptions) > 25 {
		if err := event.RespondMessage(discord.NewMessageBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.error.max_options")).
			SetFlags(discord.MessageFlagEphemeral)); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	// Create bet host
	betHost := &models.BetHost{
		ID:                  uuid.New(),
		GuildID:             *event.GuildID(),
		ChannelID:           event.Channel().ID(),
		Title:               title,
		Mode:                string(models.BetVoteTypeGuess),
		Status:              string(models.BetStatusVoting),
		OwnerID:             event.User().ID,
		AllowVoteDestChange: allowVoteChange,
		VoteDeadline:        voteDeadline,
		Locale:              string(locale),
	}

	// Save to database
	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(betHost).Error; err != nil {
			slog.Error("failed to create bet host", "error", err)
			return err
		}
		// Create options
		for i, opt := range validOptions {
			option := &models.BetOption{
				ID:         uuid.New(),
				HostID:     betHost.ID,
				OptionText: opt,
				Index:      i,
			}
			if err := tx.Create(option).Error; err != nil {
				slog.Error("failed to create bet option", "error", err)
				return err
			}
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}

	// Reload options
	var optionModels []models.BetOption
	c.GormDB().Order(
		clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
	).Where("host_id = ?", betHost.ID).Find(&optionModels)

	// Create layout components
	layoutComponents := createBetLayout(betHost, optionModels, c.GormDB(), locale)

	// Respond with the bet message using MessageBuilder with ComponentV2
	msg, err := event.Client().Rest.CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(layoutComponents...).
		BuildCreate())
	if err != nil {
		slog.Error("failed to send bet message", "error", err)
		return errors.NewError(err)
	}

	// Update message ID
	betHost.MessageID = msg.ID
	if err := c.GormDB().Save(betHost).Error; err != nil {
		slog.Error("failed to update bet message ID", "error", err)
	}

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetContent(i18n.TranslateText(locale, "command.bet.message.created")).
		SetFlags(discord.MessageFlagEphemeral)); err != nil {
		return errors.NewError(err)
	}

	return nil
}

// handleVoteButton handles clicking a vote button
func handleVoteButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 4 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	optionID, err := uuid.Parse(parts[3])
	if err != nil {
		return errors.NewError(err)
	}

	// Get bet host
	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if voting is open
		if betHost.Status != string(models.BetStatusVoting) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.voting_closed")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Get option
		var option models.BetOption
		if err := tx.First(&option, "id = ?", optionID).Error; err != nil {
			return err
		}

		var status = ""

		var existingBet models.Bet
		result := tx.Where("host_id = ? AND user_id = ?", hostID, event.User().ID).First(&existingBet)
		if result.Error == nil {
			if !betHost.AllowVoteDestChange && existingBet.OptionID != optionID {
				if err := event.RespondMessage(discord.NewMessageBuilder().
					SetContent(i18n.BuildContext().
						WithText("option", existingBet.Option.OptionText).
						WithText("amount", fmt.Sprintf("%d", existingBet.Amount)).
						ReplaceText(i18n.TranslateText(locale, "command.bet.error.already_voted_no_change"))).
					SetFlags(discord.MessageFlagEphemeral)); err != nil {
					return err
				}
				return nil
			}
			status += i18n.BuildContext().
				WithText("option", existingBet.Option.OptionText).
				WithText("amount", fmt.Sprintf("%d", existingBet.Amount)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.error.already_voted_status"))
		}

		var totalBets int64
		var totalAmount struct {
			Total int64
		}
		if err := tx.Model(&models.Bet{}).Where("host_id = ?", hostID).Count(&totalBets).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Bet{}).Where("host_id = ?", hostID).Select("SUM(amount) as total").Scan(&totalAmount).Error; err != nil {
			return err
		}

		status += i18n.BuildContext().
			WithText("votes", fmt.Sprintf("%d", totalBets)).
			WithText("total_amount", fmt.Sprintf("%d", totalAmount.Total)).
			ReplaceText(i18n.TranslateText(locale, "command.bet.message.current_stats"))

		// Show modal to enter bet amount
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(fmt.Sprintf("bet:vote:%s:%s", hostID, optionID)).
			SetTitle(i18n.BuildContext().
				WithText("option", option.OptionText).
				ReplaceText(i18n.TranslateText(locale, "command.bet.modal.vote.title"))).
			SetComponents(
				discord.NewTextDisplay(status),
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.vote.input.amount.label"),
					discord.TextInputComponent{
						CustomID:    "amount",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.vote.input.amount.placeholder"),
						Required:    true,
						MinLength:   ptr(1),
						MaxLength:   10,
						Value:       "100",
					}),
			).
			Build()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}
	return nil
}

// handleVote handles the vote submission
func handleVote(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID, ":")
	if len(parts) < 4 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	optionID, err := uuid.Parse(parts[3])
	if err != nil {
		return errors.NewError(err)
	}

	amountStr := event.Data.Text("amount")
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_amount")).
			SetFlags(discord.MessageFlagEphemeral).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {

		// Check user's gopoint balance
		var gopoint models.GoPoint
		if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  event.User().ID,
			GuildID: *event.GuildID(),
		}).Error; err != nil {
			return err
		}

		if gopoint.Points < amount {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.BuildContext().
					WithText("points", fmt.Sprintf("%d", gopoint.Points)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_points"))).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if user already voted
		var existingBet models.Bet
		result := tx.Where("host_id = ? AND user_id = ?", hostID, event.User().ID).First(&existingBet)
		if result.Error == nil {
			// User already voted, check if update is allowed

			// Check if amount decrease is attempted (always prohibited)
			if amount < existingBet.Amount {
				if err := event.CreateMessage(discord.NewMessageCreateBuilder().
					SetContent(i18n.TranslateText(locale, "command.bet.error.cannot_decrease")).
					SetFlags(discord.MessageFlagEphemeral).
					Build()); err != nil {
					return err
				}
				return nil
			}

			// Check if vote destination change is attempted
			if existingBet.OptionID != optionID {
				// Get bet host to check if vote destination change is allowed
				var betHost models.BetHost
				if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
					return err
				}

				if !betHost.AllowVoteDestChange {
					if err := event.CreateMessage(discord.NewMessageCreateBuilder().
						SetContent(i18n.TranslateText(locale, "command.bet.error.cannot_change_dest")).
						SetFlags(discord.MessageFlagEphemeral).
						Build()); err != nil {
						return err
					}
					return nil
				}
			}

			// Update the bet
			oldAmount := existingBet.Amount
			existingBet.Amount = amount
			existingBet.OptionID = optionID
			existingBet.Option = models.BetOption{ID: optionID}
			existingBet.Timestamp = time.Now().Unix()

			if err := tx.Save(&existingBet).Error; err != nil {
				return err
			}

			// Adjust points
			diff := amount - oldAmount
			gopoint.Points -= diff
			if err := tx.Save(&gopoint).Error; err != nil {
				return err
			}

			if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
				return err
			}

			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.BuildContext().
					WithText("amount", fmt.Sprintf("%d", amount)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.message.updated"))).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Create new bet
		bet := &models.Bet{
			ID:        uuid.New(),
			HostID:    hostID,
			UserID:    event.User().ID,
			OptionID:  optionID,
			Amount:    amount,
			Timestamp: time.Now().Unix(),
		}

		if err := tx.Create(bet).Error; err != nil {
			return err
		}

		// Deduct points
		gopoint.Points -= amount
		if err := tx.Save(&gopoint).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(i18n.BuildContext().
				WithText("amount", fmt.Sprintf("%d", amount)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.message.voted"))).
			SetFlags(discord.MessageFlagEphemeral).
			Build()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func updateBetMessage(c *components.Components, db *gorm.DB, client *bot.Client, hostID uuid.UUID, locale discord.Locale) error {
	var betHost models.BetHost
	if err := db.First(&betHost, "id = ?", hostID).Error; err != nil {
		return err
	}

	var options []models.BetOption
	db.Order(
		clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
	).Where("host_id = ?", hostID).Find(&options)

	layoutComponents := createBetLayout(&betHost, options, db, locale)
	_, err := client.Rest.UpdateMessage(betHost.ChannelID, betHost.MessageID, discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(layoutComponents...).
		BuildUpdate())
	return err
}

// handleDecideButton handles the decide result button
func handleDecideButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if user is owner
		if !betHost.IsOwner(event.User().ID) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_decide")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Get options
		var options []models.BetOption
		tx.Order(
			clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
		).Where("host_id = ?", hostID).Find(&options)

		// Create select menu with options plus a cancellation option
		selectOptions := make([]discord.StringSelectMenuOption, 0, len(options)+1)
		// Add cancellation option first
		selectOptions = append(selectOptions, discord.StringSelectMenuOption{
			Label:       i18n.TranslateText(locale, "command.bet.cancel_option.label"),
			Value:       "cancel",
			Description: i18n.TranslateText(locale, "command.bet.cancel_option.description"),
			Emoji:       &discord.ComponentEmoji{Name: "❌"},
		})
		for _, opt := range options {
			selectOptions = append(selectOptions, discord.StringSelectMenuOption{
				Label: opt.OptionText,
				Value: opt.ID.String(),
			})
		}

		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(fmt.Sprintf("bet:decide:%s", hostID)).
			SetTitle(i18n.TranslateText(locale, "command.bet.modal.decide.title")).
			SetComponents(
				discord.NewLabel(i18n.TranslateText(locale, "command.bet.modal.decide.input.winner.label"),
					discord.StringSelectMenuComponent{
						CustomID:    "winner",
						Placeholder: i18n.TranslateText(locale, "command.bet.modal.decide.input.winner.placeholder"),
						Options:     selectOptions,
						MinValues:   ptr(1),
						MaxValues:   len(selectOptions),
					}),
			).
			Build()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}
	return nil
}

// handleDecideResult handles the result decision
func handleDecideResult(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID, ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	selectedValues := event.Data.StringValues("winner")
	if len(selectedValues) == 0 {
		return errors.NewError(fmt.Errorf("no winner selected"))
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	// Check if cancellation was selected
	isCancelled := false
	winnerIDs := make([]uuid.UUID, 0)
	for _, val := range selectedValues {
		if val == "cancel" {
			isCancelled = true
			break
		}
		if id, err := uuid.Parse(val); err == nil {
			winnerIDs = append(winnerIDs, id)
		}
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Double check ownership
		if !betHost.IsOwner(event.User().ID) {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_decide")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return err
			}
			return nil
		}
		if err := event.DeferCreateMessage(false); err != nil {
			return errors.NewError(err)
		}

		// Get all bets
		var allBets []models.Bet
		tx.Where("host_id = ?", hostID).Find(&allBets)

		var resultMessage string

		if isCancelled {
			// Cancellation: refund all bets
			totalRefunded := int64(0)
			for _, bet := range allBets {
				var gopoint models.GoPoint
				tx.FirstOrCreate(&gopoint, models.GoPoint{
					UserID:  bet.UserID,
					GuildID: betHost.GuildID,
				})

				gopoint.Points += bet.Amount
				tx.Save(&gopoint)
				totalRefunded += bet.Amount
			}

			// Update bet host status
			betHost.Status = string(models.BetStatusCancelled)
			betHost.Winners = ""
			tx.Save(&betHost)

			resultMessage = i18n.BuildContext().
				WithText("amount", fmt.Sprintf("%d", totalRefunded)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.message.cancelled"))
		} else {
			// Normal win: distribute to winners
			// Calculate total pool
			totalPool := int64(0)
			for _, bet := range allBets {
				totalPool += bet.Amount
			}

			// Get winners' bets
			winnersBets := make([]models.Bet, 0)
			for _, bet := range allBets {
				for _, winnerID := range winnerIDs {
					if bet.OptionID == winnerID {
						winnersBets = append(winnersBets, bet)
						break
					}
				}
			}

			winnersTotal := int64(0)
			for _, bet := range winnersBets {
				winnersTotal += bet.Amount
			}

			// Distribute winnings proportionally
			if winnersTotal > 0 {
				for _, bet := range winnersBets {
					// Calculate proportional share
					share := (bet.Amount * totalPool) / winnersTotal

					var gopoint models.GoPoint
					tx.FirstOrCreate(&gopoint, models.GoPoint{
						UserID:  bet.UserID,
						GuildID: betHost.GuildID,
					})

					gopoint.Points += share
					tx.Save(&gopoint)
				}
			}

			// Update bet host status
			betHost.Status = string(models.BetStatusFinished)
			betHost.SetWinners(winnerIDs)
			tx.Save(&betHost)

			// Get winner option names
			var winnerOptions []models.BetOption
			tx.Where("id IN ?", winnerIDs).Find(&winnerOptions)
			winnerNames := make([]string, len(winnerOptions))
			for i, opt := range winnerOptions {
				winnerNames[i] = opt.OptionText
			}

			resultMessage = i18n.BuildContext().
				WithText("winners", strings.Join(winnerNames, ", ")).
				WithText("total_pool", fmt.Sprintf("%d", totalPool)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.message.result"))
		}

		// Update message
		var options []models.BetOption
		tx.Order(
			clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
		).Where("host_id = ?", hostID).Find(&options)

		layoutComponents := createBetLayout(&betHost, options, tx, locale)

		// Update message with ComponentV2
		_, _ = event.Client().Rest.UpdateMessage(betHost.ChannelID, betHost.MessageID, discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(layoutComponents...).
			BuildUpdate())

		if err := event.RespondMessage(discord.NewMessageBuilder().
			SetContent(resultMessage)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func handleCloseVoteButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if user is owner
		if !betHost.IsOwner(event.User().ID) {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_close")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return err
			}
			return nil
		}

		// Update status to closed
		betHost.Status = string(models.BetStatusClosed)
		if err := tx.Save(&betHost).Error; err != nil {
			return err
		}
		// Update bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		if err := event.RespondMessage(discord.NewMessageBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.message.closed")).
			SetFlags(discord.MessageFlagEphemeral)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.NewError(err)
	}
	return nil
}
