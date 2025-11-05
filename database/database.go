package database

import (
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
	); err != nil {
		return nil, err
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
