/*
	Copyright (C) 2022-2023  sabafly

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package api

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Model struct {
	ID        string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type MessageLog struct {
	Model
	GuildID   string
	ChannelID string
	UserID    string `gorm:"index"`
	Content   string
	Bot       bool
}

type GuildFeature struct {
	Model
	GuildID   string `gorm:"index"`
	TargetID  string
	FeatureID string
}

type VoteCreationMenu struct {
	gorm.Model
	Title       string
	Description string
	GuildID     string `gorm:"index"`
	CustomID    string
	Selections  []*VoteSelection
	ExpireAt    time.Time
	StartAt     time.Time
	Duration    time.Duration
}

type VoteSelection struct {
	Name        string
	Description string
	Emoji       discordgo.Emoji
}
