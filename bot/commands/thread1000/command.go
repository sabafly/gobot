package thread1000

import (
	"fmt"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/thread1000channel"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"slices"
)

func Command(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "thread1000",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "thread1000",
				Description: "manage thread1000",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "moderate",
						Description: "moderate thread1000",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "new",
								Description: "create thread1000 forum channel",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "name",
										Description: "name of the thread1000 forum channel",
										Required:    false,
										MaxLength:   builtin.Ptr(50),
									},
									discord.ApplicationCommandOptionString{
										Name:        "anonymous-name",
										Description: "name of the anonymous user",
										Required:    false,
										MaxLength:   builtin.Ptr(20),
									},
									discord.ApplicationCommandOptionChannel{
										Name:        "category",
										Description: "category of the thread1000 forum channel",
										Required:    false,
										ChannelTypes: []discord.ChannelType{
											discord.ChannelTypeGuildCategory,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/thread1000/moderate/new": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("thread1000.moderate.new"),
				},
				DiscordPerm: discord.PermissionManageGuild.Add(discord.PermissionManageChannels).Add(discord.PermissionManageRoles).Add(discord.PermissionManageWebhooks),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					dguild, ok := event.Guild()
					if !ok {
						g, err := event.Client().Rest().GetGuild(*event.GuildID(), false)
						if err != nil {
							return errors.NewError(err)
						}
						dguild = g.Guild
					}
					if !slices.Contains(dguild.Features, discord.GuildFeatureCommunity) {
						return errors.NewError(errors.ErrorMessage("component.thread1000.error.not_community", event))
					}
					// get guild
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					// check if exists thread1000 channel
					if c.DB().Thread1000Channel.Query().Where(thread1000channel.HasGuildWith(guild.ID(g.ID))).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("component.thread1000.error.already_exists", event))
					}
					// create thread1000 channel
					create := c.DB().Thread1000Channel.Create().
						SetGuild(g)
					name, ok := event.SlashCommandInteractionData().OptString("name")
					if ok {
						create.SetName(name)
					} else {
						name = "thread1000"
					}
					if anonymousName, ok := event.SlashCommandInteractionData().OptString("anonymous-name"); ok {
						create.SetAnonymousName(anonymousName)
					}
					ch, err := event.Client().Rest().CreateGuildChannel(*event.GuildID(), discord.GuildForumChannelCreate{
						Name:     name,
						ParentID: event.SlashCommandInteractionData().Channel("category").ID,
					})
					if err != nil {
						return errors.NewError(err)
					}
					create.SetChannelID(ch.ID())
					thread1000 := create.SaveX(event)
					if err := event.CreateMessage(discord.MessageCreate{
						Content: fmt.Sprintf("thread1000 channel created %v", thread1000.Name), // TODO: i18n
					}); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			switch event := event.(type) {
			case *events.ThreadCreate:
				if !c.DB().Thread1000Channel.Query().Where(thread1000channel.ChannelID(builtin.NonNil(event.Thread.ParentID()))).ExistX(event) {
					return nil
				}
				// get thread
				thread1000 := c.DB().Thread1000Channel.Query().Where(thread1000channel.ChannelID(builtin.NonNil(event.Thread.ParentID()))).OnlyX(event)
				thread, err := event.Client().Rest().CreateThread(thread1000.ChannelID, discord.GuildPublicThreadCreate{})
				if err != nil {
					return errors.NewError(err)
				}
				c.DB().Thread1000.Create().
					SetChannel(thread1000).
					SetThreadID(thread.ID())
			}
			return nil
		},
	}).
		SetComponent(c)
}
