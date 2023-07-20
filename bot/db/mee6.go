package db

import "github.com/disgoorg/snowflake/v2"

type Mee6LeaderBoard struct {
	Admin             bool                `json:"admin"`
	BannerURL         *string             `json:"banner_url"`
	Country           string              `json:"country"`
	Guild             Mee6Guild           `json:"guild"`
	IsMember          bool                `json:"is_member"`
	MonetizeOptions   Mee6MonetizeOptions `json:"monetize_options"`
	Page              int                 `json:"page"`
	Player            *Mee6Player         `json:"player"`
	Players           []Mee6Player        `json:"players"`
	RoleRewards       []any               `json:"role_rewards"`
	UserGuildSettings any                 `json:"user_guild_settings"`
	XpPerMessage      []int               `json:"xp_per_message"`
	XpRate            float64             `json:"xp_rate"`
}

type Mee6Guild struct {
	AllowJoin                  bool   `json:"allow_join"`
	ApplicationCommandsEnabled bool   `json:"application_commands_enabled"`
	CommandsPrefix             string `json:"commands_prefix"`
	Icon                       string `json:"icon"`
	ID                         string `json:"id"`
	InviteLeaderboard          bool   `json:"invite_leaderboard"`
	LeaderboardURL             string `json:"leaderboard_url"`
	Name                       string `json:"name"`
	Premium                    bool   `json:"premium"`
}

type Mee6MonetizeOptions struct {
	DisplayPlans        bool `json:"display_plans"`
	ShowcaseSubscribers bool `json:"showcase_subscribers"`
}

type Mee6Player struct {
	Avatar               string       `json:"avatar"`
	DetailedXp           []int64      `json:"detailed_xp"`
	Discriminator        string       `json:"discriminator"`
	GuildID              string       `json:"guild_id"`
	ID                   snowflake.ID `json:"id"`
	IsMonetizeSubscriber bool         `json:"is_monetize_subscriber"`
	Level                int64        `json:"level"`
	MessageCount         int64        `json:"message_count"`
	MonetizeXpBoost      int64        `json:"monetize_xp_boost"`
	Username             string       `json:"username"`
	Xp                   int64        `json:"xp"`
}
