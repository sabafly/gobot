package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/db"
	"github.com/sabafly/gobot/lib/handler"
	"github.com/sabafly/gobot/lib/structs"
	"github.com/sabafly/gobot/lib/translate"
)

func Poll(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionManageServer),
			Name:                     "poll",
			Description:              "create, and manage poll",
			DMPermission:             &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "create",
					Description: "create a new poll",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "title",
							Description: "title of poll",
							Required:    true,
							MaxLength:   pint(54),
						},
						discord.ApplicationCommandOptionString{
							Name:        "description",
							Description: "description of poll",
							Required:    true,
							MaxLength:   pint(2048),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-year",
							Description: "year of end time",
							Required:    true,
							MinValue:    pint(time.Now().Year()),
							MaxValue:    pint(time.Now().Year() + 1),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-month",
							Description: "month of end time",
							Required:    true,
							MinValue:    pint(1),
							MaxValue:    pint(12),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-day",
							Description: "day of end time",
							Required:    true,
							MinValue:    pint(1),
							MaxValue:    pint(31),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-hour",
							Description: "hour of end time",
							Required:    true,
							MinValue:    pint(0),
							MaxValue:    pint(23),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-minute",
							Description: "minute of end time",
							Required:    true,
							MinValue:    pint(0),
							MaxValue:    pint(59),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "time-zone",
							Description: "timezone of end time",
							Required:    true,
							MinValue:    pint(-12),
							MaxValue:    pint(+14),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "max-choice",
							Description: "Maximum number of votes a user can have",
							Required:    true,
							MinValue:    pint(1),
							MaxValue:    pint(25),
						},
						discord.ApplicationCommandOptionInt{
							Name:        "min-choice",
							Description: "Minimum number of votes a user can have",
							Required:    false,
							MinValue:    pint(1),
							MaxValue:    pint(25),
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"create": pollCreateHandler(b),
		},
	}
}

func pollCreateHandler(b *botlib.Bot) func(e *events.ApplicationCommandInteractionCreate) error {
	return func(e *events.ApplicationCommandInteractionCreate) error {
		timeLimit := time.Date(e.SlashCommandInteractionData().Int("time-year"), time.Month(e.SlashCommandInteractionData().Int("time-month")), e.SlashCommandInteractionData().Int("time-day"), e.SlashCommandInteractionData().Int("time-hour"), e.SlashCommandInteractionData().Int("time-minute"), 0, 0, time.FixedZone("", e.SlashCommandInteractionData().Int("time-zone")*60*60))
		if time.Now().After(timeLimit) {
			timeLimit = time.Now().Add(time.Hour)
		}
		min, ok := e.SlashCommandInteractionData().OptInt("min-choice")
		if !ok {
			min = 1
		}
		v := db.PollCreate{
			ID:          snowflake.New(time.Now()),
			Title:       e.SlashCommandInteractionData().String("title"),
			Description: e.SlashCommandInteractionData().String("description"),
			EndAt:       timeLimit.Unix(),
			MaxChoice:   e.SlashCommandInteractionData().Int("max-choice"),
			MinChoice:   min,
			Choices:     make(map[string]db.PollCreateChoice),
			Locale:      e.Locale(),
		}
		embeds := v.ConfigEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		err := e.CreateMessage(discord.MessageCreate{
			Embeds:     embeds,
			Components: v.Components(),
		})
		if err != nil {
			return err
		}
		if err := b.DB.PollCreate().Set(v.ID, v); err != nil {
			return err
		}
		if err := b.DB.Interactions().Set(v.ID, e.Token()); err != nil {
			return err
		}
		return nil
	}
}

func PollComponent(b *botlib.Bot) handler.Component {
	return handler.Component{
		Name: "poll",
		Handler: map[string]handler.ComponentHandler{
			"add-choice":           pollComponentAddChoice(b),
			"edit-choice":          pollComponentEditChoice(b),
			"edit-settings":        pollComponentEditSettings(b),
			"back-menu":            pollComponentBackMenu(b),
			"delete-choice":        pollCOmponentDeleteChoice(b),
			"change-choice-info":   pollComponentChangeChoiceInfo(b),
			"change-choice-emoji":  pollComponentChangeChoiceEmoji(b),
			"change-settings-menu": pollComponentChangeSettingsMenu(b),
			"change-settings":      pollComponentChangeSettings(b),
			"create":               pollComponentCreate(b),
			"create-do":            pollComponentCreateDo(b),
			"vote":                 pollComponentVote(b),
			"vote-do":              pollComponentVoteDo(b),
			"see-info":             pollComponentSeeInfo(b),
			"see-info-do":          pollComponentSeeInfoDo(b),
			"see-result":           pollComponentSeeResult(b),
			"see-result-do":        pollComponentSeeResultDo(b),
		},
	}
}

func pollComponentSeeResultDo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(snowflake.MustParse(args[4]))
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		choice, ok := p.Choices[e.StringSelectMenuInteractionData().Values[0]]
		if !ok {
			return nil
		}
		fields := []discord.EmbedField{}
		fields = append(fields, discord.EmbedField{
			Name:  translate.Message(e.Locale(), "poll_embed_field_number_of_votes"),
			Value: fmt.Sprintf("%d", len(choice.Users)),
		})
		if p.Settings.ShowUserInResult {
			var str string
			for i := range choice.Users {
				str += discord.UserMention(i) + "\r"
			}
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(e.Locale(), "poll_component_voter"),
				Value: str,
			})
		}
		embeds := []discord.Embed{
			{
				Title:  translate.Translate(e.Locale(), "poll_component_see_info_response_embed_title", map[string]any{"Name": choice.Name}),
				Fields: fields,
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentSeeResult(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		tokenID := snowflake.New(time.Now())
		err = b.DB.Interactions().Set(tokenID, e.Token())
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		component := p.SeeResultComponent(tokenID)
		err = e.CreateMessage(discord.MessageCreate{
			Flags:      discord.MessageFlagEphemeral,
			Components: component,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentSeeInfoDo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(snowflake.MustParse(args[4]))
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		choice, ok := p.Choices[e.StringSelectMenuInteractionData().Values[0]]
		if !ok {
			return nil
		}
		fields := []discord.EmbedField{}
		var isUser bool
		var isCount bool
		if _, ok := p.Users[e.Member().User.ID]; p.Settings.ShowUser != db.PollSettingShowTypeNever || p.Settings.ShowUser == db.PollSettingShowTypeAlways || ok {
			isUser = true
		}
		if _, ok := p.Users[e.Member().User.ID]; p.Settings.ShowCount != db.PollSettingShowTypeNever || p.Settings.ShowCount == db.PollSettingShowTypeAlways || ok {
			isCount = true
		}
		if isCount {
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(e.Locale(), "poll_embed_field_number_of_votes"),
				Value: fmt.Sprintf("%d", len(choice.Users)),
			})
		}
		if isUser {
			var str string
			for i := range choice.Users {
				str += discord.UserMention(i) + "\r"
			}
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(e.Locale(), "poll_component_voter"),
				Value: str,
			})
		}
		embeds := []discord.Embed{
			{
				Title:  translate.Translate(e.Locale(), "poll_component_see_info_response_embed_title", map[string]any{"Name": choice.Name}),
				Fields: fields,
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentSeeInfo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if p.Settings.ShowCount != db.PollSettingShowTypeAlways && p.Settings.ShowUser != db.PollSettingShowTypeAlways {
			if p.Settings.ShowCount == db.PollSettingShowTypeNever && p.Settings.ShowUser == db.PollSettingShowTypeNever {
				embeds := []discord.Embed{
					{
						Title: translate.Message(e.Locale(), "poll_component_cannot_use_this"),
						Color: 0xff0000,
					},
				}
				embeds = botlib.SetEmbedProperties(embeds)
				err := e.CreateMessage(discord.MessageCreate{
					Flags:  discord.MessageFlagEphemeral,
					Embeds: embeds,
				})
				if err != nil {
					return err
				}
				return nil
			}
		}
		tokenID := snowflake.New(time.Now())
		err = b.DB.Interactions().Set(tokenID, e.Token())
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		component := p.SeeInfoComponent(tokenID)
		err = e.CreateMessage(discord.MessageCreate{
			Flags:      discord.MessageFlagEphemeral,
			Components: component,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentVoteDo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		p.Users[e.Member().User.ID] = true
		for k, pc := range p.Choices {
			delete(pc.Users, e.Member().User.ID)
			p.Choices[k] = pc
		}
		var voted string
		for _, v := range e.StringSelectMenuInteractionData().Values {
			choice, ok := p.Choices[v]
			if !ok {
				err := e.CreateMessage(discord.MessageCreate{
					Content: "an critical error has occurred",
				})
				if err != nil {
					return err
				}
				return nil
			}
			choice.Users[e.Member().User.ID] = true
			p.Choices[v] = choice
			voted += fmt.Sprintf("%s | %s\r", botlib.FormatComponentEmoji(*choice.Emoji), choice.Name)
		}
		err = b.DB.Poll().Remove(p.ID)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = b.DB.Poll().Set(p.ID, p)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(snowflake.MustParse(args[4]))
		if err != nil {
			b.Logger.Error(err)
		}
		err = e.Client().Rest().DeleteInteractionResponse(e.ApplicationID(), token)
		if err != nil {
			b.Logger.Error(err)
		}
		embeds := p.MessageEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := p.MessageComponent()
		_, err = e.Client().Rest().UpdateMessage(p.ChannelID, p.MessageID, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			b.Logger.Error(err)
		}
		embeds = []discord.Embed{
			{
				Title:       translate.Message(e.Locale(), "poll_component_select_menu_vote_do_response_title"),
				Description: voted,
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		err = e.CreateMessage(discord.MessageCreate{
			Flags:  discord.MessageFlagEphemeral,
			Embeds: embeds,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentVote(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		p, err := b.DB.Poll().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		tokenID := snowflake.New(time.Now())
		if !p.Settings.CanChangeTarget {
			_, ok := p.Users[e.Member().User.ID]
			if ok {
				var voted string
				for _, v := range p.Choices {
					_, ok := v.Users[e.Member().User.ID]
					if !ok {
						continue
					}
					voted += fmt.Sprintf("%s | %s\r", botlib.FormatComponentEmoji(*v.Emoji), v.Name)
				}
				embeds := []discord.Embed{
					{
						Title: translate.Message(e.Locale(), "poll_component_button_vote_response_already_voted"),
						Fields: []discord.EmbedField{
							{
								Name:  translate.Message(e.Locale(), "poll_component_button_vote_response_already_voted_field_votes_name"),
								Value: voted,
							},
						},
					},
				}
				embeds = botlib.SetEmbedProperties(embeds)
				err := e.CreateMessage(discord.MessageCreate{
					Flags:  discord.MessageFlagEphemeral,
					Embeds: embeds,
				})
				if err != nil {
					return err
				}
				return nil
			}
		}
		components := p.VoteComponent(tokenID)
		err = e.CreateMessage(discord.MessageCreate{
			Flags:      discord.MessageFlagEphemeral,
			Components: components,
		})
		if err != nil {
			return err
		}
		if err := b.DB.Interactions().Set(tokenID, e.Token()); err != nil {
			return err
		}
		return nil
	}
}

func pollComponentCreate(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		embeds := []discord.Embed{
			{
				Title:       translate.Message(e.Locale(), "command_text_poll_create_embed_create_title"),
				Description: translate.Message(e.Locale(), "command_text_poll_create_embed_create_description"),
			},
		}
		components := []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.ChannelSelectMenuComponent{
					CustomID: fmt.Sprintf("handler:poll:create-do:%d", v.ID),
					ChannelTypes: []discord.ComponentType{
						discord.ComponentType(discord.ChannelTypeGuildText),
					},
				},
			},
			discord.ActionRowComponent{
				discord.ButtonComponent{
					Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
					CustomID: fmt.Sprintf("handler:poll:back-menu:%d", v.ID),
					Emoji: &discord.ComponentEmoji{
						ID:   snowflake.ID(1081932944739938414),
						Name: "left",
					},
				},
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentCreateDo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		poll := v.CreatePoll(e.Member().User)
		poll.GuildId = *e.GuildID()
		embeds := poll.MessageEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := poll.MessageComponent()
		val := e.ChannelSelectMenuInteractionData().Values[0]
		poll.ChannelID = val
		if poll.MaxChoice > len(poll.Choices) {
			poll.MaxChoice = len(poll.Choices)
		}
		if poll.MaxChoice < poll.MinChoice {
			poll.MinChoice = poll.MaxChoice
		}
		m, err := e.Client().Rest().CreateMessage(val, discord.MessageCreate{
			Embeds:     embeds,
			Components: components,
		})
		if err != nil {
			return err
		}
		poll.MessageID = m.ID
		err = b.DB.Poll().Set(poll.ID, poll)
		if err != nil {
			return err
		}
		go End(b, poll)
		err = e.Client().Rest().DeleteInteractionResponse(e.ApplicationID(), token)
		if err != nil {
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentAddChoice(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateModal(discord.ModalCreate{
			CustomID: fmt.Sprintf("handler:poll:add-choice:%d", v.ID),
			Title:    translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_title"),
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "name",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_label"),
						Placeholder: translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_placeholder"),
						Required:    true,
						MaxLength:   100,
					},
				},
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "description",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_label"),
						Placeholder: translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_placeholder"),
						Required:    false,
						MaxLength:   100,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentEditSettings(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		embeds := v.EditSettingsEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := v.EditSettingsComponent()
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentChangeSettingsMenu(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		val := e.StringSelectMenuInteractionData().Values[0]
		embeds := v.EditSettingsEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		switch val {
		case "1":
			components := v.ChangeSettingsMenuComponent(db.PollSettingsTypeShowUser)
			_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return err
			}
		case "2":
			components := v.ChangeSettingsMenuComponent(db.PollSettingsTypeShowCount)
			_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return err
			}
		case "3":
			components := v.ChangeSettingsMenuComponent(db.PollSettingsTypeShowTotalCount)
			_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return err
			}
		case "4":
			components := v.ChangeSettingsMenuComponent(db.PollSettingsTypeShowUserInResult)
			_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return err
			}
		case "5":
			components := v.ChangeSettingsMenuComponent(db.PollSettingsTypeCanChangeTarget)
			_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return err
			}
		default:
			b.Logger.Warn("不明な選択")
			return nil
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentChangeSettings(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		switch args[4] {
		case "show-user":
			var val db.PollSettingShowType
			switch args[5] {
			case "0":
				val = db.PollSettingShowTypeAlways
			case "1":
				val = db.PollSettingShowTypeNever
			case "2":
				val = db.PollSettingShowTypeAfterVote
			}
			v.Settings.ShowUser = val
		case "show-count":
			var val db.PollSettingShowType
			switch args[5] {
			case "0":
				val = db.PollSettingShowTypeAlways
			case "1":
				val = db.PollSettingShowTypeNever
			case "2":
				val = db.PollSettingShowTypeAfterVote
			}
			v.Settings.ShowCount = val
		case "show-total-count":
			var val db.PollSettingsBool
			switch args[5] {
			case "true":
				val = true
			case "false":
				val = false
			}
			v.Settings.ShowTotalCount = val
		case "show-user-in-result":
			var val db.PollSettingsBool
			switch args[5] {
			case "true":
				val = true
			case "false":
				val = false
			}
			v.Settings.ShowUserInResult = val
		case "can-change-target":
			var val db.PollSettingsBool
			switch args[5] {
			case "true":
				val = true
			case "false":
				val = false
			}
			v.Settings.CanChangeTarget = val
		}
		embeds := v.EditSettingsEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := v.EditSettingsComponent()
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return err
		}
		if err := b.DB.PollCreate().Remove(id); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Set(id, v); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		return nil
	}
}

func pollComponentEditChoice(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		val := e.StringSelectMenuInteractionData().Values[0]
		choice, ok := v.Choices[val]
		if !ok {
			return fmt.Errorf("poll choice not found err")
		}
		embeds := v.EditChoiceEmbed(choice.ID)
		botlib.SetEmbedProperties(embeds)
		components := v.EditChoiceComponent(choice.ID)
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollCOmponentDeleteChoice(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		delete(v.Choices, args[4])
		if err := b.DB.PollCreate().Remove(id); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Set(id, v); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = pollComponentBackMenu(b)(e)
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentChangeChoiceInfo(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateModal(discord.ModalCreate{
			CustomID: e.Data.CustomID(),
			Title:    translate.Message(e.Locale(), "command_text_poll_create_modal_change_choice_info_title"),
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "name",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_label"),
						Placeholder: translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_placeholder"),
						Required:    true,
						MaxLength:   100,
						Value:       v.Choices[args[4]].Name,
					},
				},
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "description",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_label"),
						Placeholder: translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_placeholder"),
						Required:    false,
						MaxLength:   100,
						Value:       v.Choices[args[4]].Description,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func pollComponentChangeChoiceEmoji(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		embeds := []discord.Embed{
			{
				Title:       translate.Message(e.Locale(), "command_text_poll_create_modal_change_choice_emoji_title"),
				Description: translate.Message(e.Locale(), "command_text_poll_create_modal_change_choice_emoji_description"),
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		customID := uuid.NewString()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
		var remove func()
		var removeButton func()
		remove = b.Handler.AddMessage(handler.Message{
			UUID:      uuid.New(),
			ChannelID: e.ChannelID(),
			AuthorID:  &e.Member().User.ID,
			Check: func(ctx *events.MessageCreate) bool {
				b.Logger.Debug("check")
				return structs.Twemoji.MatchString(ctx.Message.Content)
			},
			Handler: func(event *events.MessageCreate) error {
				b.Logger.Debug("called message")
				emoji := botlib.ParseComponentEmoji(event.Message.Content)
				choice := v.Choices[args[4]]
				choice.Emoji = &emoji
				v.Choices[args[4]] = choice
				err := event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID)
				if err != nil {
					return err
				}
				// インタラクショントークンを取得
				token, err = b.DB.Interactions().Get(id)
				if err != nil {
					return err
				}

				defer cancel()

				embeds = []discord.Embed{}
				embeds = v.EditChoiceEmbed(choice.ID)
				botlib.SetEmbedProperties(embeds)
				components := v.EditChoiceComponent(choice.ID)
				_, err = event.Client().Rest().UpdateInteractionResponse(event.Client().ApplicationID(), token, discord.MessageUpdate{
					Embeds:     &embeds,
					Components: &components,
				})
				if err != nil {
					return err
				}
				if err := b.DB.PollCreate().Remove(id); err != nil {
					return err
				}
				if err := b.DB.PollCreate().Set(id, v); err != nil {
					return err
				}
				return nil
			},
		})
		removeButton = b.Handler.AddComponent(handler.Component{
			Name: fmt.Sprintf("poll-%s", customID),
			Handler: map[string]handler.ComponentHandler{
				"change-choice-emoji-cancel": func(event *events.ComponentInteractionCreate) error {
					b.Logger.Debug("called cancel button")
					defer cancel()
					// インタラクショントークンを取得
					token, err := b.DB.Interactions().Get(id)
					if err != nil {
						embeds := botlib.ErrorTraceEmbed(event.Locale(), err)
						if err := event.CreateMessage(discord.MessageCreate{
							Embeds: embeds,
							Flags:  discord.MessageFlagEphemeral,
						}); err != nil {
							return err
						}
						return err
					}

					choice := v.Choices[args[4]]

					embeds = []discord.Embed{}
					embeds = v.EditChoiceEmbed(choice.ID)
					botlib.SetEmbedProperties(embeds)
					components := v.EditChoiceComponent(choice.ID)
					_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
						Embeds:     &embeds,
						Components: &components,
					})
					if err != nil {
						embeds := botlib.ErrorTraceEmbed(event.Locale(), err)
						if err := event.CreateMessage(discord.MessageCreate{
							Embeds: embeds,
							Flags:  discord.MessageFlagEphemeral,
						}); err != nil {
							return err
						}
						return err
					}
					err = event.CreateMessage(discord.MessageCreate{})
					if err != nil {
						return err
					}
					return nil
				},
			},
		})
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
			Components: &[]discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.ButtonComponent{
						CustomID: fmt.Sprintf("handler:poll-%s:change-choice-emoji-cancel", customID),
						Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
						Emoji: &discord.ComponentEmoji{
							ID:   snowflake.ID(1082319994211270686),
							Name: "cancel",
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			b.Logger.Debug(err)
		}
		b.Logger.Debug("waiting context...")
		go func() {
			defer remove()
			defer removeButton()
			<-ctx.Done()
			b.Logger.Debug("resume")
		}()
		return nil
	}
}

func pollComponentBackMenu(b *botlib.Bot) func(e *events.ComponentInteractionCreate) error {
	return func(e *events.ComponentInteractionCreate) error {
		args := strings.Split(e.Data.CustomID(), ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			return err
		}
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		embeds := v.ConfigEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := v.Components()
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		err = e.CreateMessage(discord.MessageCreate{})
		if err != nil {
			return err
		}
		return nil
	}
}

func PollModal(b *botlib.Bot) handler.Modal {
	return handler.Modal{
		Name: "poll",
		Handler: map[string]handler.ModalHandler{
			"add-choice":         pollModalAddChoice(b),
			"change-choice-info": pollModalChangeChoiceInfo(b),
		},
	}
}

func pollModalAddChoice(b *botlib.Bot) func(*events.ModalSubmitInteractionCreate) error {
	return func(e *events.ModalSubmitInteractionCreate) error {
		// IDを取り出す
		args := strings.Split(e.Data.CustomID, ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		// データベースから取得
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		// インタラクショントークンを取得
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		choiceID := uuid.NewString()
		v.Choices[choiceID] = db.PollCreateChoice{
			ID:          choiceID,
			Name:        e.ModalSubmitInteraction.Data.Text("name"),
			Description: e.ModalSubmitInteraction.Data.Text("description"),
			Position:    len(v.Choices) + 1,
			Emoji: &discord.ComponentEmoji{
				Name: botlib.Number2Emoji(len(v.Choices) + 1),
			},
		}

		// 作成パネルを更新
		embeds := v.ConfigEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
		components := v.Components()
		_, err = e.Client().Rest().UpdateInteractionResponse(e.Client().ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Remove(id); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Set(id, v); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		// 追加内容をレスポンド
		embeds = []discord.Embed{}
		embeds = append(embeds, discord.Embed{
			Title: translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_response_title"),
			Fields: []discord.EmbedField{
				{
					Name:  translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_label"),
					Value: e.ModalSubmitInteraction.Data.Text("name"),
				},
			},
		})
		if description, ok := e.ModalSubmitInteraction.Data.OptText("description"); ok {
			embeds[0].Fields = append(embeds[0].Fields, discord.EmbedField{
				Name:  translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_label"),
				Value: description,
			})
		}
		botlib.SetEmbedProperties(embeds)
		if err := e.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
			Flags:  discord.MessageFlagEphemeral,
		}); err != nil {
			return err
		}
		// 3秒後に削除
		time.Sleep(time.Second * 3)
		if err := e.Client().Rest().DeleteInteractionResponse(e.Client().ApplicationID(), e.Token()); err != nil {
			return err
		}
		return nil
	}
}

func pollModalChangeChoiceInfo(b *botlib.Bot) func(*events.ModalSubmitInteractionCreate) error {
	return func(e *events.ModalSubmitInteractionCreate) error {
		// IDを取り出す
		args := strings.Split(e.Data.CustomID, ":")
		id, err := snowflake.Parse(args[3])
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		// データベースから取得
		v, err := b.DB.PollCreate().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		// インタラクショントークンを取得
		token, err := b.DB.Interactions().Get(id)
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		choiceID := args[4]
		choice := v.Choices[choiceID]
		choice.Name = e.ModalSubmitInteraction.Data.Text("name")
		choice.Description = e.ModalSubmitInteraction.Data.Text("description")
		v.Choices[choiceID] = choice

		embeds := v.EditChoiceEmbed(choice.ID)
		botlib.SetEmbedProperties(embeds)
		components := v.EditChoiceComponent(choice.ID)
		_, err = e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Remove(id); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}
		if err := b.DB.PollCreate().Set(id, v); err != nil {
			embeds := botlib.ErrorTraceEmbed(e.Locale(), err)
			if err := e.CreateMessage(discord.MessageCreate{
				Embeds: embeds,
				Flags:  discord.MessageFlagEphemeral,
			}); err != nil {
				return err
			}
			return err
		}

		// 追加内容をレスポンド
		embeds = []discord.Embed{}
		embeds = append(embeds, discord.Embed{
			Title: translate.Message(e.Locale(), "command_text_poll_create_modal_change_choice_info_component_response_title"),
			Fields: []discord.EmbedField{
				{
					Name:  translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_name_label"),
					Value: e.ModalSubmitInteraction.Data.Text("name"),
				},
			},
		})
		if description, ok := e.ModalSubmitInteraction.Data.OptText("description"); ok {
			embeds[0].Fields = append(embeds[0].Fields, discord.EmbedField{
				Name:  translate.Message(e.Locale(), "command_text_poll_create_modal_add_choice_component_description_label"),
				Value: description,
			})
		}
		botlib.SetEmbedProperties(embeds)
		if err := e.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
			Flags:  discord.MessageFlagEphemeral,
		}); err != nil {
			return err
		}
		// 3秒後に削除
		time.Sleep(time.Second * 3)
		if err := e.Client().Rest().DeleteInteractionResponse(e.Client().ApplicationID(), e.Token()); err != nil {
			return err
		}
		return nil
	}
}

func End(b *botlib.Bot, p db.Poll) {
	if p.Finished {
		return
	}
	time.Sleep(time.Until(time.Unix(p.EndAt, 0)))
	b.Logger.Debug("finish vote")
	p, err := b.DB.Poll().Get(p.ID)
	if err != nil {
		b.Logger.Error(err)
		return
	}
	embeds := p.MessageEmbed()
	embeds = botlib.SetEmbedProperties(embeds)
	components := p.MessageComponent()
	_, err = b.Client.Rest().UpdateMessage(p.ChannelID, p.MessageID, discord.MessageUpdate{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		b.Logger.Error(err)
		return
	}
	var ranking []discord.EmbedField
	choices := []db.PollChoice{}
	for _, pc := range p.Choices {
		choices = append(choices, pc)
	}
	sort.SliceStable(choices, func(i, j int) bool {
		return len(choices[i].Users) > len(choices[j].Users)
	})
	inline := true
	rank := 1
	for i, pc := range choices {
		if i > 0 && len(choices[i-1].Users) > len(pc.Users) {
			rank = i + 1
		}
		ranking = append(ranking, discord.EmbedField{
			Name:   fmt.Sprintf("%s %s %s", botlib.FormatComponentEmoji(*pc.Emoji), pc.Name, translate.Translate(p.Locale, "poll_message_result_title", map[string]any{"Rank": rank})),
			Value:  translate.Translate(p.Locale, "poll_message_result_description", map[string]any{"Count": len(pc.Users)}),
			Inline: &inline,
		})
	}
	embeds = []discord.Embed{
		{
			Title:       translate.Message(p.Locale, "poll_message_result_embed_title"),
			Description: fmt.Sprintf("**%s**\r%s", translate.Message(p.Locale, "joined_people"), translate.Translate(p.Locale, "people", map[string]any{"Count": len(p.Users)})),
			Fields:      ranking,
		},
	}
	_, err = b.Client.Rest().CreateMessage(p.ChannelID, discord.MessageCreate{
		Embeds: embeds,
		MessageReference: &discord.MessageReference{
			MessageID: &p.MessageID,
			ChannelID: &p.ChannelID,
			GuildID:   &p.GuildId,
		},
	})
	if err != nil {
		b.Logger.Error(err)
		return
	}
	p.Finished = true
	if err := b.DB.Poll().Remove(p.ID); err != nil {
		b.Logger.Error(err)
	}
	if err := b.DB.Poll().Set(p.ID, p); err != nil {
		b.Logger.Error(err)
	}
}
