package bet

import (
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"gorm.io/gorm"
)

var (
	// schedulerCache キャッシュ: GuildID -> スケジューラーで処理されたベットIDのリスト
	schedulerCache = database.NewMemoryValues[snowflake.ID, []uuid.UUID](time.Hour)
)

func betSchedulerWorker(c *components.Components, client *bot.Client) error {
	return c.GormDB().Transaction(func(tx *gorm.DB) error {
		var bets []models.BetHost
		if err := tx.Where("vote_deadline IS NOT NULL AND vote_deadline <= ? AND (status = ? OR status = ?)", time.Now(), models.BetStatusEntry, models.BetStatusVoting).Find(&bets).Error; err != nil {
			return err
		}
		for _, betHost := range bets {
			// すでに処理されたベットはスキップ
			processedBets, ok := schedulerCache.Get(betHost.GuildID)
			if ok {
				skipped := false
				for _, processedBetID := range processedBets {
					if processedBetID == betHost.ID {
						skipped = true
						break
					}
				}
				if skipped {
					continue
				}
			}

			// ベットの状態を更新
			if betHost.Status == string(models.BetStatusEntry) {
				betHost.Status = string(models.BetStatusVoting)
			} else if betHost.Status == string(models.BetStatusVoting) {
				betHost.Status = string(models.BetStatusClosed)
			}

			// ハンドラーを呼び出してメッセージを更新
			if err := updateBetMessage(c, tx, client, betHost.ID, discord.Locale(betHost.Locale)); err != nil {
				return err
			}
			tx.Save(&betHost)

			// キャッシュに追加
			processedBets, _ = schedulerCache.Get(betHost.GuildID)
			processedBets = append(processedBets, betHost.ID)
			schedulerCache.Set(betHost.GuildID, processedBets)
		}
		return nil
	})
}
