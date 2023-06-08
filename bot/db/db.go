package db

import (
	"context"
	"fmt"
	"time"

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
	RolePanelCreate() RolePanelCreateDB
	RolePanel() RolePanelDB
	GuildData() GuildDataDB
	Calc() CalcDB
	MessagePin() MessagePinDB
	EmbedDialog() EmbedDialogDB
	Interactions() InteractionsDB
}

func SetupDatabase(cfg DBConfig) (DB, error) {
	db := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		DB:      cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res := db.Ping(ctx)
	if err := res.Err(); err != nil {
		return nil, err
	}
	return &dbImpl{
		db:              db,
		pollCreate:      &pollCreateDBImpl{db: db},
		poll:            &pollDBImpl{db: db},
		rolePanelCreate: &rolePanelCreateDBImpl{db: db},
		rolePanel:       &rolePanelDBImpl{db: db},
		guildData:       &guildDataDBImpl{db: db},
		calc:            &CalcDBImpl{db: db},
		messagePin:      &messagePinDBImpl{db: db},
		embedDialog:     &embedDialogDBImpl{db: db},
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
	messagePin      *messagePinDBImpl
	embedDialog     *embedDialogDBImpl
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

func (d *dbImpl) MessagePin() MessagePinDB {
	return d.messagePin
}

func (d *dbImpl) EmbedDialog() EmbedDialogDB {
	return d.embedDialog
}

func (d *dbImpl) Interactions() InteractionsDB {
	return d.interactions
}

func (d *dbImpl) Close() error {
	return d.db.Close()
}
