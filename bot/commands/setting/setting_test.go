package setting

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
)

func createSQLiteTable(db *gorm.DB, model any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}
	tableName := stmt.Schema.Table
	var columns []string
	var pks []string
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		colType := "TEXT"
		dbType := strings.ToLower(field.TagSettings["TYPE"])
		if field.DataType == "time" || dbType == "datetime" || dbType == "timestamp" {
			colType = "DATETIME"
		}
		if field.PrimaryKey {
			pks = append(pks, fmt.Sprintf("`%s`", field.DBName))
		}
		columns = append(columns, fmt.Sprintf("`%s` %s", field.DBName, colType))
	}
	var pkConstraint string
	if len(pks) > 0 {
		pkConstraint = fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ", "))
	}
	query := fmt.Sprintf("CREATE TABLE `%s` (%s%s)", tableName, strings.Join(columns, ", "), pkConstraint)
	if err := db.Exec(query).Error; err != nil {
		return err
	}

	for _, idx := range stmt.Schema.ParseIndexes() {
		var idxCols []string
		for _, f := range idx.Fields {
			idxCols = append(idxCols, fmt.Sprintf("`%s`", f.DBName))
		}
		uniqueStr := ""
		if idx.Class == "UNIQUE" {
			uniqueStr = "UNIQUE "
		}
		idxQuery := fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS `%s` ON `%s` (%s)", uniqueStr, idx.Name, tableName, strings.Join(idxCols, ", "))
		if err := db.Exec(idxQuery).Error; err != nil {
			return err
		}
	}
	return nil
}

func setupTestComponent(t *testing.T) (*components.Components, *gorm.DB) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.Guild{}, &models.BumpSchedule{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	conf := components.Config{
		BumpUserID: snowflake.ID(302050872383242240),
		BumpImage:  regexp.MustCompile(`https://disboard.org/images/bot-command-image-bump-.+\.png`),
		UpUserID:   snowflake.ID(761562078095867916),
		UpColor:    7506394,
	}

	c := components.New(context.Background(), conf, &database.DB{DB: gdb})
	return c, gdb
}

func TestBumpHandler_NormalCreatedAt(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1001)
	channelID := snowflake.ID(2001)
	messageID := snowflake.New(time.Now())

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	createdAt := time.Now().UTC().Truncate(time.Second)
	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        messageID,
				CreatedAt: createdAt,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	if err := bumpHandler(c, g, event); err != nil {
		t.Fatalf("bumpHandler returned error: %v", err)
	}

	var schedule models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).First(&schedule).Error; err != nil {
		t.Fatalf("schedule not found in DB: %v", err)
	}

	expectedRemindAt := createdAt.Add(2 * time.Hour)
	if !schedule.RemindAt.Equal(expectedRemindAt) {
		t.Errorf("expected RemindAt %v, got %v", expectedRemindAt, schedule.RemindAt)
	}
	if schedule.MessageID != messageID {
		t.Errorf("expected MessageID %v, got %v", messageID, schedule.MessageID)
	}
}

func TestBumpHandler_ZeroCreatedAt_UsesSnowflakeTime(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1002)
	channelID := snowflake.ID(2002)
	now := time.Now().UTC()
	messageID := snowflake.New(now)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	// Simulating GuildMessageUpdate where CreatedAt is zero
	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        messageID,
				CreatedAt: time.Time{}, // ZERO time
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	if err := bumpHandler(c, g, event); err != nil {
		t.Fatalf("bumpHandler returned error: %v", err)
	}

	var schedule models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).First(&schedule).Error; err != nil {
		t.Fatalf("schedule not found in DB: %v", err)
	}

	// RemindAt must NOT be in year 0001!
	if schedule.RemindAt.Year() <= 2000 {
		t.Fatalf("RemindAt year is invalid (%v), expected >= 2026", schedule.RemindAt.Year())
	}

	expectedRemindAt := messageID.Time().Add(2 * time.Hour)
	if schedule.RemindAt.Sub(expectedRemindAt).Abs() > time.Second {
		t.Errorf("expected RemindAt ~ %v, got %v", expectedRemindAt, schedule.RemindAt)
	}
}

func TestBumpHandler_Deduplication(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1003)
	channelID := snowflake.ID(2003)
	now := time.Now().UTC()
	messageID := snowflake.New(now)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        messageID,
				CreatedAt: now,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	// First execution (GuildMessageCreate)
	if err := bumpHandler(c, g, event); err != nil {
		t.Fatalf("first bumpHandler call failed: %v", err)
	}

	// Second execution with same message ID (GuildMessageUpdate)
	if err := bumpHandler(c, g, event); err != nil {
		t.Fatalf("second bumpHandler call failed: %v", err)
	}

	var count int64
	gdb.Model(&models.BumpSchedule{}).Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 schedule, got %d", count)
	}
}

func TestUpHandler_NormalAndZeroCreatedAt(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1004)
	channelID := snowflake.ID(2004)
	now := time.Now().UTC()
	messageID := snowflake.New(now)

	g := &models.Guild{
		ID:        guildID,
		UpEnabled: true,
	}
	_ = gdb.Create(g)

	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        messageID,
				CreatedAt: time.Time{}, // Zero CreatedAt
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "dissoku up",
				},
				Embeds: []discord.Embed{
					{
						Color: 7506394,
					},
				},
			},
		},
	}

	if err := upHandler(c, g, event); err != nil {
		t.Fatalf("upHandler returned error: %v", err)
	}

	var schedule models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeUp).First(&schedule).Error; err != nil {
		t.Fatalf("UP schedule not found: %v", err)
	}

	expectedRemindAt := messageID.Time().Add(1 * time.Hour)
	if schedule.RemindAt.Sub(expectedRemindAt).Abs() > time.Second {
		t.Errorf("expected RemindAt ~ %v, got %v", expectedRemindAt, schedule.RemindAt)
	}
}

func TestScheduler_ProcessesDueSchedules(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1005)
	channelID := snowflake.ID(2005)
	pastTime := time.Now().Add(-10 * time.Minute)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
		UpEnabled:   true,
	}
	_ = gdb.Create(g)

	// Due schedule (in the past)
	dueSchedule := models.BumpSchedule{
		GuildID:   guildID,
		ChannelID: channelID,
		Type:      models.BumpTypeBump,
		RemindAt:  pastTime,
		MessageID: snowflake.New(pastTime),
	}
	_ = gdb.Create(&dueSchedule)

	// Future schedule (not due yet)
	futureSchedule := models.BumpSchedule{
		GuildID:   guildID,
		ChannelID: channelID,
		Type:      models.BumpTypeUp,
		RemindAt:  time.Now().Add(45 * time.Minute),
		MessageID: snowflake.New(time.Now()),
	}
	_ = gdb.Create(&futureSchedule)

	cmd := Command(c)
	schedulers := cmd.Scheduler()
	if len(schedulers) == 0 {
		t.Fatalf("no schedulers registered")
	}

	worker := schedulers[0].Worker
	// Run worker with nil client (notice creation will log error but schedule will be deleted)
	if err := worker(c, nil); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}

	// Due schedule should be deleted
	var remainingDue models.BumpSchedule
	err := gdb.Where("id = ?", dueSchedule.ID).First(&remainingDue).Error
	if err == nil {
		t.Errorf("expected due schedule to be deleted from DB, but still exists")
	}

	// Future schedule should still exist in DB
	var remainingFuture models.BumpSchedule
	err = gdb.Where("id = ?", futureSchedule.ID).First(&remainingFuture).Error
	if err != nil {
		t.Errorf("expected future schedule to remain in DB, but got error: %v", err)
	}
}

func TestEventHandler_GuildMessageUpdate(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1006)
	channelID := snowflake.ID(2006)
	bumpUserID := snowflake.ID(302050872383242240)
	now := time.Now().UTC()
	messageID := snowflake.New(now)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	cmd := Command(c)
	eventHandler := cmd.OnEvent()
	if eventHandler == nil {
		t.Fatalf("OnEvent handler is nil")
	}

	appID := snowflake.ID(947088344167366698)

	// Step 1: Initial message (e.g. without embed yet)
	createEvent := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			MessageID: messageID,
			Message: discord.Message{
				ID:            messageID,
				ChannelID:     channelID,
				GuildID:       &guildID,
				ApplicationID: &appID,
				Author: discord.User{
					ID:  bumpUserID,
					Bot: true,
				},
				CreatedAt: now,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: nil, // no embed yet
			},
		},
	}
	_ = eventHandler(createEvent)

	// Schedule should not exist yet since embed wasn't present
	var count int64
	gdb.Model(&models.BumpSchedule{}).Where("guild_id = ?", guildID).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 schedules before embed, got %d", count)
	}

	// Step 2: Update event arrives with embed, but partial message (CreatedAt zero, author omitted)
	updateEvent := &events.GuildMessageUpdate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			MessageID: messageID,
			Message: discord.Message{
				ID:        messageID,
				ChannelID: channelID,
				GuildID:   &guildID,
				// Author and ApplicationID omitted by Discord in partial update
				CreatedAt: time.Time{}, // Zero CreatedAt
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
		OldMessage: discord.Message{
			ID:            messageID,
			ApplicationID: &appID,
			Author: discord.User{
				ID:  bumpUserID,
				Bot: true,
			},
			//nolint:staticcheck // SA1019: Interaction is deprecated
			Interaction: &discord.MessageInteraction{
				Name: "bump",
			},
			CreatedAt: now,
		},
	}

	if err := eventHandler(updateEvent); err != nil {
		t.Fatalf("eventHandler returned error on update: %v", err)
	}

	// Now the schedule should be created properly using snowflake/old message time!
	var schedule models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).First(&schedule).Error; err != nil {
		t.Fatalf("schedule not found after update event: %v", err)
	}

	if schedule.RemindAt.Year() <= 2000 {
		t.Fatalf("RemindAt year is invalid (%v), expected >= 2026", schedule.RemindAt.Year())
	}

	expectedRemindAt := now.Add(2 * time.Hour)
	if schedule.RemindAt.Sub(expectedRemindAt).Abs() > time.Second {
		t.Errorf("expected RemindAt ~ %v, got %v", expectedRemindAt, schedule.RemindAt)
	}
}

func TestBumpHandler_ReplacesExistingScheduleDifferentMessageID(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1007)
	channelID := snowflake.ID(2007)
	oldTime := time.Now().UTC().Add(-1 * time.Hour)
	oldMessageID := snowflake.New(oldTime)
	newTime := time.Now().UTC()
	newMessageID := snowflake.New(newTime)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	// Pre-existing schedule with oldMessageID
	oldSchedule := models.BumpSchedule{
		GuildID:   guildID,
		ChannelID: channelID,
		Type:      models.BumpTypeBump,
		RemindAt:  oldTime.Add(2 * time.Hour),
		MessageID: oldMessageID,
	}
	if err := gdb.Create(&oldSchedule).Error; err != nil {
		t.Fatalf("failed to create old schedule: %v", err)
	}

	// New bump event with different message ID
	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        newMessageID,
				CreatedAt: newTime,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	if err := bumpHandler(c, g, event); err != nil {
		t.Fatalf("bumpHandler failed to replace schedule: %v", err)
	}

	var schedules []models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).Find(&schedules).Error; err != nil {
		t.Fatalf("failed to query schedules: %v", err)
	}

	if len(schedules) != 1 {
		t.Fatalf("expected exactly 1 schedule, got %d", len(schedules))
	}
	if schedules[0].MessageID != newMessageID {
		t.Errorf("expected schedule message ID %v, got %v", newMessageID, schedules[0].MessageID)
	}
}

func TestScheduler_ProcessesMultipleDueSchedules(t *testing.T) {
	c, gdb := setupTestComponent(t)

	pastTime := time.Now().Add(-10 * time.Minute)

	// Create 3 guilds and 3 due schedules
	for i := 1; i <= 3; i++ {
		gid := snowflake.ID(2000 + i)
		cid := snowflake.ID(3000 + i)
		g := &models.Guild{
			ID:          gid,
			BumpEnabled: true,
		}
		_ = gdb.Create(g)

		s := models.BumpSchedule{
			GuildID:   gid,
			ChannelID: cid,
			Type:      models.BumpTypeBump,
			RemindAt:  pastTime.Add(time.Duration(i) * time.Minute),
			MessageID: snowflake.New(pastTime),
		}
		if err := gdb.Create(&s).Error; err != nil {
			t.Fatalf("failed to create schedule %d: %v", i, err)
		}
	}

	cmd := Command(c)
	schedulers := cmd.Scheduler()
	if len(schedulers) == 0 {
		t.Fatalf("no schedulers registered")
	}

	worker := schedulers[0].Worker
	if err := worker(c, nil); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}

	// All 3 schedules must be deleted in a single pass without statement accumulation errors
	var remainingCount int64
	if err := gdb.Model(&models.BumpSchedule{}).Count(&remainingCount).Error; err != nil {
		t.Fatalf("failed to count remaining schedules: %v", err)
	}
	if remainingCount != 0 {
		t.Errorf("expected 0 remaining schedules, got %d", remainingCount)
	}
}

func TestScheduler_GuildNotFound_DeletesSchedule(t *testing.T) {
	c, gdb := setupTestComponent(t)

	nonExistentGuildID := snowflake.ID(99999)
	channelID := snowflake.ID(88888)
	pastTime := time.Now().Add(-10 * time.Minute)

	s := models.BumpSchedule{
		GuildID:   nonExistentGuildID,
		ChannelID: channelID,
		Type:      models.BumpTypeBump,
		RemindAt:  pastTime,
		MessageID: snowflake.New(pastTime),
	}
	if err := gdb.Create(&s).Error; err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	cmd := Command(c)
	worker := cmd.Scheduler()[0].Worker
	if err := worker(c, nil); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}

	// Non-existent guild record should be removed (gorm.ErrRecordNotFound)
	var count int64
	gdb.Model(&models.BumpSchedule{}).Where("guild_id = ?", nonExistentGuildID).Count(&count)
	if count != 0 {
		t.Errorf("expected orphan schedule to be deleted, but %d remain", count)
	}
}

func TestBumpHandler_IgnoresStaleAndOlderMessages(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1008)
	channelID := snowflake.ID(2008)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	// 1. Stale message (3 hours ago, remindAt is 1 hour in the past)
	staleTime := time.Now().UTC().Add(-3 * time.Hour)
	staleMessageID := snowflake.New(staleTime)
	staleEvent := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        staleMessageID,
				CreatedAt: staleTime,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	if err := bumpHandler(c, g, staleEvent); err != nil {
		t.Fatalf("bumpHandler returned error for stale event: %v", err)
	}

	var count int64
	gdb.Model(&models.BumpSchedule{}).Where("guild_id = ?", guildID).Count(&count)
	if count != 0 {
		t.Fatalf("expected stale event to be ignored (0 schedules), got %d", count)
	}

	// 2. Current valid bump message (now)
	currentTime := time.Now().UTC()
	currentMessageID := snowflake.New(currentTime)
	currentEvent := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        currentMessageID,
				CreatedAt: currentTime,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}
	if err := bumpHandler(c, g, currentEvent); err != nil {
		t.Fatalf("bumpHandler failed for current event: %v", err)
	}

	// 3. Out-of-order update for an older message (30 minutes ago, still within 2h window but older than currentMessageID)
	olderTime := currentTime.Add(-30 * time.Minute)
	olderMessageID := snowflake.New(olderTime)
	olderEvent := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        olderMessageID,
				CreatedAt: olderTime,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}
	if err := bumpHandler(c, g, olderEvent); err != nil {
		t.Fatalf("bumpHandler failed for older event: %v", err)
	}

	var schedule models.BumpSchedule
	if err := gdb.Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).First(&schedule).Error; err != nil {
		t.Fatalf("failed to fetch schedule: %v", err)
	}
	if schedule.MessageID != currentMessageID {
		t.Errorf("expected schedule to keep newer MessageID %v, but was overwritten with %v", currentMessageID, schedule.MessageID)
	}
}

func TestBumpHandler_ConcurrentSameMessage(t *testing.T) {
	c, gdb := setupTestComponent(t)

	guildID := snowflake.ID(1009)
	channelID := snowflake.ID(2009)
	now := time.Now().UTC()
	messageID := snowflake.New(now)

	g := &models.Guild{
		ID:          guildID,
		BumpEnabled: true,
	}
	_ = gdb.Create(g)

	event := &events.GuildMessageCreate{
		GenericGuildMessage: &events.GenericGuildMessage{
			GuildID:   guildID,
			ChannelID: channelID,
			Message: discord.Message{
				ID:        messageID,
				CreatedAt: now,
				//nolint:staticcheck // SA1019: Interaction is deprecated
				Interaction: &discord.MessageInteraction{
					Name: "bump",
				},
				Embeds: []discord.Embed{
					{
						Image: &discord.EmbedResource{
							URL: "https://disboard.org/images/bot-command-image-bump.png",
						},
					},
				},
			},
		},
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			if err := bumpHandler(c, g, event); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent bumpHandler returned error: %v", err)
	}

	var count int64
	gdb.Model(&models.BumpSchedule{}).Where("guild_id = ? AND type = ?", guildID, models.BumpTypeBump).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 schedule after concurrent calls, got %d", count)
	}
}
