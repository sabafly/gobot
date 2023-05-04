package db

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/sabafly/sabafly-lib/db"
)

type DBConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	DB   int    `json:"db"`
}

type DB interface {
	db.DB
	PollCreate() PollCreateDB
	Poll() PollDB
	RolePanelCreate() RolePanelCreateDB
	RolePanel() RolePanelDB
	GuildData() GuildDataDB
	Calc() CalcDB
	Interactions() InteractionsDB
}

func SetupDatabase(cfg DBConfig) (DB, error) {
	db := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		DB:      cfg.DB,
	})
	return &dbImpl{
		db:              db,
		pollCreate:      &pollCreateDBImpl{db: db},
		poll:            &pollDBImpl{db: db},
		rolePanelCreate: &rolePanelCreateDBImpl{db: db},
		rolePanel:       &rolePanelDBImpl{db: db},
		guildData:       &guildDataDBImpl{db: db},
		calc:            &CalcDBImpl{db: db},
		interactions:    &interactionsImpl{db: db},
	}, nil
}

var _ DB = (*dbImpl)(nil)

type dbImpl struct {
	db              *redis.Client
	pollCreate      *pollCreateDBImpl
	poll            *pollDBImpl
	rolePanelCreate *rolePanelCreateDBImpl
	rolePanel       *rolePanelDBImpl
	guildData       *guildDataDBImpl
	calc            *CalcDBImpl
	interactions    *interactionsImpl
}

func (d *dbImpl) PollCreate() PollCreateDB {
	return d.pollCreate
}

func (d *dbImpl) Poll() PollDB {
	return d.poll
}

func (d *dbImpl) RolePanelCreate() RolePanelCreateDB {
	return d.rolePanelCreate
}

func (d *dbImpl) RolePanel() RolePanelDB {
	return d.rolePanel
}

func (d *dbImpl) GuildData() GuildDataDB {
	return d.guildData
}

func (d *dbImpl) Calc() CalcDB {
	return d.calc
}

func (d *dbImpl) Interactions() InteractionsDB {
	return d.interactions
}

func (d *dbImpl) Close() error {
	return d.db.Close()
}
