package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-lib/translate"
)

type RolePanelCreateDB interface {
	Get(uuid.UUID) (RolePanelCreate, error)
	Set(RolePanelCreate) error
	Del(uuid.UUID) error
}

var _ RolePanelCreateDB = (*rolePanelCreateDBImpl)(nil)

type rolePanelCreateDBImpl struct {
	db *redis.Client
}

func (r *rolePanelCreateDBImpl) Get(id uuid.UUID) (RolePanelCreate, error) {
	res := r.db.Get(context.TODO(), "rolepanelcreate"+id.String())
	if err := res.Err(); err != nil {
		return RolePanelCreate{}, err
	}
	data := RolePanelCreate{}
	if err := json.Unmarshal([]byte(res.Val()), &data); err != nil {
		return RolePanelCreate{}, err
	}
	return data, nil
}

func (r *rolePanelCreateDBImpl) Set(data RolePanelCreate) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := r.db.Set(context.TODO(), "rolepanelcreate"+data.id.String(), buf, time.Minute*14)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (r *rolePanelCreateDBImpl) Del(id uuid.UUID) error {
	res := r.db.Del(context.TODO(), "rolepanelcreate"+id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanelCreate(name, description string, locale discord.Locale) RolePanelCreate {
	return RolePanelCreate{
		id:          uuid.New(),
		Name:        name,
		Description: description,
		roles:       make(map[snowflake.ID]RolePanelCreateRole),
		Max:         25,
		Min:         0,
		locale:      locale,
	}
}

type RolePanelCreate struct {
	id          uuid.UUID
	Name        string
	Description string
	roles       map[snowflake.ID]RolePanelCreateRole
	Max         int
	Min         int
	locale      discord.Locale
}

func (r RolePanelCreate) MarshalJSON() ([]byte, error) {
	v, err := json.Marshal(&struct {
		ID          uuid.UUID                            `json:"id"`
		Name        string                               `json:"name"`
		Description string                               `json:"description"`
		Roles       map[snowflake.ID]RolePanelCreateRole `json:"roles"`
		Max         int                                  `json:"max"`
		Min         int                                  `json:"min"`
		Locale      discord.Locale                       `json:"locale"`
	}{
		ID:          r.id,
		Name:        r.Name,
		Description: r.Description,
		Roles:       r.roles,
		Max:         r.Max,
		Min:         r.Min,
		Locale:      r.locale,
	})
	return v, err
}

func (r *RolePanelCreate) UnmarshalJSON(b []byte) error {
	r2 := &struct {
		ID          uuid.UUID                            `json:"id"`
		Name        string                               `json:"name"`
		Description string                               `json:"description"`
		Roles       map[snowflake.ID]RolePanelCreateRole `json:"roles"`
		Max         int                                  `json:"max"`
		Min         int                                  `json:"min"`
		Locale      discord.Locale                       `json:"locale"`
	}{}

	err := json.Unmarshal(b, r2)
	if err != nil {
		return err
	}

	r.id = r2.ID
	r.Name = r2.Name
	r.Description = r2.Description
	r.roles = r2.Roles
	r.Max = r2.Max
	r.Min = r2.Min
	r.locale = r2.Locale

	return err
}

func (r *RolePanelCreate) selectMenuComponent(id string, min, max int) discord.StringSelectMenuComponent {
	options := []discord.StringSelectMenuOption{}
	roles := []RolePanelCreateRole{}
	for _, rpcr := range r.roles {
		roles = append(roles, rpcr)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position() < roles[j].Position()
	})
	for i, rpcr := range roles {
		emoji := rpcr.Emoji
		options = append(options, discord.StringSelectMenuOption{
			Label:       rpcr.Label,
			Description: rpcr.Description,
			Emoji:       &emoji,
			Value:       rpcr.RoleID.String(),
		})

		rpcr.position = i + 1
		r.roles[rpcr.RoleID] = rpcr
	}
	disabled := false
	if len(options) < 1 {
		disabled = true
		options = append(options, discord.StringSelectMenuOption{
			Label: "dummy",
			Value: "dummy",
		})
	}
	return discord.StringSelectMenuComponent{
		CustomID:  id,
		MinValues: &min,
		MaxValues: max,
		Options:   options,
		Disabled:  disabled,
	}
}

func (r RolePanelCreate) BaseMenuEmbed() []discord.Embed {
	inline := true
	return []discord.Embed{
		{
			Title: translate.Message(r.locale, "command_text_role_panel_create_base_menu_embed_title"),
			Fields: []discord.EmbedField{
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_base_menu_embed_field_name_name"),
					Value:  fmt.Sprintf("```\r%s```", r.Name),
					Inline: &inline,
				},
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_base_menu_embed_field_description_name"),
					Value:  fmt.Sprintf("```\r%s```", r.Description),
					Inline: &inline,
				},
			},
		},
	}
}

func (r RolePanelCreate) BaseMenuComponent() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			r.selectMenuComponent(fmt.Sprintf("handler:rolepanel:editrole:%s", r.id.String()), 1, 1),
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:rolepanel:addrole:%s", r.id),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1081653685320433724),
					Name: "plus",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:rolepanel:editpanelinfo:%s", r.id.String()),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082025248330891388),
					Name: "modify",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:rolepanel:editpanelsettings:%s", r.id.String()),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1083053845632000010),
					Name: "setting",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:rolepanel:create:%s", r.id.String()),
				Emoji: &discord.ComponentEmoji{
					Name: "🛠️",
				},
				Disabled: len(r.roles) < 1,
			},
		},
	}
}

func (r *RolePanelCreate) EditRoleMenuEmbed(id snowflake.ID) []discord.Embed {
	inline := true
	return []discord.Embed{
		{
			Title: translate.Message(r.locale, "command_text_role_panel_create_edit_role_menu_embed_title"),
			Fields: []discord.EmbedField{
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_edit_role_menu_embed_fields_role"),
					Value:  discord.RoleMention(r.roles[id].RoleID),
					Inline: &inline,
				},
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_edit_role_menu_embed_fields_display_name"),
					Value:  fmt.Sprintf("```\r%s```", r.roles[id].Label),
					Inline: &inline,
				},
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_edit_role_menu_embed_fields_description"),
					Value:  fmt.Sprintf("```\r%s```", r.roles[id].Description),
					Inline: &inline,
				},
				{
					Name:   translate.Message(r.locale, "command_text_role_panel_create_edit_role_menu_embed_fields_emoji"),
					Value:  componentEmojiFormat(r.roles[id].Emoji),
					Inline: &inline,
				},
			},
		},
	}
}

func (r *RolePanelCreate) EditRoleMenuComponent(id snowflake.ID) discord.ContainerComponent {
	return discord.ActionRowComponent{
		r.BackMainMenuButton(),
		discord.ButtonComponent{
			Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
			CustomID: fmt.Sprintf("handler:rolepanel:editroleinfo:%s:%d", r.id.String(), id),
			Emoji: &discord.ComponentEmoji{
				ID:   snowflake.ID(1082025248330891388),
				Name: "modify",
			},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
			CustomID: fmt.Sprintf("handler:rolepanel:editroleemoji:%s:%d", r.id.String(), id),
			Emoji: &discord.ComponentEmoji{
				ID:   snowflake.ID(1082267519374589992),
				Name: "smile",
			},
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
			CustomID: fmt.Sprintf("handler:rolepanel:editdeleterole:%s:%d", r.id.String(), id),
			Emoji: &discord.ComponentEmoji{
				ID:   snowflake.ID(1081940223547678757),
				Name: "trash",
			},
		},
	}
}

func (r *RolePanelCreate) AddRoleMenuEmbed() []discord.Embed {
	return []discord.Embed{
		{
			Title:       translate.Message(r.locale, "command_text_role_panel_create_add_role_menu_embed_title"),
			Description: translate.Message(r.locale, "command_text_role_panel_create_add_role_menu_embed_description"),
		},
	}
}

func (r *RolePanelCreate) AddRoleMenuComponent() []discord.ContainerComponent {
	min := 1
	max := 25
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.RoleSelectMenuComponent{
				CustomID:  fmt.Sprintf("handler:rolepanel:addroleselectmenu:%s", r.id.String()),
				MinValues: &min,
				MaxValues: max,
			},
		},
		discord.ActionRowComponent{
			r.BackMainMenuButton(),
		},
	}
}

func (r *RolePanelCreate) EditPanelSettingsEmbed() []discord.Embed {
	return []discord.Embed{
		{
			Title: translate.Message(r.locale, "command_text_role_panel_create_edit_panel_config_title"),
			Fields: []discord.EmbedField{
				{
					Name:  translate.Message(r.locale, "command_text_role_panel_create_edit_panel_config_fields_max_name"),
					Value: fmt.Sprintf("%d %s", r.Max, translate.Translates(r.locale, "role", nil, r.Max, translate.WithFallback("role"))),
				},
				{
					Name:  translate.Message(r.locale, "command_text_role_panel_create_edit_panel_config_fields_min_name"),
					Value: fmt.Sprintf("%d %s", r.Min, translate.Translates(r.locale, "role", nil, r.Min, translate.WithFallback("role"))),
				},
			},
		},
	}
}

func (r *RolePanelCreate) EditPanelSettingsComponent() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID: fmt.Sprintf("handler:rolepanel:changesettings:%s", r.id.String()),
				Options: []discord.StringSelectMenuOption{
					{
						Label: translate.Message(r.locale, "command_text_role_panel_create_edit_panel_config_fields_max_name"),
						Value: "max",
						Emoji: &discord.ComponentEmoji{
							Name: "1️⃣",
						},
					},
					{
						Label: translate.Message(r.locale, "command_text_role_panel_create_edit_panel_config_fields_min_name"),
						Value: "min",
						Emoji: &discord.ComponentEmoji{
							Name: "2️⃣",
						},
					},
				},
			},
		},
		discord.ActionRowComponent{
			r.BackMainMenuButton(),
		},
	}
}

func (r *RolePanelCreate) UUID() uuid.UUID {
	return r.id
}

func (r *RolePanelCreate) DeleteRole(id snowflake.ID) {
	delete(r.roles, id)
}

func (r RolePanelCreate) GetRole(id snowflake.ID) (RolePanelCreateRole, bool) {
	role, ok := r.roles[id]
	return role, ok
}

func (r RolePanelCreate) GetRoles() map[snowflake.ID]RolePanelCreateRole {
	return r.roles
}

func (r *RolePanelCreate) SetRole(label, description string, roleID snowflake.ID, emoji *discord.ComponentEmoji) {
	if v, ok := r.roles[roleID]; ok {
		if emoji == nil {
			emoji = &discord.ComponentEmoji{
				Name: number2Emoji(v.position),
			}
		}
		v.Label = label
		v.Description = description
		v.RoleID = roleID
		v.Emoji = *emoji
		r.roles[roleID] = v
	} else {
		if len(r.roles) >= 25 {
			panic("cannot add roles more than 25")
		}
		if emoji == nil {
			emoji = &discord.ComponentEmoji{
				Name: number2Emoji(len(r.roles) + 1),
			}
		}
		r.roles[roleID] = RolePanelCreateRole{
			position:    len(r.roles) + 1,
			Label:       label,
			Description: description,
			RoleID:      roleID,
			Emoji:       *emoji,
		}
	}
}

func (r *RolePanelCreate) Validate() {
	if r.Max <= 0 {
		r.Max = 1
	}
	if r.Max > 25 {
		r.Max = 25
	}
	if r.Min < 0 {
		r.Min = 0
	}
	if r.Min > r.Max {
		r.Min = r.Max
	}
}

func (r RolePanelCreate) BackMainMenuButton() discord.ButtonComponent {
	return discord.ButtonComponent{
		Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
		CustomID: fmt.Sprintf("handler:rolepanel:backmainmenu:%s", r.id.String()),
		Emoji: &discord.ComponentEmoji{
			ID:   snowflake.ID(1081932944739938414),
			Name: "left",
		},
	}
}

type RolePanelCreateRole struct {
	position    int
	Label       string
	Description string
	RoleID      snowflake.ID
	Emoji       discord.ComponentEmoji
}

func (r RolePanelCreateRole) MarshalJSON() ([]byte, error) {
	v, err := json.Marshal(&struct {
		Position    int                    `json:"position"`
		Label       string                 `json:"label"`
		Description string                 `json:"description"`
		RoleID      snowflake.ID           `json:"role_id"`
		Emoji       discord.ComponentEmoji `json:"emoji"`
	}{
		Position:    r.position,
		Label:       r.Label,
		Description: r.Description,
		RoleID:      r.RoleID,
		Emoji:       r.Emoji,
	})
	return v, err
}

func (r *RolePanelCreateRole) UnmarshalJSON(b []byte) error {
	r2 := &struct {
		Position    int                    `json:"position"`
		Label       string                 `json:"label"`
		Description string                 `json:"description"`
		RoleID      snowflake.ID           `json:"role_id"`
		Emoji       discord.ComponentEmoji `json:"emoji"`
	}{}
	err := json.Unmarshal(b, r2)
	if err != nil {
		return err
	}
	r.position = r2.Position
	r.Label = r2.Label
	r.Description = r2.Description
	r.RoleID = r2.RoleID
	r.Emoji = r2.Emoji
	return nil
}

func (r RolePanelCreateRole) Position() int {
	return r.position
}
