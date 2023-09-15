package commands

import (
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func RolePanel(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "role-panel",
			Description:  "summon role panels",
			DMPermission: &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": rolePanelHandler(b),
		},
	}
}

// TODO: V2に対応
func rolePanelHandler(b *botlib.Bot[*client.Client]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		mute := b.Self.DB.GuildData().Mu(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		options := []discord.StringSelectMenuOption{}
		for u, gdrp := range gData.RolePanel {
			if !gdrp.OnList {
				continue
			}
			rp, err := b.Self.DB.RolePanel().Get(u)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			options = append(options, discord.StringSelectMenuOption{
				Label:       rp.Name,
				Description: rp.Description,
				Value:       rp.UUID().String(),
			})
		}
		if len(options) == 0 {
			return botlib.ReturnErrMessage(event, "error_has_no_panel")
		}
		embeds := []discord.Embed{
			{
				Title: translate.Message(event.Locale(), "role_panel"),
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		err = event.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.StringSelectMenuComponent{
						CustomID: "handler:rolepanel:call",
						Options:  options,
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
