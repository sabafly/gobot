package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

type RolePanelDB interface {
	Get(uuid.UUID) (RolePanel, error)
	Set(RolePanel) error
	Del(uuid.UUID) error
}

var _ RolePanelDB = (*rolePanelDBImpl)(nil)

type rolePanelDBImpl struct {
	db *redis.Client
}

func (r *rolePanelDBImpl) Get(id uuid.UUID) (RolePanel, error) {
	res := r.db.HGet(context.TODO(), "rolepanel", id.String())
	if err := res.Err(); err != nil {
		return RolePanel{}, err
	}
	data := RolePanel{}
	if err := json.Unmarshal([]byte(res.Val()), &data); err != nil {
		return RolePanel{}, err
	}
	return data, nil
}

func (r *rolePanelDBImpl) Set(data RolePanel) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := r.db.HSet(context.TODO(), "rolepanel", data.id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (r *rolePanelDBImpl) Del(id uuid.UUID) error {
	res := r.db.HDel(context.TODO(), "rolepanel", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanel(panel RolePanelCreate) RolePanel {
	return RolePanel{
		RolePanelCreate: &panel,
	}
}

type RolePanel struct {
	*RolePanelCreate `json:"role_data"`
	ChannelID        snowflake.ID `json:"channel_id"`
	MessageID        snowflake.ID `json:"message_id"`
	GuildID          snowflake.ID `json:"guild_id"`
}

func (r RolePanel) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		RoleData  RolePanelCreate `json:"role_data"`
		ChannelID snowflake.ID    `json:"channel_id"`
		MessageID snowflake.ID    `json:"message_id"`
		GuildID   snowflake.ID    `json:"guild_id"`
	}{
		RoleData:  *r.RolePanelCreate,
		ChannelID: r.ChannelID,
		MessageID: r.MessageID,
		GuildID:   r.GuildID,
	})
}

func (r *RolePanel) UnmarshalJSON(data []byte) error {
	aux := &struct {
		RoleData  *RolePanelCreate `json:"role_data"`
		ChannelID snowflake.ID     `json:"channel_id"`
		MessageID snowflake.ID     `json:"message_id"`
		GuildID   snowflake.ID     `json:"guild_id"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = RolePanel{
		RolePanelCreate: aux.RoleData,
		ChannelID:       aux.ChannelID,
		MessageID:       aux.MessageID,
		GuildID:         aux.GuildID,
	}
	return nil
}

func (r *RolePanel) BuildMessage(format func([]discord.Embed) []discord.Embed) discord.MessageCreate {
	fields := []discord.EmbedField{}
	roles := []RolePanelCreateRole{}
	for _, rpcr := range r.roles {
		roles = append(roles, rpcr)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position() < roles[j].Position()
	})
	for _, v := range roles {
		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%s:%s", formatComponentEmoji(v.Emoji), v.Label),
			Value: v.Description,
		})
	}
	embeds := []discord.Embed{
		{
			Footer: &discord.EmbedFooter{
				// XXX: 変数参照にしたい
				Text: fmt.Sprintf("%s %s", "gobot", translate.Message(r.locale, "role_panel")),
			},
			Title:       r.Name,
			Description: r.Description,
			Fields:      fields,
		},
	}
	embeds = format(embeds)
	components := []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.ButtonComponent{
				CustomID: fmt.Sprintf("handler:rolepanel:use:%s", r.UUID().String()),
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				Label:    translate.Message(r.locale, "command_text_role_panel_create_build_message_component_button_use_label"),
			},
		},
	}
	return discord.MessageCreate{
		Embeds:     embeds,
		Components: components,
	}
}

func (r *RolePanel) UseMessage(format func([]discord.Embed) []discord.Embed, member discord.Member) discord.MessageCreate {
	options := []discord.StringSelectMenuOption{}
	roles := []RolePanelCreateRole{}
	for _, rpcr := range r.roles {
		roles = append(roles, rpcr)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position() < roles[j].Position()
	})
	mRoles := map[snowflake.ID]bool{}
	for _, i2 := range member.RoleIDs {
		mRoles[i2] = true
	}
	for _, v := range roles {
		emoji := v.Emoji
		options = append(options, discord.StringSelectMenuOption{
			Label:       v.Label,
			Description: v.Description,
			Value:       v.RoleID.String(),
			Emoji:       &emoji,
			Default:     mRoles[v.RoleID],
		})
	}
	embeds := []discord.Embed{
		{
			Title:       translate.Message(r.locale, "command_text_role_panel_use_embed_title"),
			Description: translate.Message(r.locale, "command_text_role_panel_use_embed_description"),
		},
	}
	embeds = format(embeds)
	components := []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID: fmt.Sprintf("handler:rolepanel:getrole:%s", r.UUID().String()),
				MaxValues: func() int {
					if r.Max > len(options) {
						return len(options)
					}
					return r.Max
				}(),
				MinValues: &r.Min,
				Options:   options,
			},
		},
	}
	return discord.MessageCreate{
		Flags:      discord.MessageFlagEphemeral,
		Embeds:     embeds,
		Components: components,
	}
}

func formatComponentEmoji(e discord.ComponentEmoji) string {
	var zeroID snowflake.ID
	if e.ID == zeroID {
		return e.Name
	}
	if e.Animated {
		return fmt.Sprintf("<a:%s:%d>", e.Name, e.ID)
	} else {
		return fmt.Sprintf("<:%s:%d>", e.Name, e.ID)
	}
}
