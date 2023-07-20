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

const GuildDataVersion = 4

type GuildData struct {
	ID              snowflake.ID                            `json:"id"`
	RolePanel       map[uuid.UUID]GuildDataRolePanel        `json:"role_panel"`
	RolePanelLimit  int                                     `json:"role_panel_limit"`
	UserPermissions map[snowflake.ID]permissions.Permission `json:"user_permissions"`
	RolePermissions map[snowflake.ID]permissions.Permission `json:"role_permissions"`
	UserLevels      map[snowflake.ID]GuildDataUserLevel     `json:"user_levels"`
	Config          GuildDataConfig                         `json:"config"`
	BumpStatus      BumpStatus                              `json:"bump_status"`

	MCStatusPanel     map[uuid.UUID]string `json:"mc_status_panel"`
	MCStatusPanelName map[string]int       `json:"mc_status_panel_name"`
	MCStatusPanelMax  int                  `json:"mc_status_panel_max"`

	UserLevelExcludeChannels map[snowflake.ID]string `json:"user_level_exclude_channels"`

	DataVersion *int `json:"data_version,omitempty"`
}

type BumpStatus struct {
	BumpEnabled     bool          `json:"bump_enabled"`
	BumpChannel     *snowflake.ID `json:"bump_channel"`
	BumpRole        *snowflake.ID `json:"bump_role"`
	BumpMessage     [2]string     `json:"bump_message"`
	BumpRemind      [2]string     `json:"bump_remind"`
	LastBump        time.Time     `json:"last_bump"`
	LastBumpChannel *snowflake.ID `json:"last_bump_channel"`
	UpEnabled       bool          `json:"up_enabled"`
	UpChannel       *snowflake.ID `json:"up_channel"`
	UpRole          *snowflake.ID `json:"up_role"`
	UpMessage       [2]string     `json:"up_message"`
	UpRemind        [2]string     `json:"up_remind"`
	LastUp          time.Time     `json:"last_up"`
	LastUpChannel   *snowflake.ID `json:"last_up_channel"`

	BumpCountMap map[snowflake.ID]uint64 `json:"bump_count_map"`
	UpCountMap   map[snowflake.ID]uint64 `json:"up_count_map"`
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
	case 2:
		g.MCStatusPanel = make(map[uuid.UUID]string)
		g.MCStatusPanelName = make(map[string]int)
		g.MCStatusPanelMax = 10
		*g.DataVersion = 3
		fallthrough
	case 3:
		g.BumpStatus.UpEnabled = true
		if g.BumpStatus.UpMessage == [2]string{} {
			g.BumpStatus.UpMessage = [2]string{
				"UPを検知したよ",
				"１時間後に通知するね！",
			}
		}
		if g.BumpStatus.UpRemind == [2]string{} {
			g.BumpStatus.UpRemind = [2]string{
				"UPできるよ!",
				"`/dissoku up` でUPしよう!",
			}
		}
		g.BumpStatus.BumpEnabled = true
		if g.BumpStatus.BumpMessage == [2]string{} {
			g.BumpStatus.BumpMessage = [2]string{
				"Bumpを検知したよ",
				"２時間後に通知するね！",
			}
		}
		if g.BumpStatus.BumpRemind == [2]string{} {
			g.BumpStatus.BumpRemind = [2]string{
				"Bumpできるよ!",
				"`/bump` でBumpしよう!",
			}
		}
		g.BumpStatus.BumpCountMap = make(map[snowflake.ID]uint64)
		g.BumpStatus.UpCountMap = make(map[snowflake.ID]uint64)
		g.UserLevelExcludeChannels = make(map[snowflake.ID]string)
		*g.DataVersion = 4
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
