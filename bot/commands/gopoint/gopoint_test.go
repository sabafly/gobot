package gopoint

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

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
