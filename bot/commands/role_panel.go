package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/bot"
	"github.com/sabafly/sabafly-lib/handler"
	"github.com/sabafly/sabafly-lib/translate"
)

func RolePanel(b *botlib.Bot[db.DB]) handler.Command {
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

func rolePanelHandler(b *botlib.Bot[db.DB]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		gData, err := b.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		options := []discord.StringSelectMenuOption{}
		for u, gdrp := range gData.RolePanel {
			if !gdrp.OnList {
				continue
			}
			rp, err := b.DB.RolePanel().Get(u)
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
