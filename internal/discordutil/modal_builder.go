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

package discordutil

import "github.com/disgoorg/disgo/discord"

func NewModalCreateBuilder() *ModalCreateBuilder {
	return &ModalCreateBuilder{}
}

type ModalCreateBuilder struct {
	discord.ModalCreate
}

func (b *ModalCreateBuilder) SetCustomID(customID string) *ModalCreateBuilder {
	b.CustomID = customID
	return b
}

func (b *ModalCreateBuilder) SetTitle(title string) *ModalCreateBuilder {
	b.Title = title
	return b
}

func (b *ModalCreateBuilder) SetComponents(components ...discord.LayoutComponent) *ModalCreateBuilder {
	b.Components = components
	return b
}

func (b *ModalCreateBuilder) SetComponent(i int, container discord.LayoutComponent) *ModalCreateBuilder {
	if len(b.Components) > i {
		b.Components[i] = container
	}
	return b
}

func (b *ModalCreateBuilder) AddActionRow(components ...discord.InteractiveComponent) *ModalCreateBuilder {
	b.Components = append(b.Components, discord.NewActionRow(components...))
	return b
}

func (b *ModalCreateBuilder) AddComponents(containers ...discord.LayoutComponent) *ModalCreateBuilder {
	b.Components = append(b.Components, containers...)
	return b
}

func (b *ModalCreateBuilder) ClearComponents() *ModalCreateBuilder {
	b.Components = []discord.LayoutComponent{}
	return b
}

func (b *ModalCreateBuilder) Build() discord.ModalCreate {
	return b.ModalCreate
}
