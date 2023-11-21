package role

import (
	"context"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/internal/discordutil"
)

func role_panel_place(ctx context.Context, panel *ent.RolePanel, place *ent.RolePanelPlaced, locale discord.Locale, client bot.Client, react bool) error {
	builder := rp_placed_message(panel, place, locale)
	if place.MessageID != nil {
		if _, err := client.Rest().UpdateMessage(place.ChannelID, *place.MessageID, builder.Update()); err != nil {
			return err
		}
		if place.Type == rolepanelplaced.TypeReaction && react {
			_ = client.Rest().RemoveAllReactions(place.ChannelID, *place.MessageID)
		}
	} else {
		m, err := client.Rest().CreateMessage(place.ChannelID, builder.Create())
		if err != nil {
			return err
		}
		*place = *place.Update().SetMessageID(m.ID).SaveX(ctx)
	}

	if place.Type == rolepanelplaced.TypeReaction && react {
		for i, r := range panel.Roles {
			if r.Emoji == nil {
				r.Emoji = &discord.ComponentEmoji{
					Name: discordutil.Index2Emoji(i),
				}
			}
			if err := client.Rest().AddReaction(place.ChannelID, *place.MessageID, discordutil.FormatComponentEmoji(*r.Emoji)); err != nil {
				return err
			}
		}
	}
	return nil
}
