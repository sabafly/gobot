package models

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/internal/xppoint"
)

type User struct {
	ID        snowflake.ID   `gorm:"primary_key;column:id;type:bigint(20) unsigned;not null"`
	Name      string         `gorm:"not null"`
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
	Locale    discord.Locale `gorm:"default:'ja'"`
	XP        xppoint.XP     `gorm:"default:0"`
	DMEnabled bool           `gorm:"not null;default:true"`
}

func (u *User) SendDM(client *bot.Client, message discord.MessageCreate) (bool, error) {
	if !u.DMEnabled {
		slog.Info("DM skipped for user as DM is disabled", "userID", u.ID)
		return false, nil
	}
	if client == nil || client.Rest == nil {
		return false, fmt.Errorf("client or client.Rest is nil")
	}

	ch, err := client.Rest.CreateDMChannel(u.ID)
	if err != nil {
		return false, fmt.Errorf("failed to create DM channel: %w", err)
	}

	_, err = client.Rest.CreateMessage(ch.ID(), message)
	if err != nil {
		return false, fmt.Errorf("failed to send DM message: %w", err)
	}

	return true, nil
}
