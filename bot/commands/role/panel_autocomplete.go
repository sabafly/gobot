package role

import (
	"fmt"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
)

func panelAutocomplete(c *components.Components, event *events.AutocompleteInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	panels := g.QueryRolePanels().Where(rolepanel.NameContains(event.Data.String("panel"))).AllX(event)
	choices := make([]discord.AutocompleteChoice, len(panels))
	for i, p := range panels {
		choices[i] = discord.AutocompleteChoiceString{
			Name:  builtin.Or(slices.ContainsFunc(panels, func(rp *ent.RolePanel) bool { return rp.ID != p.ID && rp.Name == p.Name }), fmt.Sprintf("%s (%s)", p.Name, p.ID), p.Name),
			Value: p.ID.String(),
		}
	}
	if err := event.AutocompleteResult(choices); err != nil {
		return errors.NewError(err)
	}
	return nil
}
