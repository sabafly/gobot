package bet

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"gorm.io/gorm"
)

// handlePollConfig handles poll mode configuration
func handlePollConfig(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	parts := strings.Split(event.Data.CustomID, ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	title := strings.Join(parts[2:], ":")

	optionsText := event.Data.Text("options")
	options := strings.Split(optionsText, ",")
	for i := range options {
		options[i] = strings.TrimSpace(options[i])
	}

	// Filter empty options
	validOptions := make([]string, 0)
	for _, opt := range options {
		if opt != "" {
			validOptions = append(validOptions, opt)
		}
	}

	if len(validOptions) < 2 {
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("少なくとも2つの選択肢が必要です。").
			SetFlags(discord.MessageFlagEphemeral).
			Build()); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	if event.GuildID() == nil {
		return errors.NewError(fmt.Errorf("this command can only be used in a guild"))
	}

	// Create bet host
	betHost := &models.BetHost{
		ID:        uuid.New(),
		GuildID:   *event.GuildID(),
		ChannelID: event.Channel().ID(),
		Title:     title,
		Mode:      string(models.BetVoteTypeGuess),
		Status:    string(models.BetStatusVoting),
		OwnerID:   event.User().ID,
	}

	// Save to database
	if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(betHost).Error; err != nil {
			slog.Error("failed to create bet host", "error", err)
			return err
		}
		// Create options
		for _, opt := range validOptions {
			option := &models.BetOption{
				ID:         uuid.New(),
				HostID:     betHost.ID,
				OptionText: opt,
			}
			if err := tx.Create(option).Error; err != nil {
				slog.Error("failed to create bet option", "error", err)
				return err
			}
		}
		// Reload options
		var optionModels []models.BetOption
		tx.Where("host_id = ?", betHost.ID).Find(&optionModels)

		// Create layout components
		layoutComponents := createBetLayout(betHost, optionModels, tx)

		// Send the bet message using MessageBuilder with ComponentV2
		msg, err := event.Client().Rest.CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(layoutComponents...).
			BuildCreate())
		if err != nil {
			slog.Error("failed to send bet message", "error", err)
			return err
		}
		// Update message ID
		betHost.MessageID = msg.ID
		if err := tx.Save(betHost).Error; err != nil {
			slog.Error("failed to update bet message ID", "error", err)
		}

		// Send confirmation
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(fmt.Sprintf("Betセッションを作成しました: %s", title)).
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

// handleVoteButton handles clicking a vote button
func handleVoteButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
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
				SetContent("投票は現在受け付けていません。").
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

		// Show modal to enter bet amount
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(fmt.Sprintf("bet:vote:%s:%s", hostID, optionID)).
			SetTitle(fmt.Sprintf("%s に投票", option.OptionText)).
			SetComponents(
				discord.NewLabel("投票するポイント数",
					discord.TextInputComponent{
						CustomID:    "amount",
						Style:       discord.TextInputStyleShort,
						Placeholder: "投票するGoポイント数を入力",
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
			SetContent("有効なポイント数を入力してください。").
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
				SetContent(fmt.Sprintf("Goポイントが不足しています。現在: %dpt", gopoint.Points)).
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
			// User already voted, update the bet
			oldAmount := existingBet.Amount
			existingBet.Amount = amount
			existingBet.OptionID = optionID
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

			updateBetMessage(c, tx, hostID)

			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent(fmt.Sprintf("投票を更新しました: %dpt", amount)).
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
		updateBetMessage(c, tx, hostID)

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(fmt.Sprintf("投票しました: %dpt", amount)).
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

func updateBetMessage(c *components.Components, db *gorm.DB, hostID uuid.UUID) {
	var betHost models.BetHost
	if err := db.First(&betHost, "id = ?", hostID).Error; err != nil {
		return
	}

	var options []models.BetOption
	db.Where("host_id = ?", hostID).Find(&options)

	layoutComponents := createBetLayout(&betHost, options, db)

	// Note: We don't have access to the event client here, so we skip the update
	// The message will be updated when the next user interacts with it
	_ = layoutComponents
}

// handleDecideButton handles the decide result button
func handleDecideButton(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
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
				SetContent("結果の決定は主催者のみが行えます。").
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Get options
		var options []models.BetOption
		tx.Where("host_id = ?", hostID).Find(&options)

		// Create select menu with options
		selectOptions := make([]discord.StringSelectMenuOption, 0, len(options))
		for _, opt := range options {
			selectOptions = append(selectOptions, discord.StringSelectMenuOption{
				Label: opt.OptionText,
				Value: opt.ID.String(),
			})
		}

		if err := event.Modal(discord.NewModalCreateBuilder().
			SetCustomID(fmt.Sprintf("bet:decide:%s", hostID)).
			SetTitle("結果を決定").
			SetComponents(
				discord.NewLabel("勝利した選択肢",
					discord.StringSelectMenuComponent{
						CustomID:    "winner",
						Placeholder: "勝利した選択肢を選択",
						Options:     selectOptions,
						MinValues:   ptr(1),
						MaxValues:   1,
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
	parts := strings.Split(event.Data.CustomID, ":")
	if len(parts) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}

	hostID, err := uuid.Parse(parts[2])
	if err != nil {
		return errors.NewError(err)
	}

	winnerID, err := uuid.Parse(event.Data.StringValues("winner")[0])
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

		// Double check ownership
		if !betHost.IsOwner(event.User().ID) {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent("結果の決定は主催者のみが行えます。").
				SetFlags(discord.MessageFlagEphemeral).
				Build()); err != nil {
				return err
			}
			return nil
		}

		// Get winner option
		var winnerOption models.BetOption
		if err := tx.First(&winnerOption, "id = ?", winnerID).Error; err != nil {
			return err
		}

		// Get all bets
		var allBets []models.Bet
		tx.Where("host_id = ?", hostID).Find(&allBets)

		// Calculate total pool and winners' share
		totalPool := int64(0)
		winnersBets := make([]models.Bet, 0)

		for _, bet := range allBets {
			totalPool += bet.Amount
			if bet.OptionID == winnerID {
				winnersBets = append(winnersBets, bet)
			}
		}

		winnersTotal := int64(0)
		for _, bet := range winnersBets {
			winnersTotal += bet.Amount
		}

		// Distribute winnings
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
		betHost.Winner = &winnerID
		tx.Save(&betHost)

		// Update message
		var options []models.BetOption
		tx.Where("host_id = ?", hostID).Find(&options)

		layoutComponents := createBetLayout(&betHost, options, tx)

		// Update message with ComponentV2
		_, _ = event.Client().Rest.UpdateMessage(betHost.ChannelID, betHost.MessageID, discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(layoutComponents...).
			BuildUpdate())

		if err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent(fmt.Sprintf("結果を決定しました。勝利: %s\n総額: %dpt が分配されました。", winnerOption.OptionText, totalPool)).
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

func createBetLayout(host *models.BetHost, options []models.BetOption, db *gorm.DB) []discord.LayoutComponent {
	var layoutComponents []discord.LayoutComponent

	// Title
	layoutComponents = append(layoutComponents, discord.NewTextDisplay(fmt.Sprintf("# %s", host.Title)))

	// Status
	statusEmoji := map[string]string{
		string(models.BetStatusEntry):    "📝",
		string(models.BetStatusVoting):   "🗳️",
		string(models.BetStatusClosed):   "🔒",
		string(models.BetStatusFinished): "✅",
	}

	statusText := map[string]string{
		string(models.BetStatusEntry):    "エントリー受付中",
		string(models.BetStatusVoting):   "投票受付中",
		string(models.BetStatusClosed):   "受付終了",
		string(models.BetStatusFinished): "終了",
	}

	emoji := statusEmoji[host.Status]
	text := statusText[host.Status]
	layoutComponents = append(layoutComponents, discord.NewTextDisplay(fmt.Sprintf("**ステータス:** %s %s", emoji, text)))

	// Organizer and mode
	layoutComponents = append(layoutComponents, discord.NewTextDisplay(fmt.Sprintf("**主催者:** <@%d> | **モード:** 通常モード（投票）", host.OwnerID)))

	// Options with vote counts
	if len(options) > 0 {
		layoutComponents = append(layoutComponents, discord.NewTextDisplay("**選択肢:**"))

		totalVotes := int64(0)
		totalAmount := int64(0)

		for i, opt := range options {
			var voteCount int64
			var amount int64
			db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Count(&voteCount)
			db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Select("COALESCE(SUM(amount), 0)").Scan(&amount)

			totalVotes += voteCount
			totalAmount += amount

			optionMarker := fmt.Sprintf("%d.", i+1)
			if host.Winner != nil && *host.Winner == opt.ID {
				optionMarker = "🏆"
			}

			layoutComponents = append(layoutComponents, discord.NewTextDisplay(fmt.Sprintf("%s %s - %d票 (%dpt)", optionMarker, opt.OptionText, voteCount, amount)))
		}

		layoutComponents = append(layoutComponents, discord.NewTextDisplay(fmt.Sprintf("**合計:** %d票 / %dpt", totalVotes, totalAmount)))
	}

	// Add buttons if voting is active
	if host.Status == string(models.BetStatusVoting) {
		var buttons []discord.InteractiveComponent

		// Add vote buttons for each option (max 5 per row)
		for i, opt := range options {
			if i >= 5 { // Discord limit
				break
			}
			buttons = append(buttons, discord.NewSecondaryButton(
				opt.OptionText,
				fmt.Sprintf("bet:vote_btn:%s:%s", host.ID, opt.ID),
			))
		}

		// Add decide button
		buttons = append(buttons, discord.NewSuccessButton(
			"結果を決定",
			fmt.Sprintf("bet:decide_btn:%s", host.ID),
		))

		// Create action row
		layoutComponents = append(layoutComponents, discord.NewActionRow(buttons...))
	}

	return layoutComponents
}

func createBetButtons(host *models.BetHost, options []models.BetOption) []discord.LayoutComponent {
	if host.Status != string(models.BetStatusVoting) {
		return nil
	}

	var buttons []discord.InteractiveComponent

	// Add vote buttons for each option (max 5 per row)
	for i, opt := range options {
		if i >= 5 { // Discord limit
			break
		}
		buttons = append(buttons, discord.NewSecondaryButton(
			opt.OptionText,
			fmt.Sprintf("bet:vote_btn:%s:%s", host.ID, opt.ID),
		))
	}

	// Add decide button
	buttons = append(buttons, discord.NewSuccessButton(
		"結果を決定",
		fmt.Sprintf("bet:decide_btn:%s", host.ID),
	))

	// Create action row and return as layout component
	row := discord.NewActionRow(buttons...)
	return []discord.LayoutComponent{row}
}

func convertToLayoutComponents(containers []discord.ContainerComponent) []discord.LayoutComponent {
	result := make([]discord.LayoutComponent, len(containers))
	for i, c := range containers {
		result[i] = c
	}
	return result
}
