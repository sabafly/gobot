package commands

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/db"
	"github.com/sabafly/gobot/lib/handler"
)

func Level(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:        "level",
			Description: "require manage guild permission",
		},
		Check: func(ctx *events.ApplicationCommandInteractionCreate) bool {
			if b.CheckDev(ctx.User().ID) {
				return true
			}
			permission := discord.PermissionManageGuild
			if member := ctx.Member(); member != nil && member.Permissions.Has(permission) {
				return true
			}
			_ = botlib.ReturnErrMessage(ctx, "error_no_permission", map[string]any{"Name": permission.String()})
			return false
		},
	}
}

func LevelMessage(b *botlib.Bot) handler.Message {
	return handler.Message{
		UUID: uuid.New(),
		Handler: func(event *events.MessageCreate) error {
			b.GuildDataMute.Lock()
			defer b.GuildDataMute.Unlock()
			gd, err := b.DB.GuildData().Get(*event.GuildID)
			if err != nil {
				return err
			}
			gd.Member[event.Message.Author.ID] = db.GuildDataMember{
				LastMessageID: event.MessageID,
				LastMessage:   time.Now(),
				LastVoice:     gd.Member[event.Message.Author.ID].LastVoice,
			}
			if err := b.DB.GuildData().Set(*event.GuildID, gd); err != nil {
				return err
			}
			return nil
		},
	}
}
