package user

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "user",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "user",
				Description:              "Manage user settings",
				DescriptionLocalizations: i18n.TranslateTextMap("command.user.description"),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
					discord.InteractionContextTypeBotDM,
					discord.InteractionContextTypePrivateChannel,
				},
				IntegrationTypes: []discord.ApplicationIntegrationType{
					discord.ApplicationIntegrationTypeGuildInstall,
					discord.ApplicationIntegrationTypeUserInstall,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "settings",
						Description:              "View or modify your settings",
						DescriptionLocalizations: i18n.TranslateTextMap("command.user.settings.description"),
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/user/settings": generic.CommandHandler(handleUserSettings),
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"user:toggle_dm": generic.PComponentHandler{
				ComponentHandler: handleToggleDM,
			},
		},
	}).SetComponent(c)
}

func renderUserSettingsScreen(c *components.Components, locale discord.Locale, dbUser *models.User) ([]discord.LayoutComponent, error) {
	statusStr := i18n.TranslateText(locale, "general.state.disabled")
	btnLabel := i18n.TranslateText(locale, "command.user.settings.button.dm.enable")
	btnStyle := discord.ButtonStyleSuccess

	if dbUser.DMEnabled {
		statusStr = i18n.TranslateText(locale, "general.state.enabled")
		btnLabel = i18n.TranslateText(locale, "command.user.settings.button.dm.disable")
		btnStyle = discord.ButtonStyleDanger
	}

	l := i18n.TranslateLayout(locale, "command.user.settings.view")
	componentsList := i18n.BuildContext().
		WithText("dm_status", statusStr).
		WithText("dm_btn_label", btnLabel).
		WithButtonStyle("user:toggle_dm", btnStyle).
		Translate(l)

	return componentsList, nil
}

func handleUserSettings(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	locale := event.Locale()

	dbUser, err := database.GetOrCreateUser(c.GormDB(), event.User().ID)
	if err != nil {
		return errors.NewError(err)
	}

	componentsList, err := renderUserSettingsScreen(c, locale, dbUser)
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.CreateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(componentsList...).
		AddFlags(discord.MessageFlagEphemeral).
		BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func handleToggleDM(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	locale := event.Locale()

	// Update setting by toggling it
	var dbUser models.User
	err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.FirstOrCreate(&dbUser, models.User{ID: event.User().ID}).Error; err != nil {
			return err
		}
		dbUser.DMEnabled = !dbUser.DMEnabled
		return tx.Save(&dbUser).Error
	})
	if err != nil {
		slog.Error("failed to toggle user DM settings", "error", err, "userID", event.User().ID)
		return errors.NewError(err)
	}

	componentsList, err := renderUserSettingsScreen(c, locale, &dbUser)
	if err != nil {
		return errors.NewError(err)
	}

	// Update the message in place
	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(componentsList...).
		BuildUpdate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}
