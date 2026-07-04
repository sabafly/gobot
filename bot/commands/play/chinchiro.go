package play

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

var chinchiro_sessions = database.NewMemoryValues[uuid.UUID, *ChinchiroActiveSession](time.Minute * 15)

type ChinchiroActiveSession struct {
	ID uuid.UUID
}

func evaluateHand(dices []int) (string, int, int) {
	if len(dices) != 3 {
		return "目なし", 0, 0
	}
	sort.Ints(dices)
	d1, d2, d3 := dices[0], dices[1], dices[2]

	// Pinzoro: 1-1-1
	if d1 == 1 && d2 == 1 && d3 == 1 {
		return "ピンゾロ", 30, 5
	}

	// Zoro: 2-2-2 to 6-6-6
	if d1 == d2 && d2 == d3 {
		return fmt.Sprintf("ゾロ目 (%d)", d1), 20 + d1, 3
	}

	// Shigoro: 4-5-6
	if d1 == 4 && d2 == 5 && d3 == 6 {
		return "シゴロ", 10, 2
	}

	// Hifumi: 1-2-3
	if d1 == 1 && d2 == 2 && d3 == 3 {
		return "ヒフミ", -1, -2
	}

	// Point (目): two dice are same
	if d1 == d2 {
		return fmt.Sprintf("%dの目", d3), d3, 1
	}
	if d2 == d3 {
		return fmt.Sprintf("%dの目", d1), d1, 1
	}
	if d1 == d3 {
		return fmt.Sprintf("%dの目", d2), d2, 1
	}

	return "目なし", 0, 0
}

func getDiceStr(dices []int) string {
	runes := []string{"⚀", "⚁", "⚂", "⚃", "⚄", "⚅"}
	var sb strings.Builder
	for _, d := range dices {
		if d >= 1 && d <= 6 {
			sb.WriteString(runes[d-1] + " ")
		}
	}
	return strings.TrimSpace(sb.String())
}

func parseDices(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(p)
		if err == nil {
			res = append(res, v)
		}
	}
	return res
}

func formatDices(d []int) string {
	parts := make([]string, len(d))
	for i, v := range d {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func ChinchiroPlayCommand(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	bet := int64(10)
	if betOpt, ok := event.SlashCommandInteractionData().OptInt("bet"); ok {
		bet = int64(betOpt)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if points < bet*5 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(fmt.Sprintf("⚠️ **GoPoints不足**: 親（主催者）として開始するには、最大支払額（ベットの5倍 = %d pt）を支払えるだけのポイントが必要です。（現在: %d pt）", bet*5, points)),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	session := &models.ChinchiroSession{
		ID:        uuid.New(),
		GuildID:   *event.GuildID(),
		ChannelID: event.Channel().ID(),
		MessageID: 0,
		HostUserID: event.User().ID,
		Bet:       bet,
		State:     models.ChinchiroStateLobby,
	}

	err = c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		player := &models.ChinchiroPlayer{
			SessionID: session.ID,
			UserID:    event.User().ID,
			IsHost:    true,
		}
		return tx.Create(player).Error
	})
	if err != nil {
		return errors.NewError(err)
	}

	chinchiro_sessions.Set(session.ID, &ChinchiroActiveSession{ID: session.ID})

	var dbSession models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
		return errors.NewError(err)
	}

	msgBuilder := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(ChinchiroMessage(c, &dbSession, event.Locale())...)

	err = event.RespondMessage(msgBuilder)
	if err != nil {
		return errors.NewError(err)
	}

	// Update message ID in DB
	resp, err := event.Client().Rest.GetInteractionResponse(event.ApplicationID(), event.Token())
	if err == nil && resp != nil {
		c.GormDB().Model(&models.ChinchiroSession{}).Where("id = ?", session.ID).Update("message_id", resp.ID)
	}

	return nil
}

func ChinchiroMessage(c *components.Components, session *models.ChinchiroSession, locale discord.Locale) []discord.LayoutComponent {
	uuidStr := session.ID.String()

	if session.State == models.ChinchiroStateLobby {
		var playersSB strings.Builder
		for idx, p := range session.Players {
			roleStr := ""
			if p.IsHost {
				roleStr = " (主催者/親)"
			}
			playersSB.WriteString(fmt.Sprintf("%d. <@%s>%s\n", idx+1, p.UserID.String(), roleStr))
		}

		layoutCtx := i18n.BuildContext().
			WithText("bet", strconv.FormatInt(session.Bet, 10)).
			WithText("player_count", strconv.Itoa(len(session.Players))).
			WithText("players_list", strings.TrimSpace(playersSB.String())).
			WithText("host_mention", fmt.Sprintf("<@%s>", session.HostUserID.String())).
			WithText("uuid", uuidStr).
			WithCustomID("uuid", uuidStr)

		return layoutCtx.Translate(i18n.TranslateLayout(locale, "command.play.chinchiro.lobby"))
	}

	// For Active Game / Finished Game
	var statusSB strings.Builder
	var playersSB strings.Builder

	if session.State == models.ChinchiroStateHostRolling {
		rollsLeft := 3 - session.HostRollCount
		statusSB.WriteString(fmt.Sprintf("👑 **親 (<@%s>) のロール順です**\n残りロール回数: **%d / 3**", session.HostUserID.String(), rollsLeft))
	} else if session.State == models.ChinchiroStateKidsRolling {
		kids := make([]models.ChinchiroPlayer, 0)
		for _, p := range session.Players {
			if !p.IsHost {
				kids = append(kids, p)
			}
		}
		if session.CurrentPlayerIndex < len(kids) {
			currentKid := kids[session.CurrentPlayerIndex]
			rollsLeft := 3 - currentKid.RollCount
			statusSB.WriteString(fmt.Sprintf("子 (<@%s>) のロール順です\n残りロール回数: **%d / 3**", currentKid.UserID.String(), rollsLeft))
		}
	} else if session.State == models.ChinchiroStateFinished {
		statusSB.WriteString("🏁 **ゲーム終了！結果発表** 🏁")
	}

	// Compute results if finished
	var resultsMap map[snowflake.ID]int64
	if session.State == models.ChinchiroStateFinished {
		resultsMap = make(map[snowflake.ID]int64)
		// Check if host rolled instant result
		hostDices := parseDices(session.HostDices)
		_, _, hostMult := evaluateHand(hostDices)

		hostNet := int64(0)

		for _, p := range session.Players {
			if p.IsHost {
				continue
			}

			var kidNet int64
			if hostMult > 1 {
				// Host won instantly
				kidNet = -session.Bet
			} else if hostMult < 0 {
				// Host lost instantly
				kidNet = session.Bet * 2
			} else {
				// Normal comparison
				if p.Point > session.HostPoint {
					mult := 1
					if p.Point == 30 {
						mult = 5
					} else if p.Point > 20 {
						mult = 3
					} else if p.Point == 10 {
						mult = 2
					}
					kidNet = session.Bet * int64(mult)
				} else if p.Point < session.HostPoint {
					kidNet = -session.Bet
				} else {
					kidNet = 0
				}
			}
			resultsMap[p.UserID] = kidNet
			hostNet -= kidNet
		}
		resultsMap[session.HostUserID] = hostNet
	}

	// Host status
	hostDiceStr := "未ロール"
	if session.HostRollCount > 0 {
		hDices := parseDices(session.HostDices)
		hHand, _, _ := evaluateHand(hDices)
		hostDiceStr = fmt.Sprintf("%s [%s]", getDiceStr(hDices), hHand)
	}
	hostNetStr := ""
	if session.State == models.ChinchiroStateFinished {
		net := resultsMap[session.HostUserID]
		sign := "+"
		if net < 0 {
			sign = ""
		}
		hostNetStr = fmt.Sprintf(" (収支: **%s%d pt**)", sign, net)
	}
	playersSB.WriteString(fmt.Sprintf("👑 **親**: <@%s> -> **%s**%s\n", session.HostUserID.String(), hostDiceStr, hostNetStr))

	// Kids status
	for _, p := range session.Players {
		if p.IsHost {
			continue
		}
		kidDiceStr := "待機中"
		if p.RollCount > 0 {
			kDices := parseDices(p.Dices)
			kHand, _, _ := evaluateHand(kDices)
			kidDiceStr = fmt.Sprintf("%s [%s]", getDiceStr(kDices), kHand)
		} else {
			// Find if this kid is currently active
			kids := make([]models.ChinchiroPlayer, 0)
			for _, op := range session.Players {
				if !op.IsHost {
					kids = append(kids, op)
				}
			}
			if session.State == models.ChinchiroStateKidsRolling && session.CurrentPlayerIndex < len(kids) && kids[session.CurrentPlayerIndex].UserID == p.UserID {
				kidDiceStr = "👉 ロールしてください"
			}
		}
		kidNetStr := ""
		if session.State == models.ChinchiroStateFinished {
			net := resultsMap[p.UserID]
			sign := "+"
			if net < 0 {
				sign = ""
			}
			kidNetStr = fmt.Sprintf(" (収支: **%s%d pt**)", sign, net)
		}
		playersSB.WriteString(fmt.Sprintf("子: <@%s> -> **%s**%s\n", p.UserID.String(), kidDiceStr, kidNetStr))
	}

	layoutCtx := i18n.BuildContext().
		WithText("game_status", strings.TrimSpace(statusSB.String())).
		WithText("players_status", strings.TrimSpace(playersSB.String())).
		WithText("uuid", uuidStr).
		WithCustomID("uuid", uuidStr)

	layoutKey := "command.play.chinchiro.game"
	if session.State == models.ChinchiroStateFinished {
		layoutKey = "command.play.chinchiro.finished"
	}

	return layoutCtx.Translate(i18n.TranslateLayout(locale, layoutKey))
}

func ChinchiroJoinHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	sessionID, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}

	var session models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return respondSessionNotFound(event)
		}
		return errors.NewError(err)
	}

	if session.State != models.ChinchiroStateLobby {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ すでにゲームが開始されているか、終了しています。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	// Check if already joined
	for _, p := range session.Players {
		if p.UserID == event.User().ID {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay("⚠️ すでにこのロビーに参加しています。"),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}
	}

	// Check points
	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if points < session.Bet {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(fmt.Sprintf("⚠️ **GoPoints不足**: 参加するには %d pt 必要です。（現在: %d pt）", session.Bet, points)),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	// Join player and deduct points
	txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
		player := &models.ChinchiroPlayer{
			SessionID: session.ID,
			UserID:    event.User().ID,
			IsHost:    false,
		}
		if err := tx.Create(player).Error; err != nil {
			return err
		}
		return gopoint.AddPointTx(tx, event.User().ID, *event.GuildID(), -session.Bet)
	})
	if txErr != nil {
		return errors.NewError(txErr)
	}

	// Fetch updated session
	var dbSession models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
		return errors.NewError(err)
	}

	msgUpdate := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(ChinchiroMessage(c, &dbSession, event.Locale())...).
		BuildUpdate()

	_ = event.UpdateMessage(msgUpdate)
	return nil
}

func ChinchiroStartHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	sessionID, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}

	var session models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return respondSessionNotFound(event)
		}
		return errors.NewError(err)
	}

	if session.HostUserID != event.User().ID {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ 主催者（親）のみがゲームを開始できます。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if len(session.Players) < 2 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ ゲームを開始するには、主催者以外に最低1名以上の参加者が必要です。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	// Verify Host has enough points to cover max liability: bet * 5 * num_kids
	numKids := int64(len(session.Players) - 1)
	maxLiability := session.Bet * 5 * numKids

	points, _, err := gopoint.GetPoint(c, session.HostUserID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if points < maxLiability {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(fmt.Sprintf("⚠️ **主催者ポイント不足**: 最大支払額（%d pt）の保証金が必要です。親のポイント残高を確認してください。（現在: %d pt）", maxLiability, points)),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	// Start game: deduct Host's max liability and transition state
	txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
		session.State = models.ChinchiroStateHostRolling
		if err := tx.Save(&session).Error; err != nil {
			return err
		}
		return gopoint.AddPointTx(tx, session.HostUserID, *event.GuildID(), -maxLiability)
	})
	if txErr != nil {
		return errors.NewError(txErr)
	}

	// Fetch updated session
	var dbSession models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
		return errors.NewError(err)
	}

	msgUpdate := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(ChinchiroMessage(c, &dbSession, event.Locale())...).
		BuildUpdate()

	_ = event.UpdateMessage(msgUpdate)
	return nil
}

func ChinchiroCancelHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	sessionID, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}

	var session models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return respondSessionNotFound(event)
		}
		return errors.NewError(err)
	}

	if session.HostUserID != event.User().ID {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ 主催者のみがロビーをキャンセルできます。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if session.State != models.ChinchiroStateLobby {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ すでにゲームが開始されているためキャンセルできません。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	// Refund kids, delete session
	txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
		for _, p := range session.Players {
			if !p.IsHost {
				if err := gopoint.AddPointTx(tx, p.UserID, *event.GuildID(), session.Bet); err != nil {
					return err
				}
			}
		}
		return tx.Delete(&session).Error
	})
	if txErr != nil {
		return errors.NewError(txErr)
	}

	builder := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay("🚫 **チンチロリンゲームは主催者によってキャンセルされました。**"),
			).WithAccentColor(0x7F8C8D),
		)

	_ = event.UpdateMessage(builder.BuildUpdate())
	return nil
}

func ChinchiroRollHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	sessionID, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}

	var session models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return respondSessionNotFound(event)
		}
		return errors.NewError(err)
	}

	if session.State == models.ChinchiroStateFinished || session.State == models.ChinchiroStateLobby {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ ロールできる状態ではありません。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if session.State == models.ChinchiroStateHostRolling {
		if event.User().ID != session.HostUserID {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay("⚠️ 親のロール番です。他プレイヤーはロールできません。"),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}

		// Host rolls
		dices := []int{rand.IntN(6) + 1, rand.IntN(6) + 1, rand.IntN(6) + 1}
		session.HostRollCount++
		session.HostDices = formatDices(dices)
		_, score, mult := evaluateHand(dices)

		if score != 0 || session.HostRollCount >= 3 {
			session.HostPoint = score
			if mult != 1 && mult != 0 {
				// Instant win or instant loss for host
				// Resolve game immediately
				txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
					resolveChinchiroInstantResult(tx, &session, mult)
					return tx.Save(&session).Error
				})
				if txErr != nil {
					return errors.NewError(txErr)
				}
				chinchiro_sessions.Delete(session.ID)
			} else {
				// Normal point or Menashi (after 3 rolls)
				// Transition to Kids rolling
				session.State = models.ChinchiroStateKidsRolling
				session.CurrentPlayerIndex = 0
				if err := c.GormDB().Save(&session).Error; err != nil {
					return errors.NewError(err)
				}
			}
		} else {
			// Host rolled Menashi, has remaining roll count
			if err := c.GormDB().Save(&session).Error; err != nil {
				return errors.NewError(err)
			}
		}
	} else if session.State == models.ChinchiroStateKidsRolling {
		kids := make([]models.ChinchiroPlayer, 0)
		var activeKid *models.ChinchiroPlayer
		for i := range session.Players {
			if !session.Players[i].IsHost {
				kids = append(kids, session.Players[i])
			}
		}
		if session.CurrentPlayerIndex >= len(kids) {
			return errors.NewError(fmt.Errorf("invalid current player index"))
		}
		activeKid = &kids[session.CurrentPlayerIndex]

		if event.User().ID != activeKid.UserID {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay("⚠️ あなたのロール番ではありません。"),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}

		// Active kid rolls
		dices := []int{rand.IntN(6) + 1, rand.IntN(6) + 1, rand.IntN(6) + 1}
		activeKid.RollCount++
		activeKid.Dices = formatDices(dices)
		_, score, _ := evaluateHand(dices)

		if score != 0 || activeKid.RollCount >= 3 {
			activeKid.Point = score
			// Move to next player or finish game
			txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
				if err := tx.Save(activeKid).Error; err != nil {
					return err
				}
				if session.CurrentPlayerIndex+1 >= len(kids) {
					// Reload session and players from tx first to get updated roll points
					if err := tx.Preload("Players").Where("id = ?", session.ID).First(&session).Error; err != nil {
						return err
					}
					// Set state and resolve game
					session.State = models.ChinchiroStateFinished
					resolveChinchiroNormalResults(tx, &session)
					chinchiro_sessions.Delete(session.ID)
				} else {
					session.CurrentPlayerIndex++
				}
				return tx.Save(&session).Error
			})
			if txErr != nil {
				return errors.NewError(txErr)
			}
		} else {
			// Save active kid roll state
			if err := c.GormDB().Save(activeKid).Error; err != nil {
				return errors.NewError(err)
			}
		}
	}

	// Fetch updated session
	var dbSession models.ChinchiroSession
	if err := c.GormDB().Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
		return errors.NewError(err)
	}

	msgUpdate := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(ChinchiroMessage(c, &dbSession, event.Locale())...).
		BuildUpdate()

	_ = event.UpdateMessage(msgUpdate)
	return nil
}

func resolveChinchiroInstantResult(tx *gorm.DB, session *models.ChinchiroSession, mult int) {
	numKids := int64(len(session.Players) - 1)
	maxLiability := session.Bet * 5 * numKids

	session.State = models.ChinchiroStateFinished

	if mult > 0 {
		// Host won instantly (Pinzoro, Zoro, Shigoro)
		// Host gets kids' bets
		hostRefund := maxLiability + (session.Bet * numKids)
		_ = gopoint.AddPointTx(tx, session.HostUserID, session.GuildID, hostRefund)
		// Kids get nothing
	} else if mult < 0 {
		// Host lost instantly (Hifumi)
		// Host pays 2x to each kid
		hostRefund := maxLiability - (session.Bet * 2 * numKids)
		if hostRefund > 0 {
			_ = gopoint.AddPointTx(tx, session.HostUserID, session.GuildID, hostRefund)
		}
		// Each kid gets: their bet back + 2x bet = 3x bet total
		for _, p := range session.Players {
			if !p.IsHost {
				_ = gopoint.AddPointTx(tx, p.UserID, session.GuildID, session.Bet*3)
			}
		}
	}
}

func resolveChinchiroNormalResults(tx *gorm.DB, session *models.ChinchiroSession) {
	numKids := int64(len(session.Players) - 1)
	maxLiability := session.Bet * 5 * numKids

	hostNetChange := int64(0)

	for _, p := range session.Players {
		if p.IsHost {
			continue
		}

		// Evaluate kids' point vs host's point
		// Ranks are based on the Point value
		// E.g., Pinzoro is 30, Zoro is 22-26, Shigoro is 10, Point is 1-6, Menashi is 0, Hifumi is -1
		if p.Point > session.HostPoint {
			// Kid wins
			mult := 1
			// Payout multiplier depends on kid's hand:
			// Pinzoro (30): 5x
			// Zoro (22-26): 3x
			// Shigoro (10): 2x
			// Normal point / Menashi: 1x
			if p.Point == 30 {
				mult = 5
			} else if p.Point > 20 {
				mult = 3
			} else if p.Point == 10 {
				mult = 2
			}

			// Kid receives their bet back + (bet * mult) from host = bet * (mult + 1)
			_ = gopoint.AddPointTx(tx, p.UserID, session.GuildID, session.Bet*int64(mult+1))
			hostNetChange -= session.Bet * int64(mult)
		} else if p.Point < session.HostPoint {
			// Kid loses
			// If kid gets Hifumi (-1), kid pays 1x loss (just loses their initial bet)
			// Host collects the kid's bet (which is already deducted)
			hostNetChange += session.Bet
		} else {
			// Draw
			// Kid gets their bet back
			_ = gopoint.AddPointTx(tx, p.UserID, session.GuildID, session.Bet)
		}
	}

	hostRefund := maxLiability + hostNetChange
	if hostRefund > 0 {
		_ = gopoint.AddPointTx(tx, session.HostUserID, session.GuildID, hostRefund)
	}
}

func respondSessionNotFound(event *events.ComponentInteractionCreate) errors.Error {
	builder := discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay("⚠️ 指定されたゲームセッションが見つかりません。すでに終了しているか、キャンセルされた可能性があります。"),
			).WithAccentColor(0xE74C3C),
		).
		AddFlags(discord.MessageFlagEphemeral)
	_ = event.RespondMessage(builder)
	return nil
}
