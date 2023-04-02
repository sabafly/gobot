package commands

import (
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/db"
	"github.com/sabafly/gobot/lib/handler"
)

func Util(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "util",
			Description:  "utilities",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "calc",
					Description: "in discord calculator",
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"calc": utilCommandCalcHandler(b),
		},
	}
}

func utilCommandCalcHandler(b *botlib.Bot) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		calc := db.NewCalc()
		err := b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Interactions().Set(calc.ID(), event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func UtilCalcComponent(b *botlib.Bot) handler.Component {
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

func utilCalcComponentBackHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		calc.Back()
		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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

func utilCalcComponentCEHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		calc.CE()
		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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

func utilCalcComponentCHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		calc.Reset()
		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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

func utilCalcComponentDoHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[3])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}

		calc.Do()

		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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

func utilCalcComponentActHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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
		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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

func utilCalcComponentNumHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		calcID := uuid.MustParse(args[4])
		calc, err := b.DB.Calc().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
		}
		token, err := b.DB.Interactions().Get(calcID)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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
		mes, err := calc.Message(botlib.SetEmbedProperties)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Calc().Set(calc.ID(), calc)
		if err != nil {
			return botlib.ReturnErrMessageEphemeral(event, "error_time_out", true)
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
