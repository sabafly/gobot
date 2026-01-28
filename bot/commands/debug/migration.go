package debug

import (
	"encoding/json"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	gormModels "github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/errors"
	"gorm.io/gorm"
)

func migrateEntToGormHandler(c *components.Components, entClient *ent.Client, event *events.ApplicationCommandInteractionCreate) errors.Error {
	if err := event.DeferCreateMessage(false); err != nil {
		return errors.NewError(err)
	}

	ctx := c.Ctx()
	gormDB := c.GormDB()

	if err := gormDB.Transaction(func(tx *gorm.DB) error {
		slog.Info("Starting migration...")

		// 1. Users
		slog.Info("Migrating Users...")
		users, err := entClient.User.Query().All(ctx)
		if err != nil {
			return err
		}
		for _, u := range users {
			gu := gormModels.User{
				ID:        u.ID,
				Name:      u.Name,
				CreatedAt: u.CreatedAt,
				Locale:    u.Locale,
				XP:        u.Xp,
			}
			if err := tx.Save(&gu).Error; err != nil {
				return err
			}
		}

		// 2. Guilds
		slog.Info("Migrating Guilds...")
		guilds, err := entClient.Guild.Query().WithOwner().All(ctx)
		if err != nil {
			return err
		}
		for _, g := range guilds {
			gg := gormModels.Guild{
				ID:                     g.ID,
				Name:                   g.Name,
				Locale:                 g.Locale,
				LevelUpMessage:         g.LevelUpMessage,
				LevelUpChannel:         g.LevelUpChannel,
				LevelUpExcludeChannel:  g.LevelUpExcludeChannel,
				LevelMee6Imported:      g.LevelMee6Imported,
				LevelRole:              g.LevelRole,
				Permissions:            g.Permissions,
				RemindCount:            g.RemindCount,
				RolePanelEditTimes:     g.RolePanelEditTimes,
				BumpEnabled:            g.BumpEnabled,
				BumpMessageTitle:       g.BumpMessageTitle,
				BumpMessage:            g.BumpMessage,
				BumpRemindMessageTitle: g.BumpRemindMessageTitle,
				BumpRemindMessage:      g.BumpRemindMessage,
				UpEnabled:              g.UpEnabled,
				UpMessageTitle:         g.UpMessageTitle,
				UpMessage:              g.UpMessage,
				UpRemindMessageTitle:   g.UpRemindMessageTitle,
				UpRemindMessage:        g.UpRemindMessage,
				BumpMention:            g.BumpMention,
				UpMention:              g.UpMention,
				LevelingDisabled:       g.LevelingDisabled,
				OwnerID:                g.Edges.Owner.ID,
			}
			if err := tx.Save(&gg).Error; err != nil {
				return err
			}
		}

		// 3. WordSuffix
		slog.Info("Migrating WordSuffix...")
		wordSuffixes, err := entClient.WordSuffix.Query().WithGuild().WithOwner().All(ctx)
		if err != nil {
			return err
		}
		for _, ws := range wordSuffixes {
			gws := gormModels.WordSuffix{
				ID:      ws.ID,
				Suffix:  ws.Suffix,
				Expired: ws.Expired,
				OwnerID: ws.Edges.Owner.ID,
				Rule:    string(ws.Rule),
			}
			if ws.Edges.Guild != nil {
				gws.GuildID = &ws.Edges.Guild.ID
			}
			if err := tx.Save(&gws).Error; err != nil {
				return err
			}
		}

		// 4. RolePanel Family
		slog.Info("Migrating RolePanel...")
		rolePanels, err := entClient.RolePanel.Query().WithGuild().All(ctx)
		if err != nil {
			return err
		}
		for _, rp := range rolePanels {
			grp := gormModels.RolePanel{
				ID:          rp.ID,
				Name:        rp.Name,
				Description: rp.Description,
				UpdatedAt:   rp.UpdatedAt,
				AppliedAt:   rp.AppliedAt,
				GuildID:     rp.Edges.Guild.ID,
			}
			grp.Roles = make([]gormModels.Role, len(rp.Roles))
			for i, r := range rp.Roles {
				grp.Roles[i] = gormModels.Role{ID: r.ID, Name: r.Name, Emoji: r.Emoji}
			}
			if err := tx.Save(&grp).Error; err != nil {
				return err
			}
		}

		slog.Info("Migrating RolePanelEdit...")
		rpEdits, err := entClient.RolePanelEdit.Query().WithGuild().WithParent().All(ctx)
		if err != nil {
			return err
		}
		for _, rpe := range rpEdits {
			grpe := gormModels.RolePanelEdit{
				ID:           rpe.ID,
				ChannelID:    rpe.ChannelID,
				EmojiAuthor:  rpe.EmojiAuthor,
				Token:        rpe.Token,
				SelectedRole: rpe.SelectedRole,
				Modified:     rpe.Modified,
				Name:         rpe.Name,
				Description:  rpe.Description,
				GuildID:      rpe.Edges.Guild.ID,
				ParentID:     rpe.Edges.Parent.ID,
			}
			grpe.Roles = make([]gormModels.Role, len(rpe.Roles))
			for i, r := range rpe.Roles {
				grpe.Roles[i] = gormModels.Role{ID: r.ID, Name: r.Name, Emoji: r.Emoji}
			}
			if err := tx.Save(&grpe).Error; err != nil {
				return err
			}
		}

		slog.Info("Migrating RolePanelPlaced...")
		rpPlaced, err := entClient.RolePanelPlaced.Query().WithGuild().WithRolePanel().All(ctx)
		if err != nil {
			return err
		}
		for _, rpp := range rpPlaced {
			grpp := gormModels.RolePanelPlaced{
				ID:                rpp.ID,
				MessageID:         rpp.MessageID,
				ChannelID:         rpp.ChannelID,
				Type:              string(rpp.Type),
				ButtonType:        rpp.ButtonType,
				ShowName:          rpp.ShowName,
				FoldingSelectMenu: rpp.FoldingSelectMenu,
				HideNotice:        rpp.HideNotice,
				UseDisplayName:    rpp.UseDisplayName,
				CreatedAt:         rpp.CreatedAt,
				Uses:              rpp.Uses,
				Name:              rpp.Name,
				Description:       rpp.Description,
				UpdatedAt:         rpp.UpdatedAt,
				GuildID:           rpp.Edges.Guild.ID,
				RolePanelID:       rpp.Edges.RolePanel.ID,
			}
			grpp.Roles = make([]gormModels.Role, len(rpp.Roles))
			for i, r := range rpp.Roles {
				grpp.Roles[i] = gormModels.Role{ID: r.ID, Name: r.Name, Emoji: r.Emoji}
			}
			if err := tx.Save(&grpp).Error; err != nil {
				return err
			}
		}

		// 5. MessagePin
		slog.Info("Migrating MessagePin...")
		pins, err := entClient.MessagePin.Query().WithGuild().All(ctx)
		if err != nil {
			return err
		}
		for _, p := range pins {
			gmp := gormModels.MessagePin{
				ID:        p.ID,
				GuildID:   p.Edges.Guild.ID,
				ChannelID: p.ChannelID,
				Content:   p.Content,
				Embeds:    p.Embeds,
				BeforeID:  p.BeforeID,
			}
			if rlData, err := json.Marshal(p.RateLimit); err == nil {
				var gormRL gormModels.RateLimit
				if err := json.Unmarshal(rlData, &gormRL); err == nil {
					gmp.RateLimit = gormRL
				}
			}
			if err := tx.Save(&gmp).Error; err != nil {
				return err
			}
		}

		// 6. MessageRemind
		slog.Info("Migrating MessageRemind...")
		reminds, err := entClient.MessageRemind.Query().WithGuild().All(ctx)
		if err != nil {
			return err
		}
		for _, r := range reminds {
			gmr := gormModels.MessageRemind{
				ID:        r.ID,
				GuildID:   r.Edges.Guild.ID,
				ChannelID: r.ChannelID,
				AuthorID:  r.AuthorID,
				Time:      r.Time,
				Content:   r.Content,
				Name:      r.Name,
			}
			if err := tx.Save(&gmr).Error; err != nil {
				return err
			}
		}

		// 7. Members
		slog.Info("Migrating Members...")
		members, err := entClient.Member.Query().WithGuild().WithUser().All(ctx)
		if err != nil {
			return err
		}
		for _, m := range members {
			gm := gormModels.Member{
				GuildID:           m.Edges.Guild.ID,
				UserID:            m.Edges.User.ID,
				Permission:        m.Permission,
				XP:                m.Xp,
				LastXP:            m.LastXp,
				MessageCount:      m.MessageCount,
				LastNotifiedLevel: m.LastNotifiedLevel,
				LastMessageHashes: m.LastMessageHashes,
			}
			if err := tx.Save(&gm).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		slog.Error("Migration failed", slog.Any("err", err))
		return errors.NewError(err)
	}

	if err := event.RespondMessage(
		discord.NewMessageBuilder().
			SetContent("Migration complete!"),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}
