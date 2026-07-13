package database

import (
	"log/slog"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/database/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDB(dsn string) (*DB, error) {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		return nil, err
	}

	// Set up the database connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// auto migrate models
	if err := db.AutoMigrate(
		&models.User{},
		&models.Guild{},
		&models.GoPoint{},
		&models.BetHost{},
		&models.BetOption{},
		&models.Bet{},
		&models.BetEntrant{},
		&models.Member{},
		&models.RolePanel{},
		&models.RolePanelEdit{},
		&models.RolePanelPlaced{},
		&models.MessagePin{},
		&models.MessageRemind{},
		&models.WordSuffix{},
		&models.FXPosition{},
		&models.FXOrder{},
		&models.ChinchiroSession{},
		&models.ChinchiroPlayer{},
		&models.PolymarketBet{},
		&models.GoPointTaxConfig{},
		&models.GoPointPendingTax{},
		&models.GoPointSeason{},
		&models.GoPointSeasonUser{},
	); err != nil {
		return nil, err
	}

	if db.Dialector.Name() == "mysql" {
		// Ensure go_point_season_users.points_earned is signed (bigint(20)) instead of unsigned
		if err := db.Exec("ALTER TABLE go_point_season_users MODIFY points_earned bigint(20) NOT NULL DEFAULT 0").Error; err != nil {
			slog.Error("failed to alter go_point_season_users points_earned to signed", "error", err)
		}
	}

	return &DB{DB: db}, nil
}

type DB struct {
	DB *gorm.DB
}

// GetOrCreateUser retrieves an existing user from the database or creates a new one if it doesn't exist.
// Parameters:
//   - db: The GORM database instance to use for the query
//   - userID: The Discord snowflake ID of the user
//
// Returns:
//   - *models.User: The user model (either existing or newly created)
//   - error: Any error encountered during the database operation
func GetOrCreateUser(db *gorm.DB, userID snowflake.ID) (*models.User, error) {
	user := &models.User{
		ID: userID,
	}
	if err := db.FirstOrCreate(user, models.User{ID: userID}).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetGuild retrieves a guild from the database by its ID.
// Parameters:
//   - db: The GORM database instance to use for the query
//   - guildID: The Discord snowflake ID of the guild
//
// Returns:
//   - *models.Guild: The guild model if found
//   - error: Any error encountered during the database operation (e.g., gorm.ErrRecordNotFound if guild doesn't exist)
func GetGuild(db *gorm.DB, guildID snowflake.ID) (*models.Guild, error) {
	var guild models.Guild
	if err := db.First(&guild, "id = ?", guildID).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}
