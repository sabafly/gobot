package db

import (
	"context"
	"time"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-lib/v2/permissions"
)

type GuildDataDB interface {
	Get(id snowflake.ID) (GuildData, error)
	Set(id snowflake.ID, data GuildData) error
	Del(id snowflake.ID) error
}

var _ GuildDataDB = (*guildDataDBImpl)(nil)

type guildDataDBImpl struct {
	db *redis.Client
}

func (g *guildDataDBImpl) Get(id snowflake.ID) (GuildData, error) {
	res := g.db.HGet(context.TODO(), "guild-data", id.String())
	if err := res.Err(); err != nil {
		if err != redis.Nil {
			return GuildData{}, err
		} else {
			return NewGuildData(id), nil
		}
	}
	buf := []byte(res.Val())
	val := GuildData{}
	if err := json.Unmarshal(buf, &val); err != nil {
		return GuildData{}, err
	}
	return val, nil
}

func (g *guildDataDBImpl) Set(id snowflake.ID, data GuildData) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := g.db.HSet(context.TODO(), "guild-data", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (g *guildDataDBImpl) Del(id snowflake.ID) error {
	res := g.db.HDel(context.TODO(), "guild-data", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewGuildData(id snowflake.ID) GuildData {
	g := GuildData{
		ID:             id,
		RolePanel:      make(map[uuid.UUID]GuildDataRolePanel),
		RolePanelLimit: 10,
	}
	b, _ := json.Marshal(g)
	_ = g.validate(b)
	return g
}

const GuildDataVersion = 2

type GuildData struct {
	ID              snowflake.ID                            `json:"id"`
	RolePanel       map[uuid.UUID]GuildDataRolePanel        `json:"role_panel"`
	RolePanelLimit  int                                     `json:"role_panel_limit"`
	UserPermissions map[snowflake.ID]permissions.Permission `json:"user_permissions"`
	RolePermissions map[snowflake.ID]permissions.Permission `json:"role_permissions"`
	UserLevels      map[snowflake.ID]GuildDataUserLevel     `json:"user_levels"`
	Config          GuildDataConfig                         `json:"config"`

	DataVersion *int `json:"data_version,omitempty"`
}

type GuildDataConfig struct {
	LevelUpMessage        string        `json:"level_up_message"`
	LevelUpMessageChannel *snowflake.ID `json:"level_up_message_channel"`
}

type GuildDataUserLevel struct {
	UserDataLevel
	MessageCount    int64     `json:"message_count"`
	LastMessageTime time.Time `json:"last_message_time"`
}

func (g *GuildData) UnmarshalJSON(b []byte) error {
	type guildData GuildData
	var v struct {
		guildData
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*g = GuildData(v.guildData)
	if !g.isValid() {
		if err := g.validate(b); err != nil {
			return err
		}
	}
	return nil
}

func (g GuildData) isValid() bool {
	if g.DataVersion == nil {
		return false
	}
	return *g.DataVersion > GuildDataVersion
}

func (g *GuildData) validate(b []byte) error {
	if g.DataVersion == nil {
		g.DataVersion = json.Ptr(0)
	}
	switch *g.DataVersion {
	case 0:
		g.UserPermissions = make(map[snowflake.ID]permissions.Permission)
		g.RolePermissions = make(map[snowflake.ID]permissions.Permission)
		g.UserLevels = make(map[snowflake.ID]GuildDataUserLevel)
		*g.DataVersion = 1
		fallthrough
	case 1:
		g.Config = GuildDataConfig{
			LevelUpMessage: "{mention} がレベルアップしました。 {level} lv",
		}
		*g.DataVersion = 2
		fallthrough
	case GuildDataVersion:
		return nil
	default:
		d := NewGuildData(g.ID)
		*g = d
	}
	return nil
}

type GuildDataRolePanel struct {
	OnList bool `json:"on_list"`
}
