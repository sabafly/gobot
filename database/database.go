package database

import (
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
	); err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

type DB struct {
	DB *gorm.DB
}
