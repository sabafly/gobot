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

import (
	"fmt"
	"io"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func NewMessageBuilder() MessageBuilder {
	return &messageBuilderImpl{}
}

func NewMessageBuilderFromMessage(message discord.Message) MessageBuilder {
	attachments := make([]discord.AttachmentUpdate, len(message.Attachments))
	for i, attachment := range message.Attachments {
		attachments[i] = discord.AttachmentKeep{ID: attachment.ID}
	}
	builder := messageBuilderImpl{
		TTS:              message.TTS,
		MessageReference: message.MessageReference,
		Content:          &message.Content,
		Embeds:           &message.Embeds,
		Components:       &message.Components,
		Attachments:      &attachments,
		Flags:            &message.Flags,
	}
	return &builder
}

type MessageBuilder interface {
	SetContent(content string) MessageBuilder
	SetContentf(content string, a ...any) MessageBuilder
	SetTTS(tts bool) MessageBuilder
	SetEmbeds(embeds ...discord.Embed) MessageBuilder
	SetEmbed(i int, embed discord.Embed) MessageBuilder
	AddEmbeds(embeds ...discord.Embed) MessageBuilder
	ClearEmbeds() MessageBuilder
	RemoveEmbed(i int) MessageBuilder
	SetComponents(component ...discord.LayoutComponent) MessageBuilder
	SetComponent(i int, container discord.LayoutComponent) MessageBuilder
	AddActionRow(components ...discord.InteractiveComponent) MessageBuilder
	AddComponents(containers ...discord.LayoutComponent) MessageBuilder
	RemoveComponent(i int) MessageBuilder
	ClearComponents() MessageBuilder
	AddStickers(stickerIds ...snowflake.ID) MessageBuilder
	SetStickers(stickerIds ...snowflake.ID) MessageBuilder
	ClearStickers() MessageBuilder
	SetFiles(files ...*discord.File) MessageBuilder
	SetFile(i int, file *discord.File) MessageBuilder
	AddFiles(files ...*discord.File) MessageBuilder
	AddFile(name string, description string, reader io.Reader, flags ...discord.FileFlags) MessageBuilder
	ClearFiles() MessageBuilder
	RemoveFile(i int) MessageBuilder
	SetAllowedMentions(allowedMentions *discord.AllowedMentions) MessageBuilder
	ClearAllowedMentions() MessageBuilder
	SetMessageReference(messageReference *discord.MessageReference) MessageBuilder
	SetMessageReferenceByID(messageID snowflake.ID) MessageBuilder
	SetFlags(flags discord.MessageFlags) MessageBuilder
	AddFlags(flags ...discord.MessageFlags) MessageBuilder
	RemoveFlags(flags ...discord.MessageFlags) MessageBuilder
	ClearFlags() MessageBuilder
	SetEphemeral(ephemeral bool) MessageBuilder
	SetSuppressEmbeds(suppressEmbeds bool) MessageBuilder
	SetEnforceNonce(enforceNonce bool) MessageBuilder
	SetIsComponentsV2(isComponentsV2 bool) MessageBuilder

	ClearContent() MessageBuilder
	RetainAttachments(attachments ...discord.Attachment) MessageBuilder
	RetainAttachmentsByID(attachmentIDs ...snowflake.ID) MessageBuilder

	BuildCreate() discord.MessageCreate
	BuildUpdate() discord.MessageUpdate
	BuildWebhookCreate(username string, avatarURL string, threadName string) discord.WebhookMessageCreate
	BuildWebhookUpdate() discord.WebhookMessageUpdate

	messageBuilder()
}

type messageBuilderImpl struct {
	Nonce            string                      `json:"nonce,omitempty"`
	TTS              bool                        `json:"tts,omitempty"`
	StickerIDs       []snowflake.ID              `json:"sticker_ids,omitempty"`
	MessageReference *discord.MessageReference   `json:"message_reference,omitempty"`
	Content          *string                     `json:"content,omitempty"`
	Embeds           *[]discord.Embed            `json:"embeds,omitempty"`
	Components       *[]discord.LayoutComponent  `json:"components,omitempty"`
	Attachments      *[]discord.AttachmentUpdate `json:"attachments,omitempty"`
	Files            []*discord.File             `json:"-"`
	AllowedMentions  *discord.AllowedMentions    `json:"allowed_mentions,omitempty"`
	Flags            *discord.MessageFlags       `json:"flags,omitempty"`
	EnforceNonce     bool                        `json:"enforce_nonce,omitempty"`
}

func (m *messageBuilderImpl) SetEnforceNonce(enforceNonce bool) MessageBuilder {
	m.EnforceNonce = enforceNonce
	return m
}

func (m *messageBuilderImpl) SetContent(content string) MessageBuilder {
	m.Content = &content
	return m
}

func (m *messageBuilderImpl) SetContentf(content string, args ...any) MessageBuilder {
	s := fmt.Sprintf(content, args...)
	m.Content = &s
	return m
}

func (m *messageBuilderImpl) SetTTS(tts bool) MessageBuilder {
	m.TTS = tts
	return m
}

func (m *messageBuilderImpl) SetEmbeds(embeds ...discord.Embed) MessageBuilder {
	m.Embeds = &embeds
	return m
}

func (m *messageBuilderImpl) SetEmbed(i int, embed discord.Embed) MessageBuilder {
	if m.Embeds == nil {
		m.Embeds = &[]discord.Embed{}
	}
	if len(*m.Embeds) > i {
		(*m.Embeds)[i] = embed
	}
	return m
}

func (m *messageBuilderImpl) AddEmbeds(embeds ...discord.Embed) MessageBuilder {
	if m.Embeds == nil {
		m.Embeds = &[]discord.Embed{}
	}
	*m.Embeds = append(*m.Embeds, embeds...)
	return m
}

func (m *messageBuilderImpl) ClearEmbeds() MessageBuilder {
	return m.SetEmbeds()
}

func (m *messageBuilderImpl) RemoveEmbed(i int) MessageBuilder {
	if m.Embeds == nil {
		m.Embeds = &[]discord.Embed{}
	}
	*m.Embeds = append((*m.Embeds)[:i], (*m.Embeds)[i+1:]...)
	return m
}

func (m *messageBuilderImpl) SetComponents(components ...discord.LayoutComponent) MessageBuilder {
	m.Components = &components
	return m
}

func (m *messageBuilderImpl) SetComponent(i int, container discord.LayoutComponent) MessageBuilder {
	if m.Components == nil {
		m.Components = &[]discord.LayoutComponent{}
	}
	if len(*m.Components) > i {
		(*m.Components)[i] = container
	}
	return m
}

func (m *messageBuilderImpl) AddActionRow(components ...discord.InteractiveComponent) MessageBuilder {
	if m.Components == nil {
		m.Components = &[]discord.LayoutComponent{}
	}
	*m.Components = append(*m.Components, discord.NewActionRow(components...))
	return m
}

func (m *messageBuilderImpl) AddComponents(components ...discord.LayoutComponent) MessageBuilder {
	if m.Components == nil {
		m.Components = &[]discord.LayoutComponent{}
	}
	*m.Components = append(*m.Components, components...)
	return m
}

func (m *messageBuilderImpl) RemoveComponent(i int) MessageBuilder {
	if m.Components == nil {
		m.Components = &[]discord.LayoutComponent{}
	}
	*m.Components = append((*m.Components)[:i], (*m.Components)[i+1:]...)
	return m
}

func (m *messageBuilderImpl) ClearComponents() MessageBuilder {
	return m.SetComponents()
}

func (m *messageBuilderImpl) AddStickers(stickerIds ...snowflake.ID) MessageBuilder {
	m.StickerIDs = append(m.StickerIDs, stickerIds...)
	return m
}

func (m *messageBuilderImpl) SetStickers(stickerIds ...snowflake.ID) MessageBuilder {
	m.StickerIDs = stickerIds
	return m
}

func (m *messageBuilderImpl) ClearStickers() MessageBuilder {
	return m.SetStickers()
}

func (m *messageBuilderImpl) SetFiles(files ...*discord.File) MessageBuilder {
	m.Files = files
	return m
}

func (m *messageBuilderImpl) SetFile(i int, file *discord.File) MessageBuilder {
	if len(m.Files) > i {
		m.Files[i] = file
	}
	return m
}

func (m *messageBuilderImpl) AddFiles(files ...*discord.File) MessageBuilder {
	m.Files = append(m.Files, files...)
	return m
}

func (m *messageBuilderImpl) AddFile(name string, description string, reader io.Reader, flags ...discord.FileFlags) MessageBuilder {
	m.Files = append(m.Files, discord.NewFile(name, description, reader, flags...))
	return m
}

func (m *messageBuilderImpl) ClearFiles() MessageBuilder {
	return m.SetFiles()
}

func (m *messageBuilderImpl) RemoveFile(i int) MessageBuilder {
	if len(m.Files) > i {
		m.Files = append(m.Files[:i], m.Files[i+1:]...)
	}
	return m
}

func (m *messageBuilderImpl) SetAllowedMentions(allowedMentions *discord.AllowedMentions) MessageBuilder {
	m.AllowedMentions = allowedMentions
	return m
}

func (m *messageBuilderImpl) ClearAllowedMentions() MessageBuilder {
	return m.SetAllowedMentions(nil)
}

func (m *messageBuilderImpl) SetMessageReference(messageReference *discord.MessageReference) MessageBuilder {
	m.MessageReference = messageReference
	return m
}

func (m *messageBuilderImpl) SetMessageReferenceByID(messageID snowflake.ID) MessageBuilder {
	m.MessageReference = &discord.MessageReference{MessageID: &messageID}
	return m
}

func (m *messageBuilderImpl) SetFlags(flags discord.MessageFlags) MessageBuilder {
	m.Flags = &flags
	return m
}

func (m *messageBuilderImpl) AddFlags(flags ...discord.MessageFlags) MessageBuilder {
	if m.Flags == nil {
		m.Flags = new(discord.MessageFlags)
	}
	*m.Flags = m.Flags.Add(flags...)
	return m
}

func (m *messageBuilderImpl) RemoveFlags(flags ...discord.MessageFlags) MessageBuilder {
	if m.Flags == nil {
		m.Flags = new(discord.MessageFlags)
	}
	*m.Flags = m.Flags.Remove(flags...)
	return m
}

func (m *messageBuilderImpl) ClearFlags() MessageBuilder {
	return m.SetFlags(discord.MessageFlagsNone)
}

func (m *messageBuilderImpl) SetEphemeral(ephemeral bool) MessageBuilder {
	if m.Flags == nil {
		m.Flags = new(discord.MessageFlags)
	}
	if ephemeral {
		*m.Flags = m.Flags.Add(discord.MessageFlagEphemeral)
	} else {
		*m.Flags = m.Flags.Remove(discord.MessageFlagEphemeral)
	}
	return m
}

func (m *messageBuilderImpl) SetSuppressEmbeds(suppressEmbeds bool) MessageBuilder {
	if m.Flags == nil {
		m.Flags = new(discord.MessageFlags)
	}
	if suppressEmbeds {
		*m.Flags = m.Flags.Add(discord.MessageFlagSuppressEmbeds)
	} else {
		*m.Flags = m.Flags.Remove(discord.MessageFlagSuppressEmbeds)
	}
	return m
}

func (m *messageBuilderImpl) SetIsComponentsV2(isComponentsV2 bool) MessageBuilder {
	if m.Flags == nil {
		m.Flags = new(discord.MessageFlags)
	}
	if isComponentsV2 {
		*m.Flags = m.Flags.Add(discord.MessageFlagIsComponentsV2)
	} else {
		*m.Flags = m.Flags.Remove(discord.MessageFlagIsComponentsV2)
	}
	return m
}

func (m *messageBuilderImpl) ClearContent() MessageBuilder {
	return m.SetContent("")
}

func (m *messageBuilderImpl) RetainAttachments(attachments ...discord.Attachment) MessageBuilder {
	if m.Attachments == nil {
		m.Attachments = new([]discord.AttachmentUpdate)
	}
	for _, attachment := range attachments {
		*m.Attachments = append(*m.Attachments, discord.AttachmentKeep{ID: attachment.ID})
	}
	return m
}

func (m *messageBuilderImpl) RetainAttachmentsByID(attachmentIDs ...snowflake.ID) MessageBuilder {
	if m.Attachments == nil {
		m.Attachments = new([]discord.AttachmentUpdate)
	}
	for _, attachmentID := range attachmentIDs {
		*m.Attachments = append(*m.Attachments, discord.AttachmentKeep{ID: attachmentID})
	}
	return m
}

func nonNil[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

func parseAttachments(files []*discord.File) []discord.AttachmentCreate {
	var attachments []discord.AttachmentCreate
	for i, file := range files {
		isSpoiler := file.Flags.Has(discord.FileFlagSpoiler)
		if file.Description == "" && !isSpoiler {
			continue
		}
		attachment := discord.AttachmentCreate{
			ID:          i,
			Description: file.Description,
		}
		if isSpoiler {
			attachment.IsSpoiler = &isSpoiler
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}

func (m *messageBuilderImpl) BuildCreate() discord.MessageCreate {
	attachments := parseAttachments(m.Files)
	return discord.MessageCreate{
		Nonce:            m.Nonce,
		Content:          nonNil(m.Content),
		TTS:              m.TTS,
		Embeds:           nonNil(m.Embeds),
		Components:       nonNil(m.Components),
		StickerIDs:       m.StickerIDs,
		Files:            m.Files,
		Attachments:      attachments,
		AllowedMentions:  m.AllowedMentions,
		MessageReference: m.MessageReference,
		Flags:            nonNil(m.Flags),
		EnforceNonce:     m.EnforceNonce,
	}
}

func (m *messageBuilderImpl) BuildUpdate() discord.MessageUpdate {
	return discord.MessageUpdate{
		Content:         m.Content,
		Embeds:          m.Embeds,
		Components:      m.Components,
		Attachments:     m.Attachments,
		Files:           m.Files,
		AllowedMentions: m.AllowedMentions,
		Flags:           m.Flags,
	}
}

func (m *messageBuilderImpl) BuildWebhookCreate(username string, avatarURL string, threadName string) discord.WebhookMessageCreate {
	return discord.WebhookMessageCreate{
		Content:         nonNil(m.Content),
		Username:        username,
		AvatarURL:       avatarURL,
		TTS:             m.TTS,
		Embeds:          nonNil(m.Embeds),
		Components:      nonNil(m.Components),
		Files:           m.Files,
		AllowedMentions: m.AllowedMentions,
		Flags:           nonNil(m.Flags),
		ThreadName:      threadName,
	}
}

func (m *messageBuilderImpl) BuildWebhookUpdate() discord.WebhookMessageUpdate {
	return discord.WebhookMessageUpdate{
		Content:         m.Content,
		Embeds:          m.Embeds,
		Components:      m.Components,
		Attachments:     m.Attachments,
		Files:           m.Files,
		AllowedMentions: m.AllowedMentions,
	}
}

func (m *messageBuilderImpl) messageBuilder() {}
