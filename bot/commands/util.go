package commands

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Util(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "util",
			Description:  "utilities",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:                     "calc",
					Description:              "in discord calculator",
					DescriptionLocalizations: translate.MessageMap("util_calc_command_description", false),
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{
							Name:                     "ephemeral",
							Description:              "create calculator as ephemeral message",
							DescriptionLocalizations: translate.MessageMap("util_calc_command_ephemeral_option_description", false),
							Required:                 false,
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"calc": utilCommandCalcHandler(b),
		},
	}
}

func utilCommandCalcHandler(b *botlib.Bot[*client.Client]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		calc := db.NewCalc()
		err := b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Interactions().Set(calc.ID(), event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if event.SlashCommandInteractionData().Bool("ephemeral") {
			mes.Flags = mes.Flags.Add(discord.MessageFlagEphemeral)
		}
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func UtilCalcComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "utilcalc",
		Handler: map[string]handler.ComponentHandler{
			"num":  utilCalcComponentNumHandler(b),
			"act":  utilCalcComponentActHandler(b),
			"do":   utilCalcComponentDoHandler(b),
			"c":    utilCalcComponentCHandler(b),
			"ce":   utilCalcComponentCEHandler(b),
			"back": utilCalcComponentBackHandler(b),
		},
	}
}

func utilCalcComponentBackHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		calc.Back()
		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func utilCalcComponentCEHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		calc.CE()
		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func utilCalcComponentCHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		calc.Reset()
		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func utilCalcComponentDoHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		calc.Do()

		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func utilCalcComponentActHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		switch args[3] {
		case "plus":
			calc.Plus()
		case "minus":
			calc.Minus()
		case "multiple":
			calc.Multiple()
		case "divide":
			calc.Divide()
		}
		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func utilCalcComponentNumHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.Self.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		token, err := b.Self.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		switch args[3] {
		case "±":
			calc.InputDo() // TODO: 電卓に合わせる
			if strings.HasPrefix(calc.Input, "-") {
				calc.Input = strings.TrimPrefix(calc.Input, "-")
			} else {
				calc.Input = "-" + calc.Input
			}
		case ".":
			calc.InputDo()
			if strings.Count(calc.Input, ".") == 0 {
				calc.Input += "."
			}
		default:
			calc.InputDo()
			if calc.Input == "0" {
				calc.Input = ""
			}
			if calc.Input == "-0" {
				calc.Input = "-"
			}
			calc.Input += args[3]
		}
		mes, err := calc.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}
