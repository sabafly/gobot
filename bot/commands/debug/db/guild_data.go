package db

import (
	"time"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

const GuildDataVersion = 11

type GuildData struct {
	ID             snowflake.ID                        `json:"id"`
	RolePanel      map[uuid.UUID]GuildDataRolePanel    `json:"role_panel"`
	RolePanelLimit int                                 `json:"role_panel_limit"`
	UserLevels     map[snowflake.ID]GuildDataUserLevel `json:"user_levels"`
	Config         GuildDataConfig                     `json:"config"`
	BumpStatus     BumpStatus                          `json:"bump_status"`

	MCStatusPanel     map[uuid.UUID]string `json:"mc_status_panel"`
	MCStatusPanelName map[string]int       `json:"mc_status_panel_name"`
	MCStatusPanelMax  int                  `json:"mc_status_panel_max"`

	UserLevelExcludeChannels map[snowflake.ID]string `json:"user_level_exclude_channels"`

	RolePanelV2             map[uuid.UUID]string         `json:"role_panel_v2"`
	RolePanelV2Name         map[string]int               `json:"role_panel_v2_name"`
	RolePanelV2Placed       map[string]uuid.UUID         `json:"role_panel_v2_placed"`
	RolePanelV2PlacedConfig map[string]RolePanelV2Config `json:"role_panel_v2_placed_config"`
	RolePanelV2Limit        int                          `json:"role_panel_v2_limit"`

	RolePanelV2Editing map[uuid.UUID]uuid.UUID `json:"role_panel_v2_editing"`

	RolePanelV2EditingEmoji map[uuid.UUID][2]snowflake.ID `json:"role_panel_v2_emoji"`

	DataVersion *int `json:"data_version,omitempty"`
}

func NewMessageSuffix(target snowflake.ID, suffix string, rule MessageSuffixRuleType) MessageSuffix {
	return MessageSuffix{
		Target:   target,
		Suffix:   suffix,
		RuleType: rule,
	}
}

type MessageSuffix struct {
	Target   snowflake.ID          `json:"target"`
	Suffix   string                `json:"suffix"`
	RuleType MessageSuffixRuleType `json:"rule_type"`
}

type MessageSuffixRuleType int

const (
	MessageSuffixRuleTypeWarning = iota
	MessageSuffixRuleTypeDelete
	MessageSuffixRuleTypeWebhook
)

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
	return nil
}

type GuildDataRolePanel struct {
	OnList bool `json:"on_list"`
}
