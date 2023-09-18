package db

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type DBConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	DB   int    `json:"db"`
}

func SetupDatabase(cfg DBConfig) (*DB, error) {
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
	return &DB{
		db:                   db,
		pollCreate:           &pollCreateDBImpl{db: db},
		poll:                 &pollDBImpl{db: db},
		rolePanelCreate:      &rolePanelCreateDBImpl{db: db},
		rolePanel:            &rolePanelDBImpl{db: db},
		guildData:            &guildDataDBImpl{db: db},
		calc:                 &CalcDBImpl{db: db},
		messagePin:           &messagePinDBImpl{db: db},
		embedDialog:          &embedDialogDBImpl{db: db},
		userData:             &userDataDBImpl{db: db},
		minecraftServer:      &minecraftServerDBImpl{db: db},
		minecraftStatusPanel: &minecraftStatusPanelDBImpl{db: db},
		noticeSchedule:       &noticeScheduleDBImpl{db: db},
		rolePanelV2:          &rolePanelV2DBImpl{db: db},
		rolePanelV2Edit:      &rolePanelV2EditDBImpl{db: db},
		rolePanelV2Place:     &rolePanelV2PlaceDBImpl{db: db},
		ticketDB:             newAnyDB[Ticket, uuid.UUID](db),
		guildTicketData:      newAnyDB[GuildTicketData, snowflake.ID](db),
		usedID:               newAnyDB[UsedID, snowflake.ID](db),
		interactions:         &interactionsImpl{db: db},
	}, nil
}

type DB struct {
	db                   *redis.Client
	pollCreate           *pollCreateDBImpl
	poll                 *pollDBImpl
	rolePanelCreate      *rolePanelCreateDBImpl
	rolePanel            *rolePanelDBImpl
	guildData            *guildDataDBImpl
	calc                 *CalcDBImpl
	messagePin           *messagePinDBImpl
	embedDialog          *embedDialogDBImpl
	userData             *userDataDBImpl
	minecraftServer      *minecraftServerDBImpl
	minecraftStatusPanel *minecraftStatusPanelDBImpl
	noticeSchedule       *noticeScheduleDBImpl
	rolePanelV2          *rolePanelV2DBImpl
	rolePanelV2Edit      *rolePanelV2EditDBImpl
	rolePanelV2Place     *rolePanelV2PlaceDBImpl
	ticketDB             *anyDB[Ticket, uuid.UUID]
	guildTicketData      *anyDB[GuildTicketData, snowflake.ID]
	usedID               *anyDB[UsedID, snowflake.ID]
	interactions         *interactionsImpl
}

func (d *DB) PollCreate() PollCreateDB {
	return d.pollCreate
}

func (d *DB) Poll() PollDB {
	return d.poll
}

func (d *DB) RolePanelCreate() RolePanelCreateDB {
	return d.rolePanelCreate
}

func (d *DB) RolePanel() RolePanelDB {
	return d.rolePanel
}

func (d *DB) GuildData() GuildDataDB {
	return d.guildData
}

func (d *DB) Calc() CalcDB {
	return d.calc
}

func (d *DB) MessagePin() MessagePinDB {
	return d.messagePin
}

func (d *DB) EmbedDialog() EmbedDialogDB {
	return d.embedDialog
}

func (d *DB) UserData() UserDataDB {
	return d.userData
}

func (d *DB) MinecraftServer() MinecraftServerDB {
	return d.minecraftServer
}

func (d *DB) MinecraftStatusPanel() MinecraftStatusPanelDB {
	return d.minecraftStatusPanel
}

func (d *DB) NoticeSchedule() NoticeScheduleDB {
	return d.noticeSchedule
}

func (d *DB) RolePanelV2() RolePanelV2DB {
	return d.rolePanelV2
}

func (d *DB) RolePanelV2Edit() RolePanelV2EditDB {
	return d.rolePanelV2Edit
}

func (d *DB) RolePanelV2Place() RolePanelV2PlaceDB {
	return d.rolePanelV2Place
}

func (d *DB) Ticket() AnyDB[Ticket, uuid.UUID] {
	return d.ticketDB
}

func (d *DB) GuildTicketData() AnyDB[GuildTicketData, snowflake.ID] {
	return d.guildTicketData
}

func (d *DB) UsedID() AnyDB[UsedID, snowflake.ID] {
	return d.usedID
}

func (d *DB) Interactions() InteractionsDB {
	return d.interactions
}

func (d *DB) Close() error {
	return d.db.Close()
}
