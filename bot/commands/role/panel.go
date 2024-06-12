package role

import (
	"context"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/internal/errors"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/internal/discordutil"
)

func rolePanelPlace(ctx context.Context, place *ent.RolePanelPlaced, locale discord.Locale, client bot.Client, react bool) error {
	builder := rpPlacedMessage(place, locale)
	if place.MessageID != nil {
		if _, err := client.Rest().UpdateMessage(place.ChannelID, *place.MessageID, builder.BuildUpdate()); err != nil {
			return err
		}
		if place.Type == rolepanelplaced.TypeReaction && react {
			if err := client.Rest().RemoveAllReactions(place.ChannelID, *place.MessageID); err != nil {
				return err
			}
		}
	} else {
		m, err := client.Rest().CreateMessage(place.ChannelID, builder.BuildCreate())
		if err != nil {
			return err
		}
		*place = *place.Update().SetMessageID(m.ID).SaveX(ctx)
	}

	if place.Type == rolepanelplaced.TypeReaction && react {
		for i, r := range place.Roles {
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

func createPanelPlace(ctx context.Context, c *components.Components, panelID uuid.UUID, channelID snowflake.ID, g *ent.Guild) (*ent.RolePanelPlaced, error) {

	c.DB().RolePanelPlaced.Delete().Where(rolepanelplaced.And(rolepanelplaced.Or(rolepanelplaced.MessageIDIsNil(), rolepanelplaced.TypeIsNil()), rolepanelplaced.HasGuildWith(guild.ID(g.ID)))).ExecX(ctx)

	if !g.QueryRolePanels().Where(rolepanel.ID(panelID)).ExistX(ctx) {
		return nil, errors.New("rolepanel not found")
	}

	panel := g.QueryRolePanels().Where(rolepanel.ID(panelID)).FirstX(ctx)

	return c.DB().RolePanelPlaced.Create().
		SetGuild(g).
		SetChannelID(channelID).
		SetRolePanel(panel).
		SetName(panel.Name).
		SetDescription(panel.Description).
		SetRoles(panel.Roles).
		SetUpdatedAt(time.Now()).
		SaveX(ctx), nil
}
