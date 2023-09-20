package commands

import (
	"context"
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
		err := b.Self.DB.Calc().Set(context.TODO(), calc.ID(), calc)
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
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()

		calc.Value.Back()
		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func utilCalcComponentCEHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()

		calc.Value.CE()
		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func utilCalcComponentCHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()

		calc.Value.Reset()
		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func utilCalcComponentDoHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()

		calc.Value.Do()

		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func utilCalcComponentActHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()
		switch args[3] {
		case "plus":
			calc.Value.Plus()
		case "minus":
			calc.Value.Minus()
		case "multiple":
			calc.Value.Multiple()
		case "divide":
			calc.Value.Divide()
		}
		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func utilCalcComponentNumHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.Self.DB.Calc().Get(context.TODO(), calcID)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		defer calc.Close()
		switch args[3] {
		case "±":
			calc.Value.InputDo() // TODO: 電卓に合わせる
			if strings.HasPrefix(calc.Value.Input, "-") {
				calc.Value.Input = strings.TrimPrefix(calc.Value.Input, "-")
			} else {
				calc.Value.Input = "-" + calc.Value.Input
			}
		case ".":
			calc.Value.InputDo()
			if strings.Count(calc.Value.Input, ".") == 0 {
				calc.Value.Input += "."
			}
		default:
			calc.Value.InputDo()
			if calc.Value.Input == "0" {
				calc.Value.Input = ""
			}
			if calc.Value.Input == "-0" {
				calc.Value.Input = "-"
			}
			calc.Value.Input += args[3]
		}
		mes, err := calc.Value.Message(botlib.SetEmbedsProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := calc.Set(context.TODO()); err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.UpdateMessage(discord.MessageUpdate{
			Content:    &mes.Content,
			Components: &mes.Components,
		}); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}
