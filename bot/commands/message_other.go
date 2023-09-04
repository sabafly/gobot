package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func MessageOther(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.MessageCommandCreate{
			Name:              "message other",
			NameLocalizations: translate.MessageMap("message_other_command_name", false),
			DMPermission:      &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": messageOtherHandler(b),
		},
	}
}

func messageOtherHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		result_message := discord.NewMessageCreateBuilder()
		message := event.MessageCommandInteractionData().TargetMessage()
		switch {
		case rolePanelConvertCheck(b, message):
			result_message.AddContainerComponents(
				discord.NewActionRow(
					discord.ButtonComponent{
						Style:    discord.ButtonStyleSuccess,
						Label:    translate.Message(event.Locale(), "message_other_panel_convert_button"),
						CustomID: fmt.Sprintf("handler:rp-v2:convert:%s:%s", event.Channel().ID(), message.ID),
					},
				),
			)
		default:
			embed := discord.NewEmbedBuilder()
			embed.SetTitle(translate.Message(event.Locale(), "message_other_command_not_eligible_title"))
			embed.SetDescription(translate.Message(event.Locale(), "message_other_command_not_eligible_description"))
			embed.Embed = botlib.SetEmbedProperties(embed.Embed)
			result_message.AddEmbeds(embed.Build())
		}
		result_message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.CreateMessage(result_message.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func rolePanelConvertCheck(b *botlib.Bot[*client.Client], message discord.Message) bool {
	var wid snowflake.ID
	if message.WebhookID != nil {
		wh, err := b.Client.Rest().GetWebhook(*message.WebhookID)
		if err != nil {
			b.Logger.Error(err)
		} else if wh.Type() == discord.WebhookTypeIncoming && wh.(discord.IncomingWebhook).User.ID == 716496407212589087 {
			wid = message.Author.ID
		}
	}
	switch message.Author.ID {
	case 895912135039803402, 1138119538190340146, 1137367652482957313, 971523089550671953 /*役職パネルv3*/, 917780792032251904 /*役職ボット*/, 669817785932578826 /*陽菜*/, 716496407212589087, wid /*RT*/, 718760319207473152 /*SevenBot*/, 832614051514417202 /*Glow-bot*/ :
		if len(message.Embeds) < 1 {
			return false
		}
		lines := strings.Split(message.Embeds[0].Description, "\r")
		valid_lines := 0
		for _, v := range lines {
			if !role_regexp.MatchString(v) {
				continue
			}
			valid_lines++
		}
		return valid_lines > 0
	default:
		return false
	}
}

var role_regexp = regexp.MustCompile("<@&([0-9]{18,20})>")
var role_id_regexp = regexp.MustCompile("[0-9]{18,20}")
