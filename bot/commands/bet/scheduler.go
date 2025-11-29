package bet

import (
	"log/slog"
	"slices"
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
		slog.Debug("Running bet scheduler worker")
		var bets []models.BetHost
		if err := tx.Model(&models.BetHost{}).Where("vote_deadline IS NOT NULL AND vote_deadline <= NOW() AND (status = ? OR status = ?)", models.BetStatusEntry, models.BetStatusVoting).Find(&bets).Error; err != nil {
			return err
		}
		slog.Debug("Found scheduled bets", "count", len(bets))
		for _, betHost := range bets {
			// すでに処理されたベットはスキップ
			slog.Debug("Processing scheduled bet", "betID", betHost.ID, "guildID", betHost.GuildID)
			processedBets, ok := schedulerCache.Get(betHost.GuildID)
			if ok {
				if slices.Contains(processedBets, betHost.ID) {
					slog.Debug("Skipping already processed bet", "betID", betHost.ID, "guildID", betHost.GuildID)
					continue
				}
			}

			// ベットの状態を更新
			if betHost.Status == string(models.BetStatusEntry) {
				betHost.Status = string(models.BetStatusVoting)
			} else if betHost.Status == string(models.BetStatusVoting) {
				betHost.Status = string(models.BetStatusClosed)
			}

			if err := tx.Save(&betHost).Error; err != nil {
				return err
			}
			slog.Info("Updating scheduled bet", "betID", betHost.ID, "guildID", betHost.GuildID, "newStatus", betHost.Status)
			// ハンドラーを呼び出してメッセージを更新
			if err := updateBetMessage(c, tx, client, betHost.ID, discord.Locale(betHost.Locale)); err != nil {
				return err
			}

			// キャッシュに追加
			processedBets, _ = schedulerCache.Get(betHost.GuildID)
			processedBets = append(processedBets, betHost.ID)
			schedulerCache.Set(betHost.GuildID, processedBets)
		}
		return nil
	})
}
