package bet

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/database/models"
	"gorm.io/gorm"
)

var (
	statusEmoji = map[models.BetStatus]string{
		models.BetStatusEntry:     "📝",
		models.BetStatusVoting:    "🗳️",
		models.BetStatusClosed:    "🔒",
		models.BetStatusFinished:  "✅",
		models.BetStatusCancelled: "❌",
	}

	statusText = func(key models.BetStatus) string {
		return map[models.BetStatus]string{
			models.BetStatusEntry:     "エントリー受付中",
			models.BetStatusVoting:    "投票受付中",
			models.BetStatusClosed:    "受付終了",
			models.BetStatusFinished:  "終了",
			models.BetStatusCancelled: "キャンセル（返金済み）",
		}[key]
	}

	statusColor = map[models.BetStatus]int{
		models.BetStatusEntry:     0x3498DB, // Blue
		models.BetStatusVoting:    0xF1C40F, // Yellow
		models.BetStatusClosed:    0xE67E22, // Orange
		models.BetStatusFinished:  0x2ECC71, // Green
		models.BetStatusCancelled: 0xE74C3C, // Red
	}
)

func createBetLayout(host *models.BetHost, options []models.BetOption, db *gorm.DB) []discord.LayoutComponent {
	var layoutComponents []discord.LayoutComponent

	// Header

	headerComponent := discord.NewContainer().WithAccentColor(statusColor[models.BetStatus(host.Status)])

	// Title
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("# %s", host.Title)),
	)

	// Status
	emoji := statusEmoji[models.BetStatus(host.Status)]
	text := statusText(models.BetStatus(host.Status))
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("**状態:** %s %s", emoji, text)),
	)

	// Organizer and mode
	headerComponent = headerComponent.AddComponents(
		discord.NewTextDisplay(fmt.Sprintf("**主催者:** <@%d> | **モード:** %s", host.OwnerID, host.Mode)),
	)

	// Deadline
	if host.VoteDeadline != nil {
		headerComponent = headerComponent.AddComponents(
			discord.NewLargeSeparator(),
			discord.NewTextDisplayf("**投票締め切り:** %s (%s)", discord.FormattedTimestampMention(host.VoteDeadline.Unix(), discord.TimestampStyleShortDateShortTime), discord.FormattedTimestampMention(host.VoteDeadline.Unix(), discord.TimestampStyleRelative)),
		)
	}

	layoutComponents = append(layoutComponents, headerComponent)

	// Options with vote counts
	optionsComponent := discord.NewContainer().WithAccentColor(0x95A5A6) // Gray
	optionsComponent = optionsComponent.AddComponents(
		discord.NewTextDisplay("### **選択肢**"),
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
			db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Count(&voteCount)
			db.Model(&models.Bet{}).Where("option_id = ?", opt.ID).Select("COALESCE(SUM(amount), 0)").Scan(&amount)

			totalVotes += voteCount
			totalAmount += amount

			optionMarker := fmt.Sprintf("%d.", i+1)
			winners := host.GetWinners()
			for _, winnerID := range winners {
				if winnerID == opt.ID {
					optionMarker = "🏆"
					break
				}
			}

			text := discord.NewTextDisplayf("%s %s - %d票 (%dpt)", optionMarker, opt.OptionText, voteCount, amount)
			if host.Status == string(models.BetStatusVoting) {
				optionsComponent = optionsComponent.AddComponents(discord.NewSection(text).
					WithAccessory(discord.NewSecondaryButton(
						fmt.Sprintf("%sに投票", opt.OptionText),
						fmt.Sprintf("bet:vote_btn:%s:%s", host.ID, opt.ID),
					)),
				)
			} else {
				optionsComponent = optionsComponent.AddComponents(text)
			}
		}
		optionsComponent = optionsComponent.AddComponents(
			discord.NewLargeSeparator(),
			discord.NewTextDisplay(fmt.Sprintf("**合計:** %d票 / %dpt", totalVotes, totalAmount)),
		)
	} else {
		optionsComponent = optionsComponent.AddComponents(
			discord.NewTextDisplay("_選択肢がまだ追加されていません_"),
		)
	}
	layoutComponents = append(layoutComponents, optionsComponent)

	actionRow := discord.NewActionRow()
	switch models.BetStatus(host.Status) {
	case models.BetStatusEntry:
		// Add buttons if entry is active
		actionRow = actionRow.AddComponents(discord.NewPrimaryButton(
			"エントリーする",
			fmt.Sprintf("bet:enter_btn:%s", host.ID),
		))
	case models.BetStatusVoting:
		// Add buttons if voting is active
		actionRow = actionRow.AddComponents(discord.NewDangerButton(
			"投票を締め切る",
			fmt.Sprintf("bet:close_vote_btn:%s", host.ID),
		))
	case models.BetStatusClosed:
		// Add buttons if voting is active
		actionRow = actionRow.AddComponents(discord.NewSuccessButton(
			"結果を決定",
			fmt.Sprintf("bet:decide_btn:%s", host.ID),
		))
	}
	if len(actionRow.Components) > 0 {
		layoutComponents = append(layoutComponents, actionRow)
	}

	return layoutComponents
}
