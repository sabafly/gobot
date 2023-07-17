package db

import (
	"context"
	"fmt"

	"github.com/Tnze/go-mc/chat"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

type MinecraftStatusPanelDB interface {
	Set(id uuid.UUID, panel MinecraftStatusPanel) error
	Get(id uuid.UUID) (MinecraftStatusPanel, error)
	Del(id uuid.UUID) error
}

type minecraftStatusPanelDBImpl struct {
	db *redis.Client
}

func (m minecraftStatusPanelDBImpl) Set(id uuid.UUID, panel MinecraftStatusPanel) error {
	buf, err := json.Marshal(panel)
	if err != nil {
		return err
	}
	res := m.db.HSet(context.TODO(), "mc-status-panel", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (m minecraftStatusPanelDBImpl) Get(id uuid.UUID) (MinecraftStatusPanel, error) {
	res := m.db.HGet(context.TODO(), "mc-status-panel", id.String())
	if err := res.Err(); err != nil {
		return MinecraftStatusPanel{}, err
	}
	var v MinecraftStatusPanel
	if err := json.Unmarshal([]byte(res.Val()), &v); err != nil {
		return MinecraftStatusPanel{}, err
	}
	return v, nil
}

func (m minecraftStatusPanelDBImpl) Del(id uuid.UUID) error {
	res := m.db.HDel(context.TODO(), "mc-status-panel", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewMinecraftStatusPanel(name string, guildID, channelID, messageID snowflake.ID, hash string, showAddress bool) MinecraftStatusPanel {
	return MinecraftStatusPanel{
		Name:          name,
		ID:            uuid.New(),
		GuildID:       guildID,
		ChannelID:     channelID,
		MessageID:     messageID,
		Hash:          hash,
		IsShowAddress: showAddress,
	}
}

type MinecraftStatusPanel struct {
	ID            uuid.UUID    `json:"id"`
	Name          string       `json:"name"`
	Hash          string       `json:"hash"`
	IsShowAddress bool         `json:"is_show_address"`
	GuildID       snowflake.ID `json:"guild_id"`
	ChannelID     snowflake.ID `json:"channel_id"`
	MessageID     snowflake.ID `json:"message_id"`
}

func (m MinecraftStatusPanel) Embed(address string, response *MinecraftPingResponse) discord.Embed {
	embed := discord.NewEmbedBuilder()
	embed.SetTitle(m.Name)
	embed.SetDescriptionf("```ansi\r%s ```", response.Description)
	embed.SetThumbnail("attachment://favicon.png")
	embed.AddFields(
		discord.EmbedField{
			Name:   "Players",
			Value:  fmt.Sprintf("(%d / %d)", response.Players.Online, response.Players.Max),
			Inline: json.Ptr(true),
		},
		discord.EmbedField{
			Name:   "Latency",
			Value:  "```disabled```",
			Inline: json.Ptr(true),
		},
		discord.EmbedField{
			Name:   "Version",
			Value:  fmt.Sprintf("```%s ```", chat.Text(response.Version.Name).String()),
			Inline: json.Ptr(true),
		},
		discord.EmbedField{
			Name:   "Edition",
			Value:  fmt.Sprintf("```%s ```", response.Type),
			Inline: json.Ptr(true),
		},
	)
	if len(response.Players.Sample) > 0 {
		for _, ips := range response.Players.Sample {
			embed.Fields[0].Value += "\r" + chat.Text(ips.Name).String()
		}
	}
	embed.Fields[0].Value = fmt.Sprintf("```ansi\r%s ```", embed.Fields[0].Value)
	if m.IsShowAddress {
		embed.AddFields(
			discord.EmbedField{
				Name:   "Address",
				Value:  fmt.Sprintf("```%s ```", address),
				Inline: json.Ptr(true),
			},
		)
	}
	embed.Embed = botlib.SetEmbedProperties(embed.Embed)
	return embed.Build()
}

func (m MinecraftStatusPanel) Components() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    "Refresh",
				CustomID: fmt.Sprintf("handler:minecraft:status-refresh:%s", m.ID.String()),
			},
		),
	}
}
