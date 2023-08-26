package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
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
	UserData() UserDataDB
	MinecraftServer() MinecraftServerDB
	MinecraftStatusPanel() MinecraftStatusPanelDB
	NoticeSchedule() NoticeScheduleDB
	RolePanelV2() RolePanelV2DB
	RolePanelV2Edit() RolePanelV2EditDB
	RolePanelV2Place() RolePanelV2PlaceDB
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
		db:                   db,
		pollCreate:           &pollCreateDBImpl{db: db},
		poll:                 &pollDBImpl{db: db},
		rolePanelCreate:      &rolePanelCreateDBImpl{db: db},
		rolePanel:            &rolePanelDBImpl{db: db},
		guildData:            &guildDataDBImpl{db: db, guildDataLocks: make(map[snowflake.ID]*sync.Mutex)},
		calc:                 &CalcDBImpl{db: db},
		messagePin:           &messagePinDBImpl{db: db},
		embedDialog:          &embedDialogDBImpl{db: db},
		userData:             &userDataDBImpl{db: db, userDataLocks: make(map[snowflake.ID]*sync.Mutex)},
		minecraftServer:      &minecraftServerDBImpl{db: db},
		minecraftStatusPanel: &minecraftStatusPanelDBImpl{db: db},
		noticeSchedule:       &noticeScheduleDBImpl{db: db},
		rolePanelV2:          &rolePanelV2DBImpl{db: db},
		rolePanelV2Edit:      &rolePanelV2EditDBImpl{db: db},
		rolePanelV2Place:     &rolePanelV2PlaceDBImpl{db: db},
		interactions:         &interactionsImpl{db: db},
	}, nil
}

var _ DB = (*dbImpl)(nil)

type dbImpl struct {
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
	interactions         *interactionsImpl
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

func (d *dbImpl) UserData() UserDataDB {
	return d.userData
}

func (d *dbImpl) MinecraftServer() MinecraftServerDB {
	return d.minecraftServer
}

func (d *dbImpl) MinecraftStatusPanel() MinecraftStatusPanelDB {
	return d.minecraftStatusPanel
}

func (d *dbImpl) NoticeSchedule() NoticeScheduleDB {
	return d.noticeSchedule
}

func (d *dbImpl) RolePanelV2() RolePanelV2DB {
	return d.rolePanelV2
}

func (d *dbImpl) RolePanelV2Edit() RolePanelV2EditDB {
	return d.rolePanelV2Edit
}

func (d *dbImpl) RolePanelV2Place() RolePanelV2PlaceDB {
	return d.rolePanelV2Place
}

func (d *dbImpl) Interactions() InteractionsDB {
	return d.interactions
}

func (d *dbImpl) Close() error {
	return d.db.Close()
}
