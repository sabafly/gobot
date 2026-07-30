package gopoint

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
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
		// If data type is time, or GORM tag defines it as datetime/timestamp, use DATETIME in SQLite
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
	return db.Exec(query).Error
}

func TestAddPointTx_SeasonPoints(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.GoPointSeason{}, &models.GoPointSeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	userID := snowflake.ID(1234)
	guildID := snowflake.ID(5678)

	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})

	// Create an active season
	activeSeason := models.GoPointSeason{
		ID:        uuid.New(),
		GuildID:   guildID,
		Name:      "Test Season",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
		IsActive:  true,
		Criteria:  "earned",
	}
	if err := gdb.Create(&activeSeason).Error; err != nil {
		t.Fatalf("failed to create active season: %v", err)
	}

	// 1. Add positive points (increment)
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return AddPointTx(tx, userID, guildID, 100)
	})
	if err != nil {
		t.Fatalf("failed to AddPointTx with positive points: %v", err)
	}

	// Verify points in GoPoint and GoPointSeasonUser
	var gp models.GoPoint
	if err := gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error; err != nil {
		t.Fatalf("failed to get GoPoint: %v", err)
	}
	if gp.Points != 100 {
		t.Errorf("expected GoPoint points to be 100, got %d", gp.Points)
	}

	var su models.GoPointSeasonUser
	if err := gdb.Where("season_id = ? AND user_id = ?", activeSeason.ID, userID).First(&su).Error; err != nil {
		t.Fatalf("failed to get GoPointSeasonUser: %v", err)
	}
	if su.PointsEarned != 100 {
		t.Errorf("expected PointsEarned to be 100, got %d", su.PointsEarned)
	}

	// 2. Pay negative points (deduction)
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return AddPointTx(tx, userID, guildID, -30)
	})
	if err != nil {
		t.Fatalf("failed to AddPointTx with negative points: %v", err)
	}

	// Verify points in GoPoint (should be 70) and GoPointSeasonUser (should be 70: 100 - 30)
	if err := gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error; err != nil {
		t.Fatalf("failed to get GoPoint: %v", err)
	}
	if gp.Points != 70 {
		t.Errorf("expected GoPoint points to be 70, got %d", gp.Points)
	}

	if err := gdb.Where("season_id = ? AND user_id = ?", activeSeason.ID, userID).First(&su).Error; err != nil {
		t.Fatalf("failed to get GoPointSeasonUser: %v", err)
	}
	if su.PointsEarned != 70 {
		t.Errorf("expected PointsEarned to be 70, got %d", su.PointsEarned)
	}
}

func TestProcessBackgroundTasks_ExpiredScheduledSeason(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.GoPointSeason{}, &models.GoPointSeasonUser{}, &models.GoPointTaxConfig{}, &models.GoPointPendingTax{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	ctx := context.Background()
	dbWrapper := &database.DB{DB: gdb}
	c := components.New(ctx, components.Config{}, dbWrapper)

	guildID := snowflake.ID(5678)
	_ = gdb.Create(&models.Guild{ID: guildID})

	// Create a scheduled season that expired before activation
	expiredSeason := models.GoPointSeason{
		ID:         uuid.New(),
		GuildID:    guildID,
		Name:       "Expired Scheduled Season",
		StartTime:  time.Now().Add(-2 * time.Hour),
		EndTime:    time.Now().Add(-1 * time.Hour),
		IsActive:   false,
		HasAwarded: false,
		Criteria:   "earned",
	}
	if err := gdb.Create(&expiredSeason).Error; err != nil {
		t.Fatalf("failed to create expired scheduled season: %v", err)
	}

	if err := ProcessBackgroundTasks(c, nil); err != nil {
		t.Fatalf("ProcessBackgroundTasks failed: %v", err)
	}

	var s models.GoPointSeason
	if err := gdb.Where("id = ?", expiredSeason.ID).First(&s).Error; err != nil {
		t.Fatalf("failed to fetch season: %v", err)
	}
	if s.IsActive {
		t.Errorf("expected IsActive to be false, got true")
	}
	if !s.HasAwarded {
		t.Errorf("expected HasAwarded to be true for expired scheduled season, got false")
	}
}

func TestProcessBackgroundTasks_StartScheduledSeason_WithActiveSeason(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.GoPointSeason{}, &models.GoPointSeasonUser{}, &models.GoPointTaxConfig{}, &models.GoPointPendingTax{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	ctx := context.Background()
	dbWrapper := &database.DB{DB: gdb}
	c := components.New(ctx, components.Config{}, dbWrapper)

	guildID := snowflake.ID(5678)
	_ = gdb.Create(&models.Guild{ID: guildID})

	// Create an existing active season
	oldSeason := models.GoPointSeason{
		ID:         uuid.New(),
		GuildID:    guildID,
		Name:       "Old Active Season",
		StartTime:  time.Now().Add(-2 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		IsActive:   true,
		HasAwarded: false,
		Criteria:   "earned",
	}
	_ = gdb.Create(&oldSeason)

	// Create a new scheduled season ready to start
	newScheduled := models.GoPointSeason{
		ID:         uuid.New(),
		GuildID:    guildID,
		Name:       "New Scheduled Season",
		StartTime:  time.Now().Add(-10 * time.Minute),
		EndTime:    time.Now().Add(2 * time.Hour),
		IsActive:   false,
		HasAwarded: false,
		Criteria:   "earned",
	}
	_ = gdb.Create(&newScheduled)

	if err := ProcessBackgroundTasks(c, nil); err != nil {
		t.Fatalf("ProcessBackgroundTasks failed: %v", err)
	}

	var fetchedOld models.GoPointSeason
	if err := gdb.Where("id = ?", oldSeason.ID).First(&fetchedOld).Error; err != nil {
		t.Fatalf("failed to fetch old season: %v", err)
	}
	if fetchedOld.IsActive {
		t.Errorf("expected old season IsActive to be false, got true")
	}
	if !fetchedOld.HasAwarded {
		t.Errorf("expected old season HasAwarded to be true after being replaced, got false")
	}

	var fetchedNew models.GoPointSeason
	if err := gdb.Where("id = ?", newScheduled.ID).First(&fetchedNew).Error; err != nil {
		t.Fatalf("failed to fetch new season: %v", err)
	}
	if !fetchedNew.IsActive {
		t.Errorf("expected new season IsActive to be true, got false")
	}
}
