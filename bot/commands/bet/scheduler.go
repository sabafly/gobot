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

// cacheKey generates a unique key for caching processed bets
type cacheKey struct {
	ID    uuid.UUID
	Phase string // "vote" or "entry"
}

var (
	// schedulerCache キャッシュ: GuildID -> スケジューラーで処理されたベットIDのリスト
	schedulerCache = database.NewMemoryValues[snowflake.ID, []cacheKey](time.Hour)
)

func betSchedulerWorker(c *components.Components, client *bot.Client) error {
	// トランザクション成功後にキャッシュに追加するためのローカルマップ
	// key: GuildID, value: 処理されたcacheKeyのリスト
	pendingCacheUpdates := make(map[snowflake.ID][]cacheKey)

	err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		slog.Debug("Running bet scheduler worker")
		now := time.Now()

		// Handle vote deadline
		var voteBets []models.BetHost
		if err := tx.Model(&models.BetHost{}).Where("vote_deadline IS NOT NULL AND vote_deadline <= ? AND status = ?", now, models.BetStatusVoting).Find(&voteBets).Error; err != nil {
			return err
		}
		slog.Debug("Found scheduled vote bets", "count", len(voteBets))
		for _, betHost := range voteBets {
			key := cacheKey{ID: betHost.ID, Phase: "vote"}
			// すでに処理されたベットはスキップ
			slog.Debug("Processing scheduled vote bet", "betID", betHost.ID, "guildID", betHost.GuildID)
			processedBets, ok := schedulerCache.Get(betHost.GuildID)
			if ok {
				if slices.Contains(processedBets, key) {
					slog.Debug("Skipping already processed bet", "betID", betHost.ID, "guildID", betHost.GuildID)
					continue
				}
			}

			// Close voting
			betHost.Status = string(models.BetStatusClosed)

			if err := tx.Save(&betHost).Error; err != nil {
				return err
			}
			slog.Info("Updating scheduled bet (vote deadline)", "betID", betHost.ID, "guildID", betHost.GuildID, "newStatus", betHost.Status)
			// ハンドラーを呼び出してメッセージを更新
			if err := updateBetMessage(c, tx, client, betHost.ID, discord.Locale(betHost.Locale)); err != nil {
				return err
			}

			// ローカルマップに追加（トランザクション成功後にキャッシュに反映）
			pendingCacheUpdates[betHost.GuildID] = append(pendingCacheUpdates[betHost.GuildID], key)
		}

		// Handle entry deadline (for race mode)
		var entryBets []models.BetHost
		if err := tx.Model(&models.BetHost{}).Where("entry_deadline IS NOT NULL AND entry_deadline <= ? AND status = ? AND mode = ?", now, models.BetStatusEntry, models.BetVoteTypeRace).Find(&entryBets).Error; err != nil {
			return err
		}
		slog.Debug("Found scheduled entry bets (race)", "count", len(entryBets))
		for _, betHost := range entryBets {
			key := cacheKey{ID: betHost.ID, Phase: "entry"}
			// すでに処理されたベットはスキップ
			slog.Debug("Processing scheduled entry bet", "betID", betHost.ID, "guildID", betHost.GuildID)
			processedBets, ok := schedulerCache.Get(betHost.GuildID)
			if ok {
				if slices.Contains(processedBets, key) {
					slog.Debug("Skipping already processed entry bet", "betID", betHost.ID, "guildID", betHost.GuildID)
					continue
				}
			}

			// Check if there are at least 2 entrants
			var entrantCount int64
			if err := tx.Model(&models.BetEntrant{}).Where("host_id = ?", betHost.ID).Count(&entrantCount).Error; err != nil {
				return err
			}
			if entrantCount >= 2 {
				// Move to voting phase
				betHost.Status = string(models.BetStatusVoting)
			} else {
				// Cancel if not enough entrants
				betHost.Status = string(models.BetStatusCancelled)
				// Refund entry fees
				if err := refundEntryFees(tx, &betHost); err != nil {
					return err
				}
			}

			if err := tx.Save(&betHost).Error; err != nil {
				return err
			}
			slog.Info("Updating scheduled bet (entry deadline)", "betID", betHost.ID, "guildID", betHost.GuildID, "newStatus", betHost.Status)
			// ハンドラーを呼び出してメッセージを更新
			if err := updateBetMessage(c, tx, client, betHost.ID, discord.Locale(betHost.Locale)); err != nil {
				return err
			}

			// ローカルマップに追加（トランザクション成功後にキャッシュに反映）
			pendingCacheUpdates[betHost.GuildID] = append(pendingCacheUpdates[betHost.GuildID], key)
		}

		// Handle entry deadline (for battle royale mode)
		var brEntryBets []models.BetHost
		if err := tx.Model(&models.BetHost{}).Where("entry_deadline IS NOT NULL AND entry_deadline <= ? AND status = ? AND mode = ?", now, models.BetStatusEntry, models.BetVoteTypeBattleRoyale).Find(&brEntryBets).Error; err != nil {
			return err
		}
		slog.Debug("Found scheduled entry bets (battle royale)", "count", len(brEntryBets))
		for _, betHost := range brEntryBets {
			key := cacheKey{ID: betHost.ID, Phase: "entry_br"}
			// すでに処理されたベットはスキップ
			slog.Debug("Processing scheduled entry bet (BR)", "betID", betHost.ID, "guildID", betHost.GuildID)
			processedBets, ok := schedulerCache.Get(betHost.GuildID)
			if ok {
				if slices.Contains(processedBets, key) {
					slog.Debug("Skipping already processed entry bet (BR)", "betID", betHost.ID, "guildID", betHost.GuildID)
					continue
				}
			}

			// Check if there are at least 2 entrants
			var entrantCount int64
			if err := tx.Model(&models.BetEntrant{}).Where("host_id = ?", betHost.ID).Count(&entrantCount).Error; err != nil {
				return err
			}
			if entrantCount >= 2 {
				// Move to closed phase (ready for result decision)
				betHost.Status = string(models.BetStatusClosed)
			} else {
				// Cancel if not enough entrants
				betHost.Status = string(models.BetStatusCancelled)
				// Refund entry fees
				if err := refundEntryFees(tx, &betHost); err != nil {
					return err
				}
			}

			if err := tx.Save(&betHost).Error; err != nil {
				return err
			}
			slog.Info("Updating scheduled bet (BR entry deadline)", "betID", betHost.ID, "guildID", betHost.GuildID, "newStatus", betHost.Status)
			// ハンドラーを呼び出してメッセージを更新
			if err := updateBetMessage(c, tx, client, betHost.ID, discord.Locale(betHost.Locale)); err != nil {
				return err
			}

			// ローカルマップに追加（トランザクション成功後にキャッシュに反映）
			pendingCacheUpdates[betHost.GuildID] = append(pendingCacheUpdates[betHost.GuildID], key)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// トランザクションが成功した場合のみキャッシュを更新
	for guildID, keys := range pendingCacheUpdates {
		existingKeys, _ := schedulerCache.Get(guildID)
		existingKeys = append(existingKeys, keys...)
		schedulerCache.Set(guildID, existingKeys)
	}

	return nil
}

// refundEntryFees refunds entry fees to all entrants
func refundEntryFees(tx *gorm.DB, betHost *models.BetHost) error {
	if betHost.EntryFee == nil || *betHost.EntryFee <= 0 {
		return nil
	}

	var entrants []models.BetEntrant
	if err := tx.Where("host_id = ?", betHost.ID).Find(&entrants).Error; err != nil {
		return err
	}

	for _, entrant := range entrants {
		var gopoint models.GoPoint
		if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  entrant.UserID,
			GuildID: betHost.GuildID,
		}).Error; err != nil {
			return err
		}

		gopoint.Points += *betHost.EntryFee
		if err := tx.Save(&gopoint).Error; err != nil {
			return err
		}
	}

	// Also refund prize pool to organizer
	if betHost.PrizePool != nil && *betHost.PrizePool > 0 {
		var gopoint models.GoPoint
		if err := tx.FirstOrCreate(&gopoint, models.GoPoint{
			UserID:  betHost.OwnerID,
			GuildID: betHost.GuildID,
		}).Error; err != nil {
			return err
		}

		gopoint.Points += *betHost.PrizePool
		if err := tx.Save(&gopoint).Error; err != nil {
			return err
		}
	}

	return nil
}
