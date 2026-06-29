package play

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/errors"
)

type SlotStatus string

const (
	SlotStatusNormal SlotStatus = "normal"
	SlotStatusGogo   SlotStatus = "gogo" // GoGo Lamp is lit, waiting to align 7s
	SlotStatusBB     SlotStatus = "bb"   // In Big Bonus
	SlotStatusRB     SlotStatus = "rb"   // In Regular Bonus
)

var (
	slot_values = database.NewMemoryValues[uuid.UUID, *SlotData](time.Minute * 15)
)

type SlotData struct {
	ID             uuid.UUID
	UserID         snowflake.ID
	GuildID        snowflake.ID
	Status         SlotStatus
	BonusType      string       // "BB" or "RB" (relevant in Gogo state)
	BonusSpinsLeft int          // spins left in current bonus (24 for BB, 8 for RB)
	TotalBonusWin  int64        // total points won during this bonus session
	LastReels      [3][3]string // 3x3 grid of symbols
	GogoLit        bool         // whether the GoGo lamp is currently lit
	LastSpinWin    int64        // points won in the last spin
	TotalSpins     int          // stats: total spins
	TotalSpent     int64        // stats: total points spent
	TotalWon       int64        // stats: total points won
}

func (s *SlotData) OnDelete() error {
	return nil
}

// 5 paylines: Middle, Top, Bottom, Diagonal Down, Diagonal Up
var paylines = [5][3][2]int{
	{{1, 0}, {1, 1}, {1, 2}}, // Middle
	{{0, 0}, {0, 1}, {0, 2}}, // Top
	{{2, 0}, {2, 1}, {2, 2}}, // Bottom
	{{0, 0}, {1, 1}, {2, 2}}, // Diagonal down
	{{2, 0}, {1, 1}, {0, 2}}, // Diagonal up
}

func getRandomSymbol(excludeCherry bool) string {
	symbols := []string{"🍇", "🔔", "🤡", "🔄", "7️⃣", "⬛"}
	if !excludeCherry {
		symbols = append(symbols, "🍒")
	}
	return symbols[rand.N(len(symbols))]
}

func checkWinOnLine(grid [3][3]string, lineIdx int) string {
	coords := paylines[lineIdx]
	s0 := grid[coords[0][0]][coords[0][1]]
	s1 := grid[coords[1][0]][coords[1][1]]
	s2 := grid[coords[2][0]][coords[2][1]]

	if s0 == s1 && s1 == s2 {
		return s0
	}
	if s0 == "7️⃣" && s1 == "7️⃣" && s2 == "⬛" {
		return "RB"
	}
	return ""
}

func hasAnyLineWin(grid [3][3]string) bool {
	for i := 0; i < 5; i++ {
		if checkWinOnLine(grid, i) != "" {
			return true
		}
	}
	return false
}

func hasCherryWin(grid [3][3]string) bool {
	// Cherry on left reel (Reel 0)
	return grid[0][0] == "🍒" || grid[1][0] == "🍒" || grid[2][0] == "🍒"
}

func generateLosingGrid() [3][3]string {
	var grid [3][3]string
	for {
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				grid[r][c] = getRandomSymbol(c == 0)
			}
		}
		if !hasAnyLineWin(grid) && !hasCherryWin(grid) {
			return grid
		}
	}
}

func generateLineWinGrid(symbol string) [3][3]string {
	var grid [3][3]string
	for {
		grid = generateLosingGrid()
		lineIdx := rand.N(5)
		coords := paylines[lineIdx]
		grid[coords[0][0]][coords[0][1]] = symbol
		grid[coords[1][0]][coords[1][1]] = symbol
		grid[coords[2][0]][coords[2][1]] = symbol

		winCount := 0
		for i := 0; i < 5; i++ {
			if checkWinOnLine(grid, i) != "" {
				winCount++
			}
		}
		if winCount == 1 && !hasCherryWin(grid) {
			return grid
		}
	}
}

func generateRBGrid() [3][3]string {
	var grid [3][3]string
	for {
		grid = generateLosingGrid()
		lineIdx := rand.N(5)
		coords := paylines[lineIdx]
		grid[coords[0][0]][coords[0][1]] = "7️⃣"
		grid[coords[1][0]][coords[1][1]] = "7️⃣"
		grid[coords[2][0]][coords[2][1]] = "⬛"

		winCount := 0
		for i := 0; i < 5; i++ {
			if checkWinOnLine(grid, i) == "RB" {
				winCount++
			} else if checkWinOnLine(grid, i) != "" {
				winCount += 2 // Invalidate
			}
		}
		if winCount == 1 && !hasCherryWin(grid) {
			return grid
		}
	}
}

func generateCherryWinGrid() [3][3]string {
	var grid [3][3]string
	for {
		grid = generateLosingGrid()
		row := rand.N(3)
		grid[row][0] = "🍒"

		if !hasAnyLineWin(grid) && hasCherryWin(grid) {
			return grid
		}
	}
}

func SlotPrecondition(event *events.ComponentInteractionCreate) (*SlotData, errors.Error) {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return nil, errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	id := uuid.MustParse(args[2])
	data, ok := slot_values.Get(id)
	if !ok {
		if err := errors.ErrorMessage("error.play.slot.expired", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	if data.UserID != event.User().ID {
		if err := errors.ErrorMessage("error.play.slot.not_your_game", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	return data, nil
}

func SlotPlay(c *components.Components, data *SlotData) (string, errors.Error) {
	var cost int64
	if data.Status == SlotStatusBB || data.Status == SlotStatusRB {
		cost = 1
	} else {
		cost = 3
	}

	// 1. Get current points
	userPoint, _, err := gopoint.GetPoint(c, data.UserID, data.GuildID)
	if err != nil {
		return "", errors.NewError(err)
	}

	if userPoint < cost {
		return "insufficient_points", nil
	}

	// 2. Deduct cost
	if err := gopoint.AddPoint(c, data.UserID, data.GuildID, -cost); err != nil {
		return "", errors.NewError(err)
	}

	data.TotalSpins++
	data.TotalSpent += cost

	// 3. Process according to status
	if data.Status == SlotStatusNormal {
		r := rand.N(10000)
		if r < 38 { // BB (Big Bonus) - ~1/263
			prelit := rand.N(4) == 0 // 25% 先ペカ
			if prelit {
				data.GogoLit = true
				data.LastReels = generateLineWinGrid("7️⃣")
				data.Status = SlotStatusBB
				data.BonusSpinsLeft = 24
				data.TotalBonusWin = 15
				data.LastSpinWin = 15
			} else {
				data.GogoLit = true
				data.LastReels = generateLosingGrid()
				data.Status = SlotStatusGogo
				data.BonusType = "BB"
				data.LastSpinWin = 0
			}
		} else if r < 38+28 { // RB (Regular Bonus) - ~1/357
			prelit := rand.N(4) == 0 // 25% 先ペカ
			if prelit {
				data.GogoLit = true
				data.LastReels = generateRBGrid()
				data.Status = SlotStatusRB
				data.BonusSpinsLeft = 8
				data.TotalBonusWin = 15
				data.LastSpinWin = 15
			} else {
				data.GogoLit = true
				data.LastReels = generateLosingGrid()
				data.Status = SlotStatusGogo
				data.BonusType = "RB"
				data.LastSpinWin = 0
			}
		} else if r < 38+28+1370 { // Replay - ~1/7.3
			data.LastReels = generateLineWinGrid("🔄")
			data.LastSpinWin = 3
		} else if r < 38+28+1370+1613 { // Grape - ~1/6.2
			data.LastReels = generateLineWinGrid("🍇")
			data.LastSpinWin = 7
		} else if r < 38+28+1370+1613+303 { // Cherry - ~1/33
			data.LastReels = generateCherryWinGrid()
			data.LastSpinWin = 2
		} else if r < 38+28+1370+1613+303+10 { // Clown - ~1/1000
			data.LastReels = generateLineWinGrid("🤡")
			data.LastSpinWin = 10
		} else if r < 38+28+1370+1613+303+10+10 { // Bell - ~1/1000
			data.LastReels = generateLineWinGrid("🔔")
			data.LastSpinWin = 15
		} else { // Lose
			data.LastReels = generateLosingGrid()
			data.LastSpinWin = 0
		}
	} else if data.Status == SlotStatusGogo {
		// Aligns 7s and starts bonus
		if data.BonusType == "BB" {
			data.LastReels = generateLineWinGrid("7️⃣")
			data.Status = SlotStatusBB
			data.BonusSpinsLeft = 24
			data.TotalBonusWin = 15
			data.LastSpinWin = 15
		} else {
			data.LastReels = generateRBGrid()
			data.Status = SlotStatusRB
			data.BonusSpinsLeft = 8
			data.TotalBonusWin = 15
			data.LastSpinWin = 15
		}
	} else if data.Status == SlotStatusBB || data.Status == SlotStatusRB {
		// Bonus Game Spin
		data.LastReels = generateLineWinGrid("🍇")
		data.LastSpinWin = 15
		data.BonusSpinsLeft--
		data.TotalBonusWin += 15

		if data.BonusSpinsLeft == 0 {
			data.Status = SlotStatusNormal
			data.GogoLit = false
		}
	}

	// 4. Add winning points
	if data.LastSpinWin > 0 {
		if err := gopoint.AddPoint(c, data.UserID, data.GuildID, data.LastSpinWin); err != nil {
			return "", errors.NewError(err)
		}
		data.TotalWon += data.LastSpinWin
	}

	return "", nil
}

func SlotFinish(c *components.Components, data *SlotData, event *events.ComponentInteractionCreate) errors.Error {
	slot_values.Delete(data.ID)

	container := discord.NewContainer().WithAccentColor(0x95A5A6) // Gray

	netWin := data.TotalWon - data.TotalSpent
	netWinSign := ""
	if netWin > 0 {
		netWinSign = "+"
	}

	container = container.AddComponents(
		discord.NewTextDisplay("🎰 **SLOT MACHINE CLOSED** 🎰"),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay("ゲームを終了しました。"),
		discord.NewTextDisplayf("- **総回転数**： %d 回転", data.TotalSpins),
		discord.NewTextDisplayf("- **使用ポイント**： `%d pt`", data.TotalSpent),
		discord.NewTextDisplayf("- **獲得ポイント**： `%d pt`", data.TotalWon),
		discord.NewTextDisplayf("- **収支**： `%s%d pt`", netWinSign, netWin),
	)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(container).
		BuildUpdate()); err != nil {
		return errors.NewError(err)
	}

	return nil
}

func SlotMessage(c *components.Components, data *SlotData, locale discord.Locale) []discord.LayoutComponent {
	var layoutComponents []discord.LayoutComponent

	container := discord.NewContainer().WithAccentColor(0x9B59B6) // Purple

	// Title
	container = container.AddComponents(
		discord.NewTextDisplay("🎰 **JUGGLER SLOT MACHINE** 🎰"),
		discord.NewLargeSeparator(),
	)

	// Reels
	row0 := fmt.Sprintf("│  %s  │  %s  │  %s  │", data.LastReels[0][0], data.LastReels[0][1], data.LastReels[0][2])
	row1 := fmt.Sprintf("│  %s  │  %s  │  %s  │", data.LastReels[1][0], data.LastReels[1][1], data.LastReels[1][2])
	row2 := fmt.Sprintf("│  %s  │  %s  │  %s  │", data.LastReels[2][0], data.LastReels[2][1], data.LastReels[2][2])

	container = container.AddComponents(
		discord.NewTextDisplay("┌─────────────────┐"),
		discord.NewTextDisplay(row0),
		discord.NewTextDisplay(row1),
		discord.NewTextDisplay(row2),
		discord.NewTextDisplay("└─────────────────┘"),
		discord.NewLargeSeparator(),
	)

	// GoGo Lamp
	gogoStr := "⚫"
	if data.GogoLit {
		gogoStr = "🟣"
	}
	container = container.AddComponents(
		discord.NewTextDisplay(gogoStr),
		discord.NewLargeSeparator(),
	)

	// Status Info
	var statusText string
	switch data.Status {
	case SlotStatusNormal:
		statusText = "- **モード**： 通常モード (コスト: 3pt)"
	case SlotStatusGogo:
		statusText = "- **モード**： 🟣 GOGO! CHANCE 🟣 (コスト: 3pt)"
	case SlotStatusBB:
		statusText = fmt.Sprintf("- **モード**： 🎉 BIG BONUS 🎉 (残り %d 回) (コスト: 1pt)", data.BonusSpinsLeft)
	case SlotStatusRB:
		statusText = fmt.Sprintf("- **モード**： 🎊 REGULAR BONUS 🎊 (残り %d 回) (コスト: 1pt)", data.BonusSpinsLeft)
	}

	container = container.AddComponents(
		discord.NewTextDisplay(statusText),
	)

	// Payout Info
	payoutText := fmt.Sprintf("- **今回の獲得**： `%d pt`", data.LastSpinWin)
	if data.Status == SlotStatusBB || data.Status == SlotStatusRB || data.BonusSpinsLeft > 0 || data.TotalBonusWin > 0 {
		payoutText += fmt.Sprintf(" | **ボーナス合計**： `%d pt`", data.TotalBonusWin)
	}
	container = container.AddComponents(
		discord.NewTextDisplay(payoutText),
	)

	layoutComponents = append(layoutComponents, container)

	// Action Row with Buttons
	actionRow := discord.NewActionRow()
	switch data.Status {
	case SlotStatusBB, SlotStatusRB:
		actionRow = actionRow.AddComponents(
			discord.NewSuccessButton("ボーナススピン (1pt)", fmt.Sprintf("play:slot_spin:%s", data.ID)),
		)
	case SlotStatusGogo:
		actionRow = actionRow.AddComponents(
			discord.NewSuccessButton("狙う！ (3pt)", fmt.Sprintf("play:slot_spin:%s", data.ID)),
		)
	default:
		actionRow = actionRow.AddComponents(
			discord.NewPrimaryButton("スピン (3pt)", fmt.Sprintf("play:slot_spin:%s", data.ID)),
		)
	}

	actionRow = actionRow.AddComponents(
		discord.NewDangerButton("清算/終了", fmt.Sprintf("play:slot_quit:%s", data.ID)),
	)

	layoutComponents = append(layoutComponents, actionRow)

	return layoutComponents
}
