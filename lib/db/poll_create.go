package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/lib/translate"
)

type PollCreateDB interface {
	Get(id uuid.UUID) (PollCreate, error)
	Set(id uuid.UUID, poll PollCreate) error
	Remove(id uuid.UUID) error
}

type pollCreateDBImpl struct {
	db *redis.Client
}

func (p *pollCreateDBImpl) Get(id uuid.UUID) (PollCreate, error) {
	res := p.db.Get(context.TODO(), "polls"+id.String())
	if err := res.Err(); err != nil {
		return PollCreate{}, err
	}
	buf, err := res.Result()
	if err != nil {
		return PollCreate{}, err
	}
	data := PollCreate{}
	err = json.Unmarshal([]byte(buf), &data)
	if err != nil {
		return PollCreate{}, err
	}
	return data, nil
}

func (p *pollCreateDBImpl) Set(id uuid.UUID, poll PollCreate) error {
	buf, err := json.Marshal(poll)
	if err != nil {
		return err
	}
	res := p.db.Set(context.TODO(), "polls"+id.String(), buf, time.Minute*14)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (p *pollCreateDBImpl) Remove(id uuid.UUID) error {
	res := p.db.Del(context.TODO(), "polls"+id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

type PollCreate struct {
	ID          uuid.UUID                      `json:"id"`
	Title       string                         `json:"title"`
	Description string                         `json:"description"`
	EndAt       int64                          `json:"time_limit"`
	MaxChoice   int                            `json:"max"`
	MinChoice   int                            `json:"min"`
	Choices     map[uuid.UUID]PollCreateChoice `json:"choices"`

	Locale   discord.Locale `json:"locale"`
	Settings PollSettings   `json:"settings"`
}

type PollCreateChoice struct {
	ID          uuid.UUID               `json:"id"`
	Position    int                     `json:"position"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Emoji       *discord.ComponentEmoji `json:"emoji"`
}

type PollSettings struct {
	ShowUser         PollSettingShowType `json:"show_user"`
	ShowCount        PollSettingShowType `json:"show_count"`
	ShowTotalCount   PollSettingsBool    `json:"show_total_count"`
	ShowUserInResult PollSettingsBool    `json:"show_user_in_result"`
	CanChangeTarget  PollSettingsBool    `json:"can_change_target"`
}

type PollSettingsType int

const (
	PollSettingsTypeShowUser PollSettingsType = iota + 1
	PollSettingsTypeShowCount
	PollSettingsTypeShowTotalCount
	PollSettingsTypeShowUserInResult
	PollSettingsTypeCanChangeTarget
)

type PollSettingShowType int

const (
	PollSettingShowTypeAlways = iota
	PollSettingShowTypeNever
	PollSettingShowTypeAfterVote
)

func (p PollSettingShowType) String(locale discord.Locale) string {
	var str string
	switch p {
	case PollSettingShowTypeAlways:
		str = translate.Message(locale, "command_text_poll_settings_show_type_always")
	case PollSettingShowTypeNever:
		str = translate.Message(locale, "command_text_poll_settings_show_type_never")
	case PollSettingShowTypeAfterVote:
		str = translate.Message(locale, "command_text_poll_settings_show_type_after_vote")
	default:
		str = "Unknown"
	}
	return str
}

type PollSettingsBool bool

func (p PollSettingsBool) EmojiString() string {
	if p {
		return "⭕"
	} else {
		return "❌"
	}
}

func (p *PollCreate) CreatePoll(user discord.User) Poll {
	choices := make(map[uuid.UUID]PollChoice)
	for k, pcc := range p.Choices {
		choices[k] = PollChoice{
			Name:        pcc.Name,
			Description: pcc.Description,
			ID:          pcc.ID,
			Position:    pcc.Position,
			Emoji:       pcc.Emoji,
			Users:       make(map[snowflake.ID]bool),
		}
	}
	poll := Poll{
		Username:    user.Username,
		UserAvatar:  *user.AvatarURL(),
		Users:       make(map[snowflake.ID]bool),
		ID:          uuid.New(),
		Title:       p.Title,
		Description: p.Description,
		EndAt:       p.EndAt,
		MaxChoice:   p.MaxChoice,
		MinChoice:   p.MinChoice,
		Choices:     choices,

		Locale:   p.Locale,
		Settings: p.Settings,
	}
	return poll
}

func (v *PollCreate) ConfigEmbed() []discord.Embed {
	embeds := []discord.Embed{}
	inline := true
	embeds = append(embeds, discord.Embed{
		Title: translate.Message(v.Locale, "command_text_poll_create_embed_message_title"),
		Fields: []discord.EmbedField{
			{
				Name:   translate.Message(v.Locale, "command_text_poll_create_embed_field_title"),
				Value:  fmt.Sprintf("```\r%s```", v.Title),
				Inline: &inline,
			},
			{
				Name:   translate.Message(v.Locale, "command_text_poll_create_embed_field_description"),
				Value:  fmt.Sprintf("```\r%s```", v.Description),
				Inline: &inline,
			},
			{
				Name:   translate.Message(v.Locale, "command_text_poll_create_embed_field_time_limit"),
				Value:  fmt.Sprintf("%s (%s)", discord.FormattedTimestampMention(v.EndAt, discord.TimestampStyleLongDateTime), discord.FormattedTimestampMention(v.EndAt, discord.TimestampStyleRelative)),
				Inline: &inline,
			},
			{
				Name: translate.Message(v.Locale, "command_text_poll_create_embed_field_choices"),
				Value: fmt.Sprintf("%3s: %2d\r%3s: %2d",
					translate.Message(v.Locale, "max"), v.MaxChoice,
					translate.Message(v.Locale, "min"), v.MinChoice,
				),
				Inline: &inline,
			},
		},
	})
	return embeds
}

func (v *PollCreate) Components() []discord.ContainerComponent {
	var options []discord.StringSelectMenuOption
	var disabled bool
	switch {
	case len(v.Choices) == 0:
		disabled = true
		options = []discord.StringSelectMenuOption{
			{
				Label: "if you can see this, it would be a bug!",
				Value: "dummy",
			},
		}
	case len(v.Choices) > 0:
		disabled = false
		choices := []PollCreateChoice{}
		for _, pc := range v.Choices {
			choices = append(choices, pc)
		}
		sort.Slice(choices, func(i, j int) bool {
			return choices[i].Position < choices[j].Position
		})
		for i, pc := range choices {
			o := discord.StringSelectMenuOption{
				Label:       pc.Name,
				Description: pc.Description,
				Value:       pc.ID.String(),
				Emoji:       pc.Emoji,
			}
			if o.Emoji == nil {
				o.Emoji = &discord.ComponentEmoji{
					Name: number2Emoji(i + 1),
				}
			}
			options = append(options, o)
		}
	}
	choicesSelectMenu := discord.StringSelectMenuComponent{
		CustomID:    fmt.Sprintf("handler:poll:editchoice:%s", v.ID.String()),
		Disabled:    disabled,
		Placeholder: translate.Message(v.Locale, "command_text_poll_create_embed_component_edit_choice_placeholder"),
		MaxValues:   1,
		Options:     options,
	}
	var addDisabled bool
	if len(v.Choices) >= 25 {
		addDisabled = true
	}
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			choicesSelectMenu,
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:poll:addchoice:%s", v.ID),
				Disabled: addDisabled,
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1081653685320433724),
					Name: "plus",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:changepollinfo:%s", v.ID.String()),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082025248330891388),
					Name: "modify",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:editsettings:%s", v.ID.String()),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1083053845632000010),
					Name: "setting",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:create:%s", v.ID.String()),
				Disabled: disabled,
				Emoji: &discord.ComponentEmoji{
					Name: "🛠️",
				},
			},
		},
	}
}

func (p *PollCreate) EditChoiceEmbed(choiceID uuid.UUID) []discord.Embed {
	inline := true
	return []discord.Embed{
		{
			Title: translate.Message(p.Locale, "command_text_poll_create_embed_component_edit_choice_response_message_title"),
			Fields: []discord.EmbedField{
				{
					Name:   translate.Message(p.Locale, "command_text_poll_create_modal_add_choice_component_name_label"),
					Value:  fmt.Sprintf("```\r%s```", p.Choices[choiceID].Name),
					Inline: &inline,
				},
				{
					Name:   translate.Message(p.Locale, "command_text_poll_create_modal_add_choice_component_description_label"),
					Value:  fmt.Sprintf("```\r%s```", p.Choices[choiceID].Description),
					Inline: &inline,
				},
				{
					Name:  translate.Message(p.Locale, "command_text_poll_create_embed_component_edit_choice_response_field_emoji_name"),
					Value: componentEmojiFormat(*p.Choices[choiceID].Emoji),
				},
			},
		},
	}
}

func (p *PollCreate) EditChoiceComponent(choiceID uuid.UUID) []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:poll:backmenu:%s", p.ID.String()),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1081932944739938414),
					Name: "left",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:changechoiceinfo:%s:%s", strings.ReplaceAll(p.ID.String(), "-", ""), strings.ReplaceAll(choiceID.String(), "-", "")),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082025248330891388),
					Name: "modify",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:changechoiceemoji:%s:%s", strings.ReplaceAll(p.ID.String(), "-", ""), strings.ReplaceAll(choiceID.String(), "-", "")),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082267519374589992),
					Name: "smile",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:deletechoice:%s:%s", strings.ReplaceAll(p.ID.String(), "-", ""), strings.ReplaceAll(choiceID.String(), "-", "")),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1081940223547678757),
					Name: "trash",
				},
			},
		},
	}
}

func (p *PollCreate) EditSettingsEmbed() []discord.Embed {
	return []discord.Embed{
		{
			Title: translate.Message(p.Locale, "command_text_poll_create_embed_component_edit_settings_response_message_title"),
			Fields: []discord.EmbedField{
				{
					Name:  translate.Message(p.Locale, "command_text_poll_settings_type_show_user"),
					Value: p.Settings.ShowUser.String(p.Locale),
				},
				{
					Name:  translate.Message(p.Locale, "command_text_poll_settings_type_show_count"),
					Value: p.Settings.ShowCount.String(p.Locale),
				},
				{
					Name:  translate.Message(p.Locale, "command_text_poll_settings_type_show_total_count"),
					Value: p.Settings.ShowTotalCount.EmojiString(),
				},
				{
					Name:  translate.Message(p.Locale, "command_text_poll_settings_type_show_user_in_result"),
					Value: p.Settings.ShowUserInResult.EmojiString(),
				},
				{
					Name:  translate.Message(p.Locale, "command_text_poll_settings_type_can_change_target"),
					Value: p.Settings.CanChangeTarget.EmojiString(),
				},
			},
		},
	}
}

func (p *PollCreate) EditSettingsComponent() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:poll:backmenu:%s", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1081932944739938414),
					Name: "left",
				},
			},
		},
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID:    fmt.Sprintf("handler:poll:changesettingsmenu:%s", p.ID),
				Placeholder: translate.Message(p.Locale, "command_text_poll_create_embed_component_edit_settings_response_select_menu_placeholder"),
				Options: []discord.StringSelectMenuOption{
					{
						Label: translate.Message(p.Locale, "command_text_poll_settings_type_show_user"),
						Value: "1",
						Emoji: &discord.ComponentEmoji{
							Name: "1️⃣",
						},
					},
					{
						Label: translate.Message(p.Locale, "command_text_poll_settings_type_show_count"),
						Value: "2",
						Emoji: &discord.ComponentEmoji{
							Name: "2️⃣",
						},
					},
					{
						Label: translate.Message(p.Locale, "command_text_poll_settings_type_show_total_count"),
						Value: "3",
						Emoji: &discord.ComponentEmoji{
							Name: "3️⃣",
						},
					},
					{
						Label: translate.Message(p.Locale, "command_text_poll_settings_type_show_user_in_result"),
						Value: "4",
						Emoji: &discord.ComponentEmoji{
							Name: "4️⃣",
						},
					},
					{
						Label: translate.Message(p.Locale, "command_text_poll_settings_type_can_change_target"),
						Value: "5",
						Emoji: &discord.ComponentEmoji{
							Name: "5️⃣",
						},
					},
				},
			},
		},
	}
}

func (p *PollCreate) ChangeSettingsMenuComponent(t PollSettingsType) []discord.ContainerComponent {
	resAction := discord.ActionRowComponent{}
	switch t {
	case PollSettingsTypeShowUser:
		resAction = discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showuser:0", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_always"),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showuser:1", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_never"),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showuser:2", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_after_vote"),
			},
		}
	case PollSettingsTypeShowCount:
		resAction = discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showcount:0", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_always"),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showcount:1", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_never"),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showcount:2", p.ID),
				Label:    translate.Message(p.Locale, "command_text_poll_settings_show_type_after_vote"),
			},
		}
	case PollSettingsTypeShowTotalCount:
		resAction = discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showtotalcount:true", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082691057931788368),
					Name: "o_",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showtotalcount:false", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082689149557014549),
					Name: "x_",
				},
			},
		}
	case PollSettingsTypeShowUserInResult:
		resAction = discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showuserinresult:true", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082691057931788368),
					Name: "o_",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:showuserinresult:false", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082689149557014549),
					Name: "x_",
				},
			},
		}
	case PollSettingsTypeCanChangeTarget:
		resAction = discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:canchangetarget:true", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082691057931788368),
					Name: "o_",
				},
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:poll:changesettings:%s:canchangetarget:false", p.ID),
				Emoji: &discord.ComponentEmoji{
					ID:   snowflake.ID(1082689149557014549),
					Name: "x_",
				},
			},
		}
	}
	res := p.EditSettingsComponent()
	res = append(res, resAction)
	return res
}

func number2Emoji(n int) string {
	return string(rune('🇦' - 1 + n))
}

func componentEmojiFormat(e discord.ComponentEmoji) string {
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
