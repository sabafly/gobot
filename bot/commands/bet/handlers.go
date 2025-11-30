package bet

import (
	stderrors "errors"
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

	// Parse prize pool (optional)
	prizePoolStr, ok := event.Data.OptText("prize_pool")
	var prizePool *int64
	if ok && strings.TrimSpace(prizePoolStr) != "" {
		pool, err := strconv.ParseInt(prizePoolStr, 10, 64)
		if err != nil || pool < 0 {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_prize_pool")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		if pool > 0 {
			prizePool = &pool
		}
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
		PrizePool:           prizePool,
		VoteDeadline:        voteDeadline,
		Locale:              string(locale),
	}

	// Use a single transaction for prize pool deduction and bet host creation
	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		// Check and deduct prize pool from organizer's points
		if prizePool != nil && *prizePool > 0 {
			var gopoint models.GoPoint
			if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
				UserID:  event.User().ID,
				GuildID: *event.GuildID(),
			}).Error; err != nil {
				return err
			}

			if gopoint.Points < *prizePool {
				if err := event.RespondMessage(discord.NewMessageBuilder().
					SetContent(i18n.BuildContext().
						WithText("pool", fmt.Sprintf("%d", *prizePool)).
						WithText("points", fmt.Sprintf("%d", gopoint.Points)).
						ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_prize_pool"))).
					SetFlags(discord.MessageFlagEphemeral)); err != nil {
					return err
				}
				return nil
			}

			// Deduct prize pool from organizer
			gopoint.Points -= *prizePool
			if err := tx.Save(&gopoint).Error; err != nil {
				return err
			}
		}

		// Create bet host
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
	if err := c.GormDB().Order(
		clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
	).Where("host_id = ?", betHost.ID).Find(&optionModels).Error; err != nil {
		return errors.NewError(err)
	}

	// Create layout components
	layoutComponents, err := createBetLayout(betHost, optionModels, c.GormDB(), locale)
	if err != nil {
		slog.Error("failed to create bet layout", "error", err)
		return errors.NewError(err)
	}

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
		if result.Error != nil {
			if !stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			// No existing bet, continue
		} else {
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

		// Check if user already voted
		var existingBet models.Bet
		result := tx.Where("host_id = ? AND user_id = ?", hostID, event.User().ID).First(&existingBet)
		if result.Error != nil && !stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if result.Error == nil {
			// User already voted, check if update is allowed

			// If no change in amount or option, return early to avoid unnecessary updates
			if amount == existingBet.Amount && optionID == existingBet.OptionID {
				if err := event.CreateMessage(discord.NewMessageCreateBuilder().
					SetContent(i18n.TranslateText(locale, "command.bet.message.no_change")).
					SetFlags(discord.MessageFlagEphemeral).
					Build()); err != nil {
					return err
				}
				return nil
			}

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

			// Check if user has sufficient points for the difference
			pointsDifference := amount - existingBet.Amount
			if pointsDifference > 0 && gopoint.Points < pointsDifference {
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

			// Update the bet
			existingBet.Amount = amount
			existingBet.OptionID = optionID
			existingBet.Option = models.BetOption{ID: optionID}
			existingBet.Timestamp = time.Now().Unix()

			if err := tx.Save(&existingBet).Error; err != nil {
				return err
			}

			// Adjust points by the difference
			gopoint.Points -= pointsDifference
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

		// Check if user has sufficient points for new bet
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
	if err := db.Order(
		clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
	).Where("host_id = ?", hostID).Find(&options).Error; err != nil {
		return fmt.Errorf("failed to load options: %w", err)
	}

	layoutComponents, err := createBetLayout(&betHost, options, db, locale)
	if err != nil {
		return err
	}
	_, err = client.Rest.UpdateMessage(betHost.ChannelID, betHost.MessageID, discord.NewMessageBuilder().
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
		if err := tx.Order(
			clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
		).Where("host_id = ?", hostID).Find(&options).Error; err != nil {
			return err
		}

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

		// Get all entrants (for entry fee refund/distribution in race mode)
		var allEntrants []models.BetEntrant
		tx.Where("host_id = ?", hostID).Find(&allEntrants)

		// Calculate entry fee pool
		entryFeePool := int64(0)
		if betHost.EntryFee != nil && *betHost.EntryFee > 0 {
			entryFeePool = *betHost.EntryFee * int64(len(allEntrants))
		}

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

			// Refund entry fees to all entrants
			if betHost.EntryFee != nil && *betHost.EntryFee > 0 {
				for _, entrant := range allEntrants {
					var gopoint models.GoPoint
					tx.FirstOrCreate(&gopoint, models.GoPoint{
						UserID:  entrant.UserID,
						GuildID: betHost.GuildID,
					})

					gopoint.Points += *betHost.EntryFee
					tx.Save(&gopoint)
					totalRefunded += *betHost.EntryFee
				}
			}

			// Refund prize pool to organizer
			if betHost.PrizePool != nil && *betHost.PrizePool > 0 {
				var gopoint models.GoPoint
				tx.FirstOrCreate(&gopoint, models.GoPoint{
					UserID:  betHost.OwnerID,
					GuildID: betHost.GuildID,
				})

				gopoint.Points += *betHost.PrizePool
				tx.Save(&gopoint)
				totalRefunded += *betHost.PrizePool
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
				// Total pool includes bet pool and organizer's prize pool
				totalDistributablePool := totalPool
				if betHost.PrizePool != nil && *betHost.PrizePool > 0 {
					totalDistributablePool += *betHost.PrizePool
				}

				for _, bet := range winnersBets {
					// Calculate proportional share
					share := (bet.Amount * totalDistributablePool) / winnersTotal

					var gopoint models.GoPoint
					tx.FirstOrCreate(&gopoint, models.GoPoint{
						UserID:  bet.UserID,
						GuildID: betHost.GuildID,
					})

					gopoint.Points += share
					tx.Save(&gopoint)
				}
			} else if betHost.PrizePool != nil && *betHost.PrizePool > 0 && len(winnerIDs) > 0 {
				// No bets but prize pool exists - distribute prize pool equally to winners
				sharePerWinner := *betHost.PrizePool / int64(len(winnerIDs))
				for _, winnerID := range winnerIDs {
					// Find the entrant for this winner option
					for _, entrant := range allEntrants {
						if entrant.OptionID == winnerID {
							var gopoint models.GoPoint
							tx.FirstOrCreate(&gopoint, models.GoPoint{
								UserID:  entrant.UserID,
								GuildID: betHost.GuildID,
							})

							gopoint.Points += sharePerWinner
							tx.Save(&gopoint)
							break
						}
					}
				}
			}

			// Distribute entry fee pool to winning entrants (for race mode)
			if entryFeePool > 0 && len(winnerIDs) > 0 {
				// Find winning entrants
				winningEntrants := make([]models.BetEntrant, 0)
				for _, entrant := range allEntrants {
					for _, winnerID := range winnerIDs {
						if entrant.OptionID == winnerID {
							winningEntrants = append(winningEntrants, entrant)
							break
						}
					}
				}

				// Distribute entry fee pool equally among winning entrants
				if len(winningEntrants) > 0 {
					sharePerWinner := entryFeePool / int64(len(winningEntrants))
					for _, entrant := range winningEntrants {
						var gopoint models.GoPoint
						tx.FirstOrCreate(&gopoint, models.GoPoint{
							UserID:  entrant.UserID,
							GuildID: betHost.GuildID,
						})

						gopoint.Points += sharePerWinner
						tx.Save(&gopoint)
					}
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

			// Build result message with entry fee pool and prize pool info if applicable
			prizePoolAmount := int64(0)
			if betHost.PrizePool != nil {
				prizePoolAmount = *betHost.PrizePool
			}

			if entryFeePool > 0 || prizePoolAmount > 0 {
				resultMessage = i18n.BuildContext().
					WithText("winners", strings.Join(winnerNames, ", ")).
					WithText("total_pool", fmt.Sprintf("%d", totalPool)).
					WithText("entry_pool", fmt.Sprintf("%d", entryFeePool)).
					WithText("prize_pool", fmt.Sprintf("%d", prizePoolAmount)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.message.result_with_pools"))
			} else {
				resultMessage = i18n.BuildContext().
					WithText("winners", strings.Join(winnerNames, ", ")).
					WithText("total_pool", fmt.Sprintf("%d", totalPool)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.message.result"))
			}
		}

		// Update message
		var options []models.BetOption
		tx.Order(
			clause.OrderByColumn{Column: clause.Column{Name: "index"}, Desc: false},
		).Where("host_id = ?", hostID).Find(&options)

		layoutComponents, err := createBetLayout(&betHost, options, tx, locale)
		if err != nil {
			return err
		}

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

// handleRaceConfig handles race mode configuration
func handleRaceConfig(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
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

	// Parse prize pool (optional)
	prizePoolStr, ok := event.Data.OptText("prize_pool")
	var prizePool *int64
	if ok && strings.TrimSpace(prizePoolStr) != "" {
		pool, err := strconv.ParseInt(prizePoolStr, 10, 64)
		if err != nil || pool < 0 {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_prize_pool")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		if pool > 0 {
			prizePool = &pool
		}
	}

	// Parse entry fee (optional)
	entryFeeStr, ok := event.Data.OptText("entry_fee")
	var entryFee *int64
	if ok && strings.TrimSpace(entryFeeStr) != "" {
		fee, err := strconv.ParseInt(entryFeeStr, 10, 64)
		if err != nil || fee < 0 {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_entry_fee")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		if fee > 0 {
			entryFee = &fee
		}
	}

	// Parse entry deadline (optional)
	entryDeadlineStr, ok := event.Data.OptText("entry_deadline")
	var entryDeadline *time.Time
	if ok && strings.TrimSpace(entryDeadlineStr) != "" {
		mins, err := strconv.Atoi(entryDeadlineStr)
		if err != nil {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_entry_deadline")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		entryDeadline = ptr(time.Now().Add(time.Duration(mins) * time.Minute))
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	// Check and deduct prize pool from organizer's points
	if prizePool != nil && *prizePool > 0 {
		var gopoint models.GoPoint
		if err := c.GormDB().FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  event.User().ID,
			GuildID: *event.GuildID(),
		}).Error; err != nil {
			return errors.NewError(err)
		}

		if gopoint.Points < *prizePool {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.BuildContext().
					WithText("pool", fmt.Sprintf("%d", *prizePool)).
					WithText("points", fmt.Sprintf("%d", gopoint.Points)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_prize_pool"))).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}

		// Deduct prize pool from organizer
		gopoint.Points -= *prizePool
		if err := c.GormDB().Save(&gopoint).Error; err != nil {
			return errors.NewError(err)
		}
	}

	// Create bet host for race mode
	betHost := &models.BetHost{
		ID:                  uuid.New(),
		GuildID:             *event.GuildID(),
		ChannelID:           event.Channel().ID(),
		Title:               title,
		Mode:                string(models.BetVoteTypeRace),
		Status:              string(models.BetStatusEntry), // Start with entry phase
		OwnerID:             event.User().ID,
		AllowVoteDestChange: allowVoteChange,
		EntryFee:            entryFee,
		PrizePool:           prizePool,
		EntryDeadline:       entryDeadline,
		VoteDeadline:        nil, // Vote deadline can be set manually via start_vote button
		Locale:              string(locale),
	}

	// Save to database
	if err := c.GormDB().Create(betHost).Error; err != nil {
		slog.Error("failed to create bet host", "error", err)
		return errors.NewError(err)
	}

	// Create layout components (no options yet for race mode)
	layoutComponents, err := createBetLayout(betHost, []models.BetOption{}, c.GormDB(), locale)
	if err != nil {
		slog.Error("failed to create bet layout", "error", err)
		return errors.NewError(err)
	}

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
		SetContent(i18n.TranslateText(locale, "command.bet.message.race_created")).
		SetFlags(discord.MessageFlagEphemeral)); err != nil {
		return errors.NewError(err)
	}

	return nil
}

// handleEntryButton handles the entry button click for race mode
func handleEntryButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if entry is open
		if betHost.Status != string(models.BetStatusEntry) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.entry_closed")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if user is already entered
		var existingEntrant models.BetEntrant
		result := tx.Where("host_id = ? AND user_id = ?", hostID, event.User().ID).First(&existingEntrant)
		if result.Error != nil {
			if !stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			// No existing entrant, continue
		} else {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.already_entered")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check and deduct entry fee if set
		if betHost.EntryFee != nil && *betHost.EntryFee > 0 {
			var gopoint models.GoPoint
			if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
				UserID:  event.User().ID,
				GuildID: *event.GuildID(),
			}).Error; err != nil {
				return err
			}

			if gopoint.Points < *betHost.EntryFee {
				if err := event.CreateMessage(discord.NewMessageCreateBuilder().
					SetContent(i18n.BuildContext().
						WithText("fee", fmt.Sprintf("%d", *betHost.EntryFee)).
						WithText("points", fmt.Sprintf("%d", gopoint.Points)).
						ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_entry_fee"))).
					SetFlags(discord.MessageFlagEphemeral).
					Build()); err != nil {
					return err
				}
				return nil
			}

			// Deduct entry fee
			gopoint.Points -= *betHost.EntryFee
			if err := tx.Save(&gopoint).Error; err != nil {
				return err
			}
		}

		// Create a new option for this entrant
		optionText := event.User().EffectiveName()

		option := &models.BetOption{
			ID:         uuid.New(),
			HostID:     hostID,
			OptionText: optionText,
			Index:      0, // Will be updated
		}

		// Get current max index
		var maxIndex int
		if err := tx.Model(&models.BetOption{}).Where("host_id = ?", hostID).Select("COALESCE(MAX(\"index\"), -1)").Scan(&maxIndex).Error; err != nil {
			return err
		}
		option.Index = maxIndex + 1

		if err := tx.Create(option).Error; err != nil {
			return err
		}

		// Create entrant record
		entrant := &models.BetEntrant{
			ID:       uuid.New(),
			HostID:   hostID,
			UserID:   event.User().ID,
			OptionID: option.ID,
		}

		if err := tx.Create(entrant).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		// Build response message
		var responseMsg string
		if betHost.EntryFee != nil && *betHost.EntryFee > 0 {
			responseMsg = i18n.BuildContext().
				WithText("fee", fmt.Sprintf("%d", *betHost.EntryFee)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.message.entered_with_fee"))
		} else {
			responseMsg = i18n.TranslateText(locale, "command.bet.message.entered")
		}

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(responseMsg).
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

// handleStartVoteButton handles the start vote button click for race mode
func handleStartVoteButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
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
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_start_vote")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if still in entry phase
		if betHost.Status != string(models.BetStatusEntry) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.not_in_entry_phase")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if there are at least 2 entrants
		var entrantCount int64
		if err := tx.Model(&models.BetEntrant{}).Where("host_id = ?", hostID).Count(&entrantCount).Error; err != nil {
			return err
		}
		if entrantCount < 2 {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.min_entrants")).
				SetComponents(discord.NewActionRow(
					discord.NewDangerButton(
						i18n.TranslateText(locale, "command.bet.button.cancel"),
						fmt.Sprintf("bet:cancel_entry_btn:%s", hostID),
					),
				)).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Update status to voting
		betHost.Status = string(models.BetStatusVoting)
		if err := tx.Save(&betHost).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.message.vote_started")).
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

// handleBattleRoyaleConfig handles battle royale mode configuration
func handleBattleRoyaleConfig(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	// Defer the response to acknowledge the interaction
	if err := event.DeferCreateMessage(true); err != nil {
		return errors.NewError(err)
	}

	locale := event.Locale()

	title := event.Data.Text("title")

	// Parse entry fee (required for battle royale)
	entryFeeStr := event.Data.Text("entry_fee")
	var entryFee int64
	fee, err := strconv.ParseInt(entryFeeStr, 10, 64)
	if err != nil || fee <= 0 {
		if err := event.RespondMessage(discord.NewMessageBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_entry_fee_required")).
			SetFlags(discord.MessageFlagEphemeral)); err != nil {
			return errors.NewError(err)
		}
		return nil
	}
	entryFee = fee

	// Parse prize pool (optional)
	prizePoolStr, ok := event.Data.OptText("prize_pool")
	var prizePool *int64
	if ok && strings.TrimSpace(prizePoolStr) != "" {
		pool, err := strconv.ParseInt(prizePoolStr, 10, 64)
		if err != nil || pool < 0 {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_prize_pool")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		if pool > 0 {
			prizePool = &pool
		}
	}

	// Parse entry deadline (optional)
	entryDeadlineStr, ok := event.Data.OptText("entry_deadline")
	var entryDeadline *time.Time
	if ok && strings.TrimSpace(entryDeadlineStr) != "" {
		mins, err := strconv.Atoi(entryDeadlineStr)
		if err != nil {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.invalid_entry_deadline")).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
		entryDeadline = ptr(time.Now().Add(time.Duration(mins) * time.Minute))
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	// Check and deduct prize pool from organizer's points
	if prizePool != nil && *prizePool > 0 {
		var gopoint models.GoPoint
		if err := c.GormDB().FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  event.User().ID,
			GuildID: *event.GuildID(),
		}).Error; err != nil {
			return errors.NewError(err)
		}

		if gopoint.Points < *prizePool {
			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetContent(i18n.BuildContext().
					WithText("pool", fmt.Sprintf("%d", *prizePool)).
					WithText("points", fmt.Sprintf("%d", gopoint.Points)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_prize_pool"))).
				SetFlags(discord.MessageFlagEphemeral)); err != nil {
				return errors.NewError(err)
			}
			return nil
		}

		// Deduct prize pool from organizer
		gopoint.Points -= *prizePool
		if err := c.GormDB().Save(&gopoint).Error; err != nil {
			return errors.NewError(err)
		}
	}

	// Create bet host for battle royale mode
	betHost := &models.BetHost{
		ID:                  uuid.New(),
		GuildID:             *event.GuildID(),
		ChannelID:           event.Channel().ID(),
		Title:               title,
		Mode:                string(models.BetVoteTypeBattleRoyale),
		Status:              string(models.BetStatusEntry), // Start with entry phase
		OwnerID:             event.User().ID,
		AllowVoteDestChange: false, // Not applicable for battle royale
		EntryFee:            &entryFee,
		PrizePool:           prizePool,
		EntryDeadline:       entryDeadline,
		VoteDeadline:        nil, // No voting in battle royale
		Locale:              string(locale),
	}

	// Save to database
	if err := c.GormDB().Create(betHost).Error; err != nil {
		slog.Error("failed to create bet host", "error", err)
		return errors.NewError(err)
	}

	// Create layout components (no options yet for battle royale mode)
	layoutComponents, err := createBetLayout(betHost, []models.BetOption{}, c.GormDB(), locale)
	if err != nil {
		slog.Error("failed to create bet layout", "error", err)
		return errors.NewError(err)
	}

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
		SetContent(i18n.TranslateText(locale, "command.bet.message.battle_royale_created")).
		SetFlags(discord.MessageFlagEphemeral)); err != nil {
		return errors.NewError(err)
	}

	return nil
}

// handleBattleRoyaleEntryButton handles the entry button click for battle royale mode
func handleBattleRoyaleEntryButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if entry is open
		if betHost.Status != string(models.BetStatusEntry) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.entry_closed")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if user is already entered
		var existingEntrant models.BetEntrant
		result := tx.Where("host_id = ? AND user_id = ?", hostID, event.User().ID).First(&existingEntrant)
		if result.Error != nil {
			if !stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			// No existing entrant, continue
		} else {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.already_entered_br")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check and deduct entry fee (required for battle royale)
		if betHost.EntryFee == nil || *betHost.EntryFee <= 0 {
			return fmt.Errorf("battle royale must have entry fee")
		}

		var gopoint models.GoPoint
		if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  event.User().ID,
			GuildID: *event.GuildID(),
		}).Error; err != nil {
			return err
		}

		if gopoint.Points < *betHost.EntryFee {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.BuildContext().
					WithText("fee", fmt.Sprintf("%d", *betHost.EntryFee)).
					WithText("points", fmt.Sprintf("%d", gopoint.Points)).
					ReplaceText(i18n.TranslateText(locale, "command.bet.error.insufficient_entry_fee"))).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Deduct entry fee
		gopoint.Points -= *betHost.EntryFee
		if err := tx.Save(&gopoint).Error; err != nil {
			return err
		}

		// Create a new option for this entrant
		optionText := event.User().EffectiveName()

		option := &models.BetOption{
			ID:         uuid.New(),
			HostID:     hostID,
			OptionText: optionText,
			Index:      0, // Will be updated
		}

		// Get current max index
		var maxIndex int
		if err := tx.Model(&models.BetOption{}).Where("host_id = ?", hostID).Select("COALESCE(MAX(\"index\"), -1)").Scan(&maxIndex).Error; err != nil {
			return err
		}
		option.Index = maxIndex + 1

		if err := tx.Create(option).Error; err != nil {
			return err
		}

		// Create entrant record
		entrant := &models.BetEntrant{
			ID:       uuid.New(),
			HostID:   hostID,
			UserID:   event.User().ID,
			OptionID: option.ID,
		}

		if err := tx.Create(entrant).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		responseMsg := i18n.BuildContext().
			WithText("fee", fmt.Sprintf("%d", *betHost.EntryFee)).
			ReplaceText(i18n.TranslateText(locale, "command.bet.message.entered_br"))

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(responseMsg).
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

// handleBattleRoyaleCloseEntryButton handles the close entry button click for battle royale mode
func handleBattleRoyaleCloseEntryButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
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
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_close_entry")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if still in entry phase
		if betHost.Status != string(models.BetStatusEntry) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.not_in_entry_phase")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if there are at least 2 entrants
		var entrantCount int64
		if err := tx.Model(&models.BetEntrant{}).Where("host_id = ?", hostID).Count(&entrantCount).Error; err != nil {
			return err
		}
		if entrantCount < 2 {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.min_entrants_br")).
				SetComponents(discord.NewActionRow(
					discord.NewDangerButton(
						i18n.TranslateText(locale, "command.bet.button.cancel"),
						fmt.Sprintf("bet:cancel_entry_btn:%s", hostID),
					),
				)).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Update status to closed (ready for result decision)
		betHost.Status = string(models.BetStatusClosed)
		if err := tx.Save(&betHost).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(i18n.TranslateText(locale, "command.bet.message.entry_closed_br")).
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

// handleCancelEntryButton handles the cancel button click during entry phase
func handleCancelEntryButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()
	parts := strings.Split(event.Data.CustomID(), ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var betHost models.BetHost
		if err := tx.First(&betHost, "id = ?", hostID).Error; err != nil {
			return err
		}

		// Check if user is owner
		if !betHost.IsOwner(event.User().ID) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.only_organizer_cancel")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Check if still in entry phase
		if betHost.Status != string(models.BetStatusEntry) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(i18n.TranslateText(locale, "command.bet.error.cannot_cancel_after_entry")).
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Refund entry fees to all entrants
		var totalRefunded int64 = 0
		if betHost.EntryFee != nil && *betHost.EntryFee > 0 {
			var entrants []models.BetEntrant
			tx.Where("host_id = ?", hostID).Find(&entrants)

			for _, entrant := range entrants {
				var gopoint models.GoPoint
				tx.FirstOrCreate(&gopoint, models.GoPoint{
					UserID:  entrant.UserID,
					GuildID: betHost.GuildID,
				})

				gopoint.Points += *betHost.EntryFee
				tx.Save(&gopoint)
				totalRefunded += *betHost.EntryFee
			}
		}

		// Refund prize pool to organizer
		if betHost.PrizePool != nil && *betHost.PrizePool > 0 {
			var gopoint models.GoPoint
			tx.FirstOrCreate(&gopoint, models.GoPoint{
				UserID:  betHost.OwnerID,
				GuildID: betHost.GuildID,
			})

			gopoint.Points += *betHost.PrizePool
			tx.Save(&gopoint)
			totalRefunded += *betHost.PrizePool
		}

		// Update status to cancelled
		betHost.Status = string(models.BetStatusCancelled)
		if err := tx.Save(&betHost).Error; err != nil {
			return err
		}

		// Update the bet message
		if err := updateBetMessage(c, tx, event.Client(), hostID, locale); err != nil {
			return err
		}

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(i18n.BuildContext().
				WithText("amount", fmt.Sprintf("%d", totalRefunded)).
				ReplaceText(i18n.TranslateText(locale, "command.bet.message.cancelled"))).
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
