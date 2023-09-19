package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

type PollDB interface {
	GetAll() ([]Poll, error)
	Get(id uuid.UUID) (Poll, error)
	Set(id uuid.UUID, poll Poll) error
	Del(id uuid.UUID) error
}

var _ PollDB = (*pollDBImpl)(nil)

type pollDBImpl struct {
	db *redis.Client
}

func (p *pollDBImpl) GetAll() ([]Poll, error) {
	res := p.db.HGetAll(context.TODO(), "poll")
	if err := res.Err(); err != nil {
		return []Poll{}, err
	}
	rmap := res.Val()
	rt := []Poll{}
	for _, v := range rmap {
		data := Poll{}
		err := json.Unmarshal([]byte(v), &data)
		if err != nil {
			return []Poll{}, err
		}
		rt = append(rt, data)
	}
	return rt, nil
}

func (p *pollDBImpl) Get(id uuid.UUID) (Poll, error) {
	res := p.db.HGet(context.TODO(), "poll", id.String())
	if err := res.Err(); err != nil {
		return Poll{}, err
	}
	buf, err := res.Result()
	if err != nil {
		return Poll{}, err
	}
	data := Poll{}
	err = json.Unmarshal([]byte(buf), &data)
	if err != nil {
		return Poll{}, err
	}
	return data, nil
}

func (p *pollDBImpl) Set(id uuid.UUID, poll Poll) error {
	buf, err := json.Marshal(poll)
	if err != nil {
		return err
	}
	res := p.db.HSet(context.TODO(), "poll", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (p *pollDBImpl) Del(id uuid.UUID) error {
	res := p.db.HDel(context.TODO(), "poll", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

type Poll struct {
	Username    string                `json:"username"`
	UserAvatar  string                `json:"user_avatar"`
	Users       map[snowflake.ID]bool `json:"users"`
	ID          uuid.UUID             `json:"id"`
	MessageID   snowflake.ID          `json:"message_id"`
	GuildId     snowflake.ID          `json:"guild_id"`
	ChannelID   snowflake.ID
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	EndAt       int64                    `json:"end_at"`
	MaxChoice   int                      `json:"max"`
	MinChoice   int                      `json:"min"`
	Choices     map[uuid.UUID]PollChoice `json:"choices"`

	Locale   discord.Locale `json:"locale"`
	Settings PollSettings   `json:"settings"`
	Finished bool           `json:"finished"`
}

type PollChoice struct {
	Users       map[snowflake.ID]bool   `json:"users"`
	ID          uuid.UUID               `json:"id"`
	Position    int                     `json:"position"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Emoji       *discord.ComponentEmoji `json:"emoji"`
}

func (p *Poll) MessageEmbed() []discord.Embed {
	choicesEmbed := discord.Embed{
		Title: translate.Message(p.Locale, "poll_choices_title"),
	}
	inline := true
	choices := []PollChoice{}
	for _, pc := range p.Choices {
		choices = append(choices, pc)
	}
	sort.Slice(choices, func(i, j int) bool {
		return choices[i].Position < choices[j].Position
	})
	for _, pc := range choices {
		choicesEmbed.Fields = append(choicesEmbed.Fields, discord.EmbedField{
			Name:   fmt.Sprintf("%s | %s", botlib.FormatComponentEmoji(*pc.Emoji), pc.Name),
			Value:  pc.Description,
			Inline: &inline,
		})
	}
	fields := []discord.EmbedField{
		{
			Name:  translate.Message(p.Locale, "poll_embed_field_end_at"),
			Value: discord.FormattedTimestampMention(p.EndAt, discord.TimestampStyleLongDateTime),
		},
		{
			Name: translate.Message(p.Locale, "poll_embed_field_number_of_votes"),
			Value: fmt.Sprintf("%3s: %2d\r%3s: %2d",
				translate.Message(p.Locale, "max"), p.MaxChoice,
				translate.Message(p.Locale, "min"), p.MinChoice,
			),
		},
	}
	fieldEx := []discord.EmbedField{}
	if p.Settings.ShowTotalCount {
		var count int
		for _, pc := range p.Choices {
			count += len(pc.Users)
		}
		fieldEx = append(fieldEx, discord.EmbedField{
			Name:  translate.Message(p.Locale, "poll_embed_field_ex_total_votes"),
			Value: fmt.Sprintf("%d", len(p.Users)),
		})
	}
	fields = append(fields, fieldEx...)

	return []discord.Embed{
		{
			Title:       p.Title,
			Description: p.Description,
			Author: &discord.EmbedAuthor{
				Name:    p.Username,
				IconURL: p.UserAvatar,
			},
			Fields: fields,
		},
		choicesEmbed,
	}
}

func (p *Poll) MessageComponent() []discord.ContainerComponent {
	if time.Now().After(time.Unix(p.EndAt, 0)) {
		return []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.ButtonComponent{
					Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
					CustomID: fmt.Sprintf("handler:poll:seeresult:%s", p.ID),
					Label:    translate.Message(p.Locale, "poll_component_button_see_result_label"),
				},
			},
		}
	} else {
		return []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.ButtonComponent{
					Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
					CustomID: fmt.Sprintf("handler:poll:vote:%s", p.ID),
					Label:    translate.Message(p.Locale, "poll_component_button_vote_label"),
				},
				discord.ButtonComponent{
					Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
					CustomID: fmt.Sprintf("handler:poll:seeinfo:%s", p.ID),
					Label:    translate.Message(p.Locale, "poll_component_button_see_info_label"),
				},
			},
		}
	}
}

func (p *Poll) VoteComponent(tokenID uuid.UUID) []discord.ContainerComponent {
	var options []discord.StringSelectMenuOption
	choices := []PollChoice{}
	for _, pc := range p.Choices {
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
				Name: botlib.Number2Emoji(i + 1),
			}
		}
		options = append(options, o)
	}
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID:  fmt.Sprintf("handler:poll:votedo:%s:%s", p.ID, tokenID),
				Options:   options,
				MaxValues: p.MaxChoice,
				MinValues: &p.MinChoice,
			},
		},
	}
}

func (p *Poll) SeeInfoComponent(tokenID uuid.UUID) []discord.ContainerComponent {
	var options []discord.StringSelectMenuOption
	choices := []PollChoice{}
	for _, pc := range p.Choices {
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
				Name: botlib.Number2Emoji(i + 1),
			}
		}
		options = append(options, o)
	}
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID: fmt.Sprintf("handler:poll:seeinfodo:%s:%s", p.ID, tokenID),
				Options:  options,
			},
		},
	}
}

func (p *Poll) SeeResultComponent(tokenID uuid.UUID) []discord.ContainerComponent {
	var options []discord.StringSelectMenuOption
	choices := []PollChoice{}
	for _, pc := range p.Choices {
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
				Name: botlib.Number2Emoji(i + 1),
			}
		}
		options = append(options, o)
	}
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.StringSelectMenuComponent{
				CustomID: fmt.Sprintf("handler:poll:seeresultdo:%s:%s", p.ID, tokenID),
				Options:  options,
			},
		},
	}
}
