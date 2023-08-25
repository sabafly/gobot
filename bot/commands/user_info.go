package commands

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func UserInfo(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.UserCommandCreate{
			Name:              "info",
			NameLocalizations: translate.MessageMap("user_info_command", false),
			DMPermission:      &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": userInfoUserCommandHandler(b),
		},
	}
}

func userInfoUserCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		b.Self.UserDataLock(event.UserCommandInteractionData().TargetID()).Lock()
		defer b.Self.UserDataLock(event.UserCommandInteractionData().TargetID()).Unlock()
		member := event.UserCommandInteractionData().TargetMember()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(member.User.Tag())
		embed.SetDescription(user_flags2emoji(member.User.PublicFlags))
		embed.SetThumbnail(member.EffectiveAvatarURL())
		var nick string
		if member.Nick != nil {
			nick = *member.Nick
		}
		embed.AddField("Nick", nick, false)
		var global_name string
		if member.User.GlobalName != nil {
			global_name = *member.User.GlobalName
		}
		embed.AddField("GlobalName", global_name, false)
		embed.AddField("Created At", discord.FormattedTimestampMention(member.CreatedAt().Unix(), discord.TimestampStyleLongDateTime), false)
		embed.AddField("Joined At", discord.FormattedTimestampMention(member.JoinedAt.Unix(), discord.TimestampStyleLongDateTime), false)
		var roles string
		for i, id := range member.RoleIDs {
			roles += fmt.Sprintf("%d. %s\r", i+1, discord.RoleMention(id))
		}
		embed.AddField("Roles", roles, false)
		var status string
		presence, ok := event.Client().Caches().Presence(*event.GuildID(), member.User.ID)
		if ok {
			status = fmt.Sprintf(
				"Desktop: %s\rMobile: %s\rWeb: %s",
				botlib.StatusString(presence.ClientStatus.Desktop),
				botlib.StatusString(presence.ClientStatus.Mobile),
				botlib.StatusString(presence.ClientStatus.Web),
			)
		}
		embed.AddField("Status "+botlib.StatusString(presence.Status), status, true)

		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		ud, err := b.Self.DB.UserData().Get(member.User.ID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		embed2 := discord.NewEmbedBuilder()
		embed2.SetTitle("User Info")
		if _, ok := gd.UserLevels[member.User.ID]; !ok {
			gd.UserLevels[member.User.ID] = db.NewGuildDataUserLevel()
		}
		embed2.AddField("Level", fmt.Sprintf("%s lv (%s xp)", gd.UserLevels[member.User.ID].Level().String(), humanize.SI(float64(gd.UserLevels[member.User.ID].Point.Int64()), "")), false)
		embed2.AddField("Message Count", fmt.Sprintf("%d", gd.UserLevels[member.User.ID].MessageCount), false)
		var birthday string
		if ud.BirthDay != [2]int{} {
			btime := time.Date(time.Now().Year(), time.Month(ud.BirthDay[0]), ud.BirthDay[1], 0, 0, 0, 0, time.Local)
			if btime.Before(time.Now()) {
				btime = time.Date(time.Now().Year()+1, time.Month(ud.BirthDay[0]), ud.BirthDay[1], 0, 0, 0, 0, time.Local)
			}
			birthday = fmt.Sprintf("%s (%s)", discord.FormattedTimestampMention(btime.Unix(), discord.TimestampStyleShortDate), discord.FormattedTimestampMention(btime.Unix(), discord.TimestampStyleRelative))
		}
		embed2.AddField("Birthday", birthday, false)

		message := discord.NewMessageCreateBuilder()
		message.SetEmbeds(embed.Build(), embed2.Build())
		message.Embeds = botlib.SetEmbedsProperties(message.Embeds)
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func user_flags2emoji(flags discord.UserFlags) string {
	var str string
	if flags.Has(discord.UserFlagTeamUser) {
		str += "<:TEAM_PSEUDO_USER:1133432677438062642>"
	}
	if flags.Has(discord.UserFlagActiveDeveloper) {
		str += "<:ACTIVE_DEVELOPER:1133431399895027854>"
	}
	if flags.Has(discord.UserFlagHouseBalance) {
		str += "<:HYPTESQUAD_ONLINE_HOUSE_1:1133432354099183717>"
	}
	if flags.Has(discord.UserFlagHouseBravery) {
		str += "<:HYPESQUAD_ONLINE_HOUSE_2:1133432075719020686>"
	}
	if flags.Has(discord.UserFlagHouseBrilliance) {
		str += "<:HYPESQUAD_ONLINE_HOUSE_3:1133432218551861258>"
	}
	if flags.Has(discord.UserFlagBugHunterLevel1) {
		str += "<:BUG_HUNTER_LEVEL_1:1133431932307390525>"
	}
	if flags.Has(discord.UserFlagBugHunterLevel2) {
		str += "<:BUG_HUNTER_LEVEL_2:1133432825668968518>"
	}
	if flags.Has(discord.UserFlagEarlySupporter) {
		str += "<:PREMIUM_EARLY_SUPPORTER:1133432535184052315>"
	}
	if flags.Has(discord.UserFlagEarlyVerifiedBotDeveloper) {
		str += "<:VERIFIED_DEVELOPER:1133433147451777035>"
	}
	if flags.Has(discord.UserFlagHypeSquadEvents) {
		str += "<:HYPESQUAD:1133431760936513627>"
	}
	if flags.Has(discord.UserFlagDiscordCertifiedModerator) {
		str += "<:CERTIFIED_MODERATOR:1133433285255639092>"
	}
	return str
}
