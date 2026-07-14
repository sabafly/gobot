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

// 21-symbol Juggler-like reel strips
var reelLeft = [21]string{
	"7️⃣", "🍇", "🔄", "🍒", "⬛", "🍇", "🤡", "🔄", "🍒", "7️⃣",
	"🍇", "🔄", "🔔", "⬛", "🍇", "🤡", "🔄", "🍇", "7️⃣", "🍇", "🔄",
}

var reelMiddle = [21]string{
	"7️⃣", "🍇", "🔄", "🤡", "⬛", "🍇", "🔔", "🔄", "7️⃣", "🍇",
	"🔄", "🤡", "⬛", "🍇", "🔄", "🍇", "7️⃣", "🍇", "🔄", "🤡", "⬛",
}

var reelRight = [21]string{
	"7️⃣", "🍇", "🔄", "🤡", "⬛", "🍇", "🔔", "🔄", "7️⃣", "🍇",
	"🔄", "🤡", "⬛", "🍇", "🔄", "🍇", "7️⃣", "🍇", "🔄", "🤡", "⬛",
}

// 5 paylines: Middle, Top, Bottom, Diagonal Down, Diagonal Up
var paylines = [5][3][2]int{
	{{1, 0}, {1, 1}, {1, 2}}, // Middle
	{{0, 0}, {0, 1}, {0, 2}}, // Top
	{{2, 0}, {2, 1}, {2, 2}}, // Bottom
	{{0, 0}, {1, 1}, {2, 2}}, // Diagonal down
	{{2, 0}, {1, 1}, {0, 2}}, // Diagonal up
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

func hasCherryWin(grid [3][3]string) bool {
	// Cherry on left reel (Reel 0)
	return grid[0][0] == "🍒" || grid[1][0] == "🍒" || grid[2][0] == "🍒"
}

func getGridForStops(i, j, k int) [3][3]string {
	return [3][3]string{
		{reelLeft[i], reelMiddle[j], reelRight[k]},
		{reelLeft[(i+1)%21], reelMiddle[(j+1)%21], reelRight[(k+1)%21]},
		{reelLeft[(i+2)%21], reelMiddle[(j+2)%21], reelRight[(k+2)%21]},
	}
}

type StopCombination struct {
	i, j, k int
}

func findMatchingReelStops(outcome string) [3]int {
	var matches []StopCombination

	for i := range 21 {
		for j := range 21 {
			for k := range 21 {
				grid := getGridForStops(i, j, k)

				// Evaluate wins
				lineWins := make([]string, 0)
				for p := range 5 {
					winSymbol := checkWinOnLine(grid, p)
					if winSymbol != "" {
						lineWins = append(lineWins, winSymbol)
					}
				}

				hasCherry := hasCherryWin(grid)

				matched := false
				switch outcome {
				case "BB":
					if len(lineWins) == 1 && lineWins[0] == "7️⃣" && !hasCherry {
						matched = true
					}
				case "RB":
					if len(lineWins) == 1 && lineWins[0] == "RB" && !hasCherry {
						matched = true
					}
				case "🔄":
					if len(lineWins) == 1 && lineWins[0] == "🔄" && !hasCherry {
						matched = true
					}
				case "🍇":
					if len(lineWins) == 1 && lineWins[0] == "🍇" && !hasCherry {
						matched = true
					}
				case "🍒":
					if len(lineWins) == 0 && hasCherry {
						matched = true
					}
				case "🤡":
					if len(lineWins) == 1 && lineWins[0] == "🤡" && !hasCherry {
						matched = true
					}
				case "🔔":
					if len(lineWins) == 1 && lineWins[0] == "🔔" && !hasCherry {
						matched = true
					}
				case "lose":
					if len(lineWins) == 0 && !hasCherry {
						matched = true
					}
				}

				if matched {
					matches = append(matches, StopCombination{i, j, k})
				}
			}
		}
	}

	if len(matches) == 0 {
		return [3]int{rand.N(21), rand.N(21), rand.N(21)}
	}

	picked := matches[rand.N(len(matches))]
	return [3]int{picked.i, picked.j, picked.k}
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

	// Clone the state by value copying the struct
	nextData := *data

	nextData.TotalSpins++
	nextData.TotalSpent += cost

	// 3. Process according to status
	switch nextData.Status {
	case SlotStatusNormal:
		r := rand.N(10000)
		if r < 38 { // BB (Big Bonus) - ~1/263
			prelit := rand.N(4) == 0 // 25% 先ペカ
			if prelit {
				nextData.GogoLit = true
				stops := findMatchingReelStops("BB")
				nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
				nextData.Status = SlotStatusBB
				nextData.BonusSpinsLeft = 24
				nextData.TotalBonusWin = 15
				nextData.LastSpinWin = 15
			} else {
				nextData.GogoLit = true
				stops := findMatchingReelStops("lose")
				nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
				nextData.Status = SlotStatusGogo
				nextData.BonusType = "BB"
				nextData.LastSpinWin = 0
			}
		} else if r < 38+28 { // RB (Regular Bonus) - ~1/357
			prelit := rand.N(4) == 0 // 25% 先ペカ
			if prelit {
				nextData.GogoLit = true
				stops := findMatchingReelStops("RB")
				nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
				nextData.Status = SlotStatusRB
				nextData.BonusSpinsLeft = 8
				nextData.TotalBonusWin = 15
				nextData.LastSpinWin = 15
			} else {
				nextData.GogoLit = true
				stops := findMatchingReelStops("lose")
				nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
				nextData.Status = SlotStatusGogo
				nextData.BonusType = "RB"
				nextData.LastSpinWin = 0
			}
		} else if r < 38+28+1370 { // Replay - ~1/7.3
			stops := findMatchingReelStops("🔄")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 3
		} else if r < 38+28+1370+1613 { // Grape - ~1/6.2
			stops := findMatchingReelStops("🍇")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 7
		} else if r < 38+28+1370+1613+303 { // Cherry - ~1/33
			stops := findMatchingReelStops("🍒")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 2
		} else if r < 38+28+1370+1613+303+10 { // Clown - ~1/1000
			stops := findMatchingReelStops("🤡")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 10
		} else if r < 38+28+1370+1613+303+10+10 { // Bell - ~1/1000
			stops := findMatchingReelStops("🔔")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 15
		} else { // Lose
			stops := findMatchingReelStops("lose")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.LastSpinWin = 0
		}
	case SlotStatusGogo:
		// Aligns 7s and starts bonus
		if nextData.BonusType == "BB" {
			stops := findMatchingReelStops("BB")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.Status = SlotStatusBB
			nextData.BonusSpinsLeft = 24
			nextData.TotalBonusWin = 15
			nextData.LastSpinWin = 15
		} else {
			stops := findMatchingReelStops("RB")
			nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
			nextData.Status = SlotStatusRB
			nextData.BonusSpinsLeft = 8
			nextData.TotalBonusWin = 15
			nextData.LastSpinWin = 15
		}
	case SlotStatusBB, SlotStatusRB:
		// Bonus Game Spin
		stops := findMatchingReelStops("🍇")
		nextData.LastReels = getGridForStops(stops[0], stops[1], stops[2])
		nextData.LastSpinWin = 15
		nextData.BonusSpinsLeft--
		nextData.TotalBonusWin += 15

		if nextData.BonusSpinsLeft == 0 {
			nextData.Status = SlotStatusNormal
			nextData.GogoLit = false
		}
	}

	// 4. Update points and state atomically
	nextData.TotalWon += nextData.LastSpinWin
	netChange := nextData.LastSpinWin - cost
	if netChange != 0 {
		if err := gopoint.AddPoint(c, data.UserID, data.GuildID, netChange); err != nil {
			return "", errors.NewError(err)
		}
	}

	*data = nextData
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
