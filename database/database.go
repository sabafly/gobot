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

func GetOrCreateUser(db *gorm.DB, userID snowflake.ID) (*models.User, error) {
	user := &models.User{
		ID: userID,
	}
	if err := db.FirstOrCreate(user, models.User{ID: userID}).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func GetGuild(db *gorm.DB, guildID snowflake.ID) (*models.Guild, error) {
	var guild models.Guild
	if err := db.First(&guild, "id = ?", guildID).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}
