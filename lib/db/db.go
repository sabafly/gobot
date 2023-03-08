package db

import (
	"fmt"

	"github.com/go-redis/redis/v8"
)

type DBConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	DB   int    `json:"db"`
}

type DB interface {
	Close() error
	PollCreate() PollCreateDB
	Poll() PollDB
	Interactions() InteractionsDB
}

func SetupDatabase(cfg DBConfig) (DB, error) {
	db := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		DB:      cfg.DB,
	})
	return &dbImpl{
		db:           db,
		pollCreate:   &pollCreateDBImpl{db: db},
		poll:         &pollDBImpl{db: db},
		interactions: &interactionsImpl{db: db},
	}, nil
}

type dbImpl struct {
	db           *redis.Client
	pollCreate   *pollCreateDBImpl
	poll         *pollDBImpl
	interactions *interactionsImpl
}

func (d *dbImpl) PollCreate() PollCreateDB {
	return d.pollCreate
}

func (d *dbImpl) Poll() PollDB {
	return d.poll
}

func (d *dbImpl) Interactions() InteractionsDB {
	return d.interactions
}

func (d *dbImpl) Close() error {
	return d.db.Close()
}
