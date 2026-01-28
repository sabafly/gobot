/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package level

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/smap"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/xppoint"
)

func requiredPointHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	mem, err := c.MemberCreate(event, event.User(), *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	levelNum := uint64(0)
	if l, ok := event.SlashCommandInteractionData().OptInt("level"); ok {
		levelNum = uint64(l)
	} else {
		levelNum = mem.XP.Level() + 1
	}
	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.required-point.embed.title", translate.WithTemplate(map[string]any{"Level": levelNum}))).
		SetDescriptionf("# `%d`xp\n%s\n%s",
			xppoint.TotalPoint(levelNum),
			translate.Message(event.Locale(), "components.level.required-point.embed.description", translate.WithTemplate(map[string]any{"User": event.Member().EffectiveName(), "Xp": mem.XP})),
			translate.Message(event.Locale(), "components.level.required-point.embed.description.diff", translate.WithTemplate(map[string]any{"Xp": builtin.Or(xppoint.TotalPoint(levelNum) > uint64(mem.XP), xppoint.TotalPoint(levelNum)-uint64(mem.XP), 0)})),
		).
		Build()

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func rankHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	target, ok := event.SlashCommandInteractionData().OptMember("target")
	if !ok {
		target = *event.Member()
	}
	if target.User.Bot || target.User.System {
		return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
	}
	m, err := c.MemberCreate(event, target.User, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	gl, err := c.GuildRequest(event.Client(), g.ID)
	if err != nil {
		return errors.NewError(err)
	}

	var members []models.Member
	if err := c.GormDB().Where("guild_id = ?", g.ID).Order("xp desc").Find(&members).Error; err != nil {
		return errors.NewError(err)
	}

	ids := make([]int, len(members))
	for i, mem := range members {
		ids[i] = mem.ID
	}
	index := slices.Index(ids, m.ID)

	embed := levelMessage(g, gl, m, index, target.Member, event)

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func leaderboardHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	const pageCount = 25
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	gl, err := c.GuildRequest(event.Client(), g.ID)
	if err != nil {
		return errors.NewError(err)
	}
	page := event.SlashCommandInteractionData().Int("page")
	if page < 1 {
		page = 1
	}

	var count int64
	c.GormDB().Model(&models.Member{}).Where("guild_id = ?", g.ID).Count(&count)

	if int64(page) > (count+pageCount-1)/pageCount {
		return errors.NewError(errors.ErrorMessage("errors.invalid.page", event))
	}

	var members []models.Member
	c.GormDB().Where("guild_id = ?", g.ID).Order("xp desc").Offset((page - 1) * pageCount).Limit(pageCount).Find(&members)

	var leaderboard string
	for i, m := range members {
		leaderboard += fmt.Sprintf("**#%d | %s XP: `%d` Level: `%d`**\n",
			i+1+((page-1)*pageCount),
			discord.UserMention(m.UserID),
			m.XP, m.XP.Level(),
		)
	}

	embed := discord.NewEmbedBuilder().
		SetEmbedAuthor(
			&discord.EmbedAuthor{
				Name:    g.Name,
				IconURL: builtin.NonNil(gl.IconURL()),
			},
		).
		SetTitlef("🏆%s(%d/%d)",
			translate.Message(event.Locale(), "components.level.leaderboard.title"),
			page,
			(count+pageCount-1)/pageCount,
		).
		SetDescription(leaderboard).
		Build()

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func transferHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	gl, err := c.GuildRequest(event.Client(), g.ID)
	if err != nil {
		return errors.NewError(err)
	}

	to := event.SlashCommandInteractionData().Member("to")
	from, ok := event.SlashCommandInteractionData().OptMember("from")
	if !ok {
		from = *event.Member()
	}
	if from.User.Bot || from.User.System || to.User.Bot || to.User.System {
		return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
	}
	if to.User.ID == from.User.ID {
		return errors.NewError(errors.ErrorMessage("errors.invalid.self.target", event))
	}

	fromUser, err := c.MemberCreate(event, from.User, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	toUser, err := c.MemberCreate(event, to.User, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	movedXp := uint64(fromUser.XP)
	fromUser.XP = xppoint.XP(0)
	fromUser.LastNotifiedLevel = nil
	c.GormDB().Save(fromUser)

	if toUser, err = addXp(event, movedXp, event.Client(), toUser, g, event.Channel().ID(), to.EffectiveName(), true, c); err != nil {
		return errors.NewError(err)
	}

	var ids []int
	c.GormDB().Model(&models.Member{}).Where("guild_id = ?", g.ID).Order("xp desc").Pluck("id", &ids)

	fromIndex, toIndex := slices.Index(ids, fromUser.ID), slices.Index(ids, toUser.ID)

	embedsList := []discord.Embed{
		levelMessage(g, gl, fromUser, fromIndex, from.Member, event),
		levelMessage(g, gl, toUser, toIndex, to.Member, event),
		discord.NewEmbedBuilder().SetTitlef("`%d`xp 移動しました", movedXp).Build(),
	}

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedsProperties(embedsList)...).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func upMessageHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	actionRow := discord.NewLabel(
		translate.Message(event.Locale(), "components.level.up.message.modal.input.message"),
		discord.TextInputComponent{
			CustomID:    "message",
			Style:       discord.TextInputStyleParagraph,
			MinLength:   builtin.Ptr(1),
			MaxLength:   140,
			Required:    true,
			Placeholder: translate.Message(event.Locale(), "components.level.up.message.modal.input.message.placeholder"),
			Value:       g.LevelUpMessage,
		},
	)

	if err := event.Modal(
		discord.NewModalCreateBuilder().
			SetTitle(translate.Message(event.Locale(), "components.level.up.message.modal.title")).
			SetCustomID("level:up_message_modal").
			SetComponents(actionRow).
			Build(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func upMessageChannelHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if channel, ok := event.SlashCommandInteractionData().OptChannel("channel"); ok {
		g.LevelUpChannel = &channel.ID
	} else {
		g.LevelUpChannel = nil
	}
	c.GormDB().Save(g)

	var channelText string
	if g.LevelUpChannel != nil {
		channelText = discord.ChannelMention(builtin.NonNil(g.LevelUpChannel))
	} else {
		channelText = translate.Message(event.Locale(), "components.level.up.message-channel.default")
	}

	content := translate.Message(event.Locale(), "components.level.up.message-channel.message",
		translate.WithTemplate(map[string]any{
			"Channel": channelText,
		}),
	)

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetContent(content).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func excludeChannelAddHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	channel := event.SlashCommandInteractionData().Channel("channel")
	if slices.Contains(g.LevelUpExcludeChannel, channel.ID) {
		return errors.NewError(errors.ErrorMessage("errors.already_exist", event))
	}
	g.LevelUpExcludeChannel = append(g.LevelUpExcludeChannel, channel.ID)
	c.GormDB().Save(g)

	content := translate.Message(event.Locale(), "components.level.exclude-channel.add.message",
		translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
	)

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetContent(content).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func excludeChannelRemoveHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	channel := event.SlashCommandInteractionData().Channel("channel")
	index := slices.Index(g.LevelUpExcludeChannel, channel.ID)
	if index == -1 {
		return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
	}
	g.LevelUpExcludeChannel = slices.Delete(g.LevelUpExcludeChannel, index, index+1)
	c.GormDB().Save(g)

	content := translate.Message(event.Locale(), "components.level.exclude-channel.remove.message",
		translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
	)

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetContent(content).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func excludeChannelClearHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	g.LevelUpExcludeChannel = []snowflake.ID{}
	c.GormDB().Save(g)

	content := translate.Message(event.Locale(), "components.level.exclude-channel.clear.message")

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetContent(content).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func excludeChannelListHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	var listStr string
	for i, id := range g.LevelUpExcludeChannel {
		listStr += fmt.Sprintf("%d. %s\n", i+1, discord.ChannelMention(id))
	}

	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.exclude-channel.list.message")).
		SetDescription(builtin.Or(listStr != "", listStr, "- "+translate.Message(event.Locale(), "components.level.exclude-channel.list.message.none"))).
		Build()

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func importMee6Handler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if g.LevelMee6Imported {
		return errors.NewError(errors.ErrorMessage("components.level.import-mee6.message.already", event))
	}

	var discordMembers []discord.Member
	afterID := snowflake.ID(0)
	for {
		m, err := event.Client().Rest.GetMembers(*event.GuildID(), 1000, afterID)
		if err != nil {
			return errors.NewError(err)
		}
		discordMembers = append(discordMembers, m...)
		if len(m) < 1000 {
			break
		}
		afterID = m[len(m)-1].User.ID
	}

	slog.Info("mee6インポート", slog.Any("gid", event.GuildID()), slog.Int("member_count", len(discordMembers)))

	importedCount := 0
	url := fmt.Sprintf("https://mee6.xyz/api/plugins/levels/leaderboard/%s", event.GuildID().String())
	for page := 0; true; page++ {
		response, err := http.Get(fmt.Sprintf("%s?page=%d", url, page))
		if err != nil || response.StatusCode != http.StatusOK {
			sc := 0
			if response != nil {
				sc = response.StatusCode
			}
			if sc == http.StatusUnauthorized {
				return errors.NewError(errors.ErrorMessage("components.level.import-mee6.message.unauthorized", event))
			}
			if importedCount > 0 {
				break
			}
			return errors.NewError(fmt.Errorf("mee6 API error: %d", sc))
		}
		var leaderboard struct {
			Players []struct {
				ID string `json:"id"`
				Xp int64  `json:"xp"`
			} `json:"players"`
		}
		if err := json.NewDecoder(response.Body).Decode(&leaderboard); err != nil {
			_ = response.Body.Close()
			return errors.NewError(err)
		}
		_ = response.Body.Close()
		if len(leaderboard.Players) < 1 {
			break
		}
		for _, player := range leaderboard.Players {
			pID, _ := snowflake.Parse(player.ID)
			idx := slices.IndexFunc(discordMembers, func(m discord.Member) bool { return m.User.ID == pID })
			if idx != -1 {
				m, _ := c.MemberCreate(event, discordMembers[idx].User, *event.GuildID())
				m.XP = xppoint.XP(player.Xp)
				c.GormDB().Save(m)
				importedCount++
			}
		}
	}

	g.LevelMee6Imported = true
	c.GormDB().Save(g)

	if err := event.RespondMessage(discord.NewMessageBuilder().SetContent(fmt.Sprintf("# SUCCEED\n```| IMPORTED MEMBER COUNT | %d```", importedCount))); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func resetHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	target := event.SlashCommandInteractionData().Member("target")
	m, err := c.MemberCreate(event, target.User, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	m.XP = xppoint.XP(0)
	c.GormDB().Save(m)

	content := translate.Message(event.Locale(), "components.level.reset.message",
		translate.WithTemplate(map[string]any{"User": discord.UserMention(target.User.ID)}),
	)

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetContent(content).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func roleSetHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if len(g.LevelRole) >= 20 {
		return errors.NewError(errors.ErrorMessage("errors.create.reach_max", event))
	}
	levelNum := event.SlashCommandInteractionData().Int("level")
	role := event.SlashCommandInteractionData().Role("role")
	g.LevelRole = builtin.NonNilMap(g.LevelRole)
	g.LevelRole[levelNum] = role.ID
	self, valid := event.Client().Caches.SelfMember(*event.GuildID())
	if !valid {
		return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
	}
	var roles []discord.Role
	for _, id := range self.RoleIDs {
		if r, ok := event.Client().Caches.Role(*event.GuildID(), id); ok {
			roles = append(roles, r)
		}
	}
	highestRole := discordutil.GetHighestRole(roles)
	if highestRole == nil || role.Managed || role.Compare(*highestRole) != -1 || role.ID == *event.GuildID() {
		return errors.NewError(errors.ErrorMessage("errors.invalid.role", event))
	}

	c.GormDB().Save(g)

	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.role.set.message.embed.title")).
		SetDescription(
			translate.Message(event.Locale(), "components.level.role.set.message.embed.description",
				translate.WithTemplate(map[string]any{
					"Level": strconv.Itoa(levelNum),
					"Role":  discord.RoleMention(role.ID),
				}),
			),
		).
		Build()

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func roleRemoveHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	g.LevelRole = builtin.NonNilMap(g.LevelRole)
	levelNum := event.SlashCommandInteractionData().Int("level")
	rID, ok := g.LevelRole[levelNum]
	if !ok {
		return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
	}
	delete(g.LevelRole, levelNum)

	c.GormDB().Save(g)

	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.role.remove.message.embed.title")).
		SetDescription(
			translate.Message(event.Locale(), "components.level.role.remove.message.embed.description",
				translate.WithTemplate(map[string]any{
					"Level": strconv.Itoa(levelNum),
					"Role":  discord.RoleMention(rID),
				}),
			),
		).
		Build()

	if err := event.CreateMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed)).BuildCreate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func roleListHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	g.LevelRole = builtin.NonNilMap(g.LevelRole)
	var listStr string
	for k, v := range smap.MakeSortMap(g.LevelRole).Iter(cmp.Compare[int]) {
		listStr += "- " + translate.Message(event.Locale(), "components.level.role.list.message", translate.WithTemplate(map[string]any{"Level": strconv.Itoa(k), "Role": discord.RoleMention(v)})) + "\n"
	}

	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.role.list.message.embed.title")).
		SetDescription(builtin.Or(listStr != "", listStr, translate.Message(event.Locale(), "components.level.role.list.message.none"))).
		Build()

	if err := event.RespondMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed))); err != nil {
		return errors.NewError(err)
	}
	return nil
}
