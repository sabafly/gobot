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
 */

package discordsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/i18n"
)

const (
	MaxComponentsPerMessage    = 40
	MaxTextDisplayRunesTotal   = 4000
	SafeTextDisplayRunesTotal  = 3800
	SafeReaderTextDisplayRunes = 3500
	SafeReaderMaxComponents    = 32

	MaxActionRowButtons       = 5
	MaxButtonLabelRunes       = 80
	MaxCustomIDRunes          = 100
	MaxButtonURLRunes         = 512
	MaxSelectOptions          = 25
	MaxSelectTextRunes        = 100
	MaxSelectPlaceholderRunes = 150
	MaxMediaGalleryItems      = 10
	MaxMediaDescriptionRunes  = 1024

	MaxModalTitleRunes       = 45
	MaxModalLabels           = 5
	MaxLabelTextRunes        = 45
	MaxLabelDescriptionRunes = 100
	MaxTextInputPlaceholder  = 100
	MaxTextInputLength       = 4000

	MaxGuildChannels       = 500
	MaxCategoryChannels    = 50
	MaxChannelNameRunes    = 100
	MaxTextTopicRunes      = 1024
	MaxForumTopicRunes     = 4096
	MaxCommandNameRunes    = 32
	MaxCommandDescRunes    = 100
	MaxAutocompleteChoices = 25

	JSONErrUnknownChannel       rest.JSONErrorCode = 10003
	JSONErrUnknownMessage       rest.JSONErrorCode = 10008
	JSONErrUnknownInteraction   rest.JSONErrorCode = 10062
	JSONErrOpeningDMsTooFast    rest.JSONErrorCode = 40003
	JSONErrInteractionAcked     rest.JSONErrorCode = 40060
	JSONErrForumTagRequired     rest.JSONErrorCode = 40067
	JSONErrMissingAccess        rest.JSONErrorCode = 50001
	JSONErrCannotDMUser         rest.JSONErrorCode = 50007
	JSONErrMissingPermissions   rest.JSONErrorCode = 50013
	JSONErrInvalidFormBody      rest.JSONErrorCode = 50035
	JSONErrThreadArchived       rest.JSONErrorCode = 50083
	JSONErrThreadLocked         rest.JSONErrorCode = 160005
	JSONErrMaxActiveThreads     rest.JSONErrorCode = 160006
	DefaultDebouncerDelay                          = 2500 * time.Millisecond
	DefaultDMCooldown                              = 3 * time.Minute
	DefaultInvalidReqWindow                        = 10 * time.Minute
	DefaultInvalidReqThreshold                     = 500
	DefaultMountCircuitFails                       = 3
	DefaultMountCircuitCooldown                    = 5 * time.Minute
)

// TruncateRunes truncates s to at most maxRunes UTF-8 runes.
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

// PayloadValidator enforces Discord API component, modal, and command constraints before REST calls.
type PayloadValidator struct{}

// ValidateAndClampMessageCreate verifies and normalizes a MessageCreate payload against Discord limits.
func (v PayloadValidator) ValidateAndClampMessageCreate(mc discord.MessageCreate) (discord.MessageCreate, error) {
	if mc.Flags.Has(discord.MessageFlagIsComponentsV2) {
		mc.Content = ""
		mc.Embeds = nil
		mc.StickerIDs = nil
		mc.Poll = nil

		validated, totalCount, totalTextRunes, err := validateLayoutComponents(mc.Components)
		if err != nil {
			return mc, err
		}
		if totalCount == 0 {
			return mc, errors.New("components v2 message must include at least 1 component")
		}
		if totalCount > MaxComponentsPerMessage {
			return mc, fmt.Errorf("components v2 total component count %d exceeds %d", totalCount, MaxComponentsPerMessage)
		}
		if totalTextRunes > MaxTextDisplayRunesTotal {
			return mc, fmt.Errorf("components v2 text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
		}
		mc.Components = validated
	}
	return mc, nil
}

// ValidateAndClampMessageUpdate verifies and normalizes a MessageUpdate payload against Discord limits.
func (v PayloadValidator) ValidateAndClampMessageUpdate(mu discord.MessageUpdate, isV2 bool) (discord.MessageUpdate, error) {
	if isV2 {
		flags := discord.MessageFlagIsComponentsV2
		if mu.Flags != nil {
			flags = mu.Flags.Add(discord.MessageFlagIsComponentsV2)
		}
		mu.Flags = &flags
		mu.Content = nil
		mu.Embeds = nil
		if mu.Components != nil {
			validated, totalCount, totalTextRunes, err := validateLayoutComponents(*mu.Components)
			if err != nil {
				return mu, err
			}
			if totalCount == 0 {
				return mu, errors.New("components v2 message must include at least 1 component")
			}
			if totalCount > MaxComponentsPerMessage {
				return mu, fmt.Errorf("components v2 total component count %d exceeds %d", totalCount, MaxComponentsPerMessage)
			}
			if totalTextRunes > MaxTextDisplayRunesTotal {
				return mu, fmt.Errorf("components v2 text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
			}
			mu.Components = &validated
		}
	}
	return mu, nil
}

func validateLayoutComponents(list []discord.LayoutComponent) ([]discord.LayoutComponent, int, int, error) {
	out := make([]discord.LayoutComponent, 0, len(list))
	totalCount := 0
	totalTextRunes := 0

	for _, comp := range list {
		if comp == nil {
			continue
		}
		switch c := comp.(type) {
		case discord.TextDisplayComponent:
			if strings.TrimSpace(c.Content) == "" {
				return nil, 0, 0, errors.New("empty text display component is not allowed")
			}
			totalCount++
			totalTextRunes += utf8.RuneCountInString(c.Content)
			out = append(out, c)

		case discord.SeparatorComponent:
			totalCount++
			out = append(out, c)

		case discord.MediaGalleryComponent:
			if len(c.Items) == 0 {
				continue
			}
			items := make([]discord.MediaGalleryItem, 0, min(len(c.Items), MaxMediaGalleryItems))
			for _, it := range c.Items {
				if len(items) >= MaxMediaGalleryItems {
					break
				}
				u, err := url.Parse(it.Media.URL)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					continue
				}
				it.Description = TruncateRunes(it.Description, MaxMediaDescriptionRunes)
				items = append(items, it)
			}
			if len(items) == 0 {
				continue
			}
			c.Items = items
			totalCount++
			out = append(out, c)

		case discord.ActionRowComponent:
			row, rowCount, err := validateActionRow(c)
			if err != nil {
				return nil, 0, 0, err
			}
			if rowCount == 0 {
				continue
			}
			totalCount += rowCount
			out = append(out, row)

		case discord.SectionComponent:
			sec, cnt, runes, err := validateSection(c)
			if err != nil {
				return nil, 0, 0, err
			}
			totalCount += cnt
			totalTextRunes += runes
			out = append(out, sec)

		case discord.ContainerComponent:
			cont, cnt, runes, err := validateContainer(c)
			if err != nil {
				return nil, 0, 0, err
			}
			if cnt == 0 {
				continue
			}
			totalCount += cnt
			totalTextRunes += runes
			out = append(out, cont)

		default:
			totalCount++
			out = append(out, comp)
		}
	}
	return out, totalCount, totalTextRunes, nil
}

func validateContainer(c discord.ContainerComponent) (discord.ContainerComponent, int, int, error) {
	subLayout := make([]discord.LayoutComponent, 0, len(c.Components))
	for _, sub := range c.Components {
		if lc, ok := sub.(discord.LayoutComponent); ok {
			subLayout = append(subLayout, lc)
		}
	}
	validated, subCount, subRunes, err := validateLayoutComponents(subLayout)
	if err != nil {
		return c, 0, 0, err
	}
	if subCount == 0 {
		return c, 0, 0, nil
	}
	children := make([]discord.ContainerSubComponent, 0, len(validated))
	for _, lc := range validated {
		if sc, ok := lc.(discord.ContainerSubComponent); ok {
			children = append(children, sc)
		}
	}
	c.Components = children
	return c, 1 + subCount, subRunes, nil
}

func validateSection(s discord.SectionComponent) (discord.SectionComponent, int, int, error) {
	if len(s.Components) == 0 || len(s.Components) > 3 {
		return s, 0, 0, fmt.Errorf("section must have 1-3 child components, got %d", len(s.Components))
	}
	if s.Accessory == nil {
		return s, 0, 0, errors.New("section accessory cannot be nil")
	}
	cnt := 1
	runes := 0
	for _, child := range s.Components {
		td, ok := child.(discord.TextDisplayComponent)
		if !ok {
			cnt++
			continue
		}
		if strings.TrimSpace(td.Content) == "" {
			return s, 0, 0, errors.New("section text display cannot be empty")
		}
		cnt++
		runes += utf8.RuneCountInString(td.Content)
	}
	if btn, ok := s.Accessory.(discord.ButtonComponent); ok {
		b, err := validateButton(btn)
		if err != nil {
			return s, 0, 0, err
		}
		s.Accessory = b
	}
	cnt++
	return s, cnt, runes, nil
}

func validateActionRow(row discord.ActionRowComponent) (discord.ActionRowComponent, int, error) {
	if len(row.Components) == 0 {
		return row, 0, nil
	}
	out := make([]discord.InteractiveComponent, 0, len(row.Components))
	for _, ic := range row.Components {
		switch item := ic.(type) {
		case discord.ButtonComponent:
			if len(out) >= MaxActionRowButtons {
				break
			}
			b, err := validateButton(item)
			if err != nil {
				return row, 0, err
			}
			out = append(out, b)

		case discord.StringSelectMenuComponent:
			if len(row.Components) > 1 {
				return row, 0, errors.New("select menu must be the only component in an action row")
			}
			sel, err := validateStringSelect(item, false)
			if err != nil {
				return row, 0, err
			}
			out = append(out, sel)

		default:
			out = append(out, ic)
		}
	}
	row.Components = out
	return row, 1 + len(out), nil
}

func validateButton(b discord.ButtonComponent) (discord.ButtonComponent, error) {
	if b.Style != discord.ButtonStylePremium {
		b.Label = TruncateRunes(strings.TrimSpace(b.Label), MaxButtonLabelRunes)
		if b.Label == "" && b.Emoji == nil {
			return b, errors.New("button must have a non-empty label")
		}
	}
	if b.Style == discord.ButtonStyleLink {
		b.CustomID = ""
		u, err := url.Parse(b.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return b, fmt.Errorf("invalid link button URL: %q", b.URL)
		}
		if utf8.RuneCountInString(b.URL) > MaxButtonURLRunes {
			return b, fmt.Errorf("link button URL exceeds %d chars", MaxButtonURLRunes)
		}
	} else if b.Style != discord.ButtonStylePremium {
		b.URL = ""
		if b.CustomID == "" || utf8.RuneCountInString(b.CustomID) > MaxCustomIDRunes {
			return b, fmt.Errorf("button custom_id length must be 1-%d chars: %q", MaxCustomIDRunes, b.CustomID)
		}
	}
	return b, nil
}

func validateStringSelect(sel discord.StringSelectMenuComponent, inModal bool) (discord.StringSelectMenuComponent, error) {
	if sel.CustomID == "" || utf8.RuneCountInString(sel.CustomID) > MaxCustomIDRunes {
		return sel, fmt.Errorf("select custom_id must be 1-%d chars", MaxCustomIDRunes)
	}
	if inModal && sel.Disabled {
		return sel, errors.New("disabled components are not allowed in modals")
	}
	if len(sel.Options) == 0 {
		return sel, errors.New("string select menu must have at least 1 option")
	}
	if len(sel.Options) > MaxSelectOptions {
		sel.Options = sel.Options[:MaxSelectOptions]
	}
	seenVal := make(map[string]struct{}, len(sel.Options))
	opts := make([]discord.StringSelectMenuOption, 0, len(sel.Options))
	for _, opt := range sel.Options {
		opt.Label = TruncateRunes(strings.TrimSpace(opt.Label), MaxSelectTextRunes)
		opt.Value = TruncateRunes(strings.TrimSpace(opt.Value), MaxSelectTextRunes)
		opt.Description = TruncateRunes(strings.TrimSpace(opt.Description), MaxSelectTextRunes)
		if opt.Label == "" || opt.Value == "" {
			return sel, errors.New("string select option label and value cannot be empty")
		}
		if _, dup := seenVal[opt.Value]; dup {
			return sel, fmt.Errorf("duplicate string select option value: %q", opt.Value)
		}
		seenVal[opt.Value] = struct{}{}
		opts = append(opts, opt)
	}
	sel.Options = opts
	sel.Placeholder = TruncateRunes(sel.Placeholder, MaxSelectPlaceholderRunes)
	if sel.MaxValues > len(opts) {
		sel.MaxValues = len(opts)
	}
	if sel.MinValues != nil && *sel.MinValues > len(opts) {
		v := len(opts)
		sel.MinValues = &v
	}
	return sel, nil
}

// ValidateModal checks Discord Modal constraints (<= 45 title/label, <= 5 labels, no disabled fields).
func (v PayloadValidator) ValidateModal(m discord.ModalCreate) (discord.ModalCreate, error) {
	m.Title = TruncateRunes(strings.TrimSpace(m.Title), MaxModalTitleRunes)
	if m.Title == "" {
		return m, errors.New("modal title cannot be empty")
	}
	if m.CustomID == "" || utf8.RuneCountInString(m.CustomID) > MaxCustomIDRunes {
		return m, fmt.Errorf("modal custom_id must be 1-%d chars", MaxCustomIDRunes)
	}
	if len(m.Components) == 0 || len(m.Components) > MaxModalLabels {
		return m, fmt.Errorf("modal must have 1-%d components, got %d", MaxModalLabels, len(m.Components))
	}

	out := make([]discord.LayoutComponent, 0, len(m.Components))
	totalTextRunes := 0
	for _, comp := range m.Components {
		switch c := comp.(type) {
		case discord.LabelComponent:
			c.Label = TruncateRunes(strings.TrimSpace(c.Label), MaxLabelTextRunes)
			if c.Label == "" {
				return m, errors.New("modal label text cannot be empty")
			}
			c.Description = TruncateRunes(strings.TrimSpace(c.Description), MaxLabelDescriptionRunes)
			switch sub := c.Component.(type) {
			case discord.TextInputComponent:
				if sub.CustomID == "" || utf8.RuneCountInString(sub.CustomID) > MaxCustomIDRunes {
					return m, fmt.Errorf("text input custom_id must be 1-%d chars", MaxCustomIDRunes)
				}
				sub.Placeholder = TruncateRunes(sub.Placeholder, MaxTextInputPlaceholder)
				if sub.MaxLength <= 0 || sub.MaxLength > MaxTextInputLength {
					sub.MaxLength = MaxTextInputLength
				}
				if sub.MinLength != nil {
					if *sub.MinLength < 0 || *sub.MinLength > sub.MaxLength {
						return m, fmt.Errorf("text input min_length %d invalid for max_length %d", *sub.MinLength, sub.MaxLength)
					}
				}
				if sub.Value != "" {
					valRunes := utf8.RuneCountInString(sub.Value)
					if valRunes > sub.MaxLength {
						sub.Value = TruncateRunes(sub.Value, sub.MaxLength)
					}
					if sub.MinLength != nil && utf8.RuneCountInString(sub.Value) < *sub.MinLength {
						return m, errors.New("text input pre-filled value is shorter than min_length")
					}
				}
				c.Component = sub
			case discord.StringSelectMenuComponent:
				sel, err := validateStringSelect(sub, true)
				if err != nil {
					return m, err
				}
				c.Component = sel
			case discord.UserSelectMenuComponent:
				if sub.Disabled {
					return m, errors.New("disabled components are not allowed in modals")
				}
			case discord.RoleSelectMenuComponent:
				if sub.Disabled {
					return m, errors.New("disabled components are not allowed in modals")
				}
			case discord.MentionableSelectMenuComponent:
				if sub.Disabled {
					return m, errors.New("disabled components are not allowed in modals")
				}
			case discord.ChannelSelectMenuComponent:
				if sub.Disabled {
					return m, errors.New("disabled components are not allowed in modals")
				}
			}
			out = append(out, c)
		case discord.TextDisplayComponent:
			if strings.TrimSpace(c.Content) == "" {
				return m, errors.New("modal text display cannot be empty")
			}
			totalTextRunes += utf8.RuneCountInString(c.Content)
			if totalTextRunes > MaxTextDisplayRunesTotal {
				return m, fmt.Errorf("modal text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
			}
			out = append(out, c)
		default:
			out = append(out, comp)
		}
	}
	m.Components = out
	return m, nil
}

// ValidateApplicationCommands checks command names, option names, and all description localizations.
func (v PayloadValidator) ValidateApplicationCommands(cmds []discord.ApplicationCommandCreate) error {
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case discord.SlashCommandCreate:
			if err := validateCmdName(c.Name); err != nil {
				return err
			}
			if err := validateDescAndLocalizations(c.Description, c.DescriptionLocalizations, c.Name); err != nil {
				return err
			}
			if err := validateOptions(c.Options); err != nil {
				return err
			}
		case discord.MessageCommandCreate:
			if c.Name == "" || utf8.RuneCountInString(c.Name) > MaxCommandNameRunes {
				return fmt.Errorf("context command name %q must be 1-%d chars", c.Name, MaxCommandNameRunes)
			}
		}
	}
	return nil
}

func validateCmdName(name string) error {
	if name == "" || utf8.RuneCountInString(name) > MaxCommandNameRunes {
		return fmt.Errorf("command name %q must be 1-%d chars", name, MaxCommandNameRunes)
	}
	if name != strings.ToLower(name) || strings.ContainsRune(name, ' ') {
		return fmt.Errorf("command name %q must be lowercase without spaces", name)
	}
	return nil
}

func validateDescAndLocalizations(desc string, locs map[discord.Locale]string, path string) error {
	if desc == "" || utf8.RuneCountInString(desc) > MaxCommandDescRunes {
		return fmt.Errorf("description for %q must be 1-%d chars", path, MaxCommandDescRunes)
	}
	for loc, text := range locs {
		if text == "" || utf8.RuneCountInString(text) > MaxCommandDescRunes {
			return fmt.Errorf("localized description (%s) for %q must be 1-%d chars (got %d)", loc, path, MaxCommandDescRunes, utf8.RuneCountInString(text))
		}
	}
	return nil
}

func validateOptions(opts []discord.ApplicationCommandOption) error {
	for _, opt := range opts {
		if err := validateCmdName(opt.OptionName()); err != nil {
			return err
		}
		if err := validateDescAndLocalizations(opt.OptionDescription(), nil, opt.OptionName()); err != nil {
			return err
		}
		switch o := opt.(type) {
		case discord.ApplicationCommandOptionSubCommand:
			if err := validateDescAndLocalizations(o.Description, o.DescriptionLocalizations, o.Name); err != nil {
				return err
			}
			if err := validateOptions(o.Options); err != nil {
				return err
			}
		case discord.ApplicationCommandOptionSubCommandGroup:
			if err := validateDescAndLocalizations(o.Description, o.DescriptionLocalizations, o.Name); err != nil {
				return err
			}
			for _, sub := range o.Options {
				if err := validateCmdName(sub.Name); err != nil {
					return err
				}
				if err := validateDescAndLocalizations(sub.Description, sub.DescriptionLocalizations, sub.Name); err != nil {
					return err
				}
				if err := validateOptions(sub.Options); err != nil {
					return err
				}
			}
		case discord.ApplicationCommandOptionString:
			if err := validateDescAndLocalizations(o.Description, o.DescriptionLocalizations, o.Name); err != nil {
				return err
			}
			if len(o.Choices) > MaxAutocompleteChoices {
				return fmt.Errorf("option %q choices exceed %d", o.Name, MaxAutocompleteChoices)
			}
		}
	}
	return nil
}

// ValidateAutocompleteChoices truncates choices to Discord's 25-item / 100-char bounds.
func ValidateAutocompleteChoices(choices []discord.AutocompleteChoice) []discord.AutocompleteChoice {
	if len(choices) > MaxAutocompleteChoices {
		choices = choices[:MaxAutocompleteChoices]
	}
	out := make([]discord.AutocompleteChoice, 0, len(choices))
	for _, c := range choices {
		switch ch := c.(type) {
		case discord.AutocompleteChoiceString:
			ch.Name = TruncateRunes(ch.Name, MaxSelectTextRunes)
			ch.Value = TruncateRunes(ch.Value, MaxSelectTextRunes)
			out = append(out, ch)
		default:
			out = append(out, c)
		}
	}
	return out
}

// GuildStateReader abstracts cached Discord guild/channel/role lookups for preflight checks and unit testing.
type GuildStateReader interface {
	GuildFeatures(guildID snowflake.ID) ([]discord.GuildFeature, bool)
	GuildChannelCount(guildID snowflake.ID) int
	CategoryChildCount(guildID, categoryID snowflake.ID) int
	BotGuildPermissions(guildID snowflake.ID) (discord.Permissions, bool)
	BotHighestRolePosition(guildID snowflake.ID) (int, bool)
	Role(guildID, roleID snowflake.ID) (discord.Role, bool)
}

type clientGuildStateReader struct {
	client *bot.Client
}

// NewClientGuildStateReader wraps a live disgo *bot.Client cache as a GuildStateReader.
func NewClientGuildStateReader(client *bot.Client) GuildStateReader {
	return &clientGuildStateReader{client: client}
}

func (r *clientGuildStateReader) GuildFeatures(guildID snowflake.ID) ([]discord.GuildFeature, bool) {
	if r.client == nil {
		return nil, false
	}
	g, ok := r.client.Caches.Guild(guildID)
	if !ok {
		return nil, false
	}
	return g.Features, true
}

func (r *clientGuildStateReader) GuildChannelCount(guildID snowflake.ID) int {
	if r.client == nil {
		return 0
	}
	cnt := 0
	for range r.client.Caches.ChannelsForGuild(guildID) {
		cnt++
	}
	return cnt
}

func (r *clientGuildStateReader) CategoryChildCount(guildID, categoryID snowflake.ID) int {
	if r.client == nil {
		return 0
	}
	cnt := 0
	for ch := range r.client.Caches.ChannelsForGuild(guildID) {
		if ch.ParentID() != nil && *ch.ParentID() == categoryID {
			cnt++
		}
	}
	return cnt
}

func (r *clientGuildStateReader) BotGuildPermissions(guildID snowflake.ID) (discord.Permissions, bool) {
	if r.client == nil {
		return 0, false
	}
	self, ok := r.client.Caches.SelfMember(guildID)
	if !ok {
		return 0, false
	}
	return r.client.Caches.MemberPermissions(self), true
}

func (r *clientGuildStateReader) BotHighestRolePosition(guildID snowflake.ID) (int, bool) {
	if r.client == nil {
		return 0, false
	}
	self, ok := r.client.Caches.SelfMember(guildID)
	if !ok {
		return 0, false
	}
	highest := 0
	for _, role := range r.client.Caches.MemberRoles(self) {
		if role.Position > highest {
			highest = role.Position
		}
	}
	return highest, true
}

func (r *clientGuildStateReader) Role(guildID, roleID snowflake.ID) (discord.Role, bool) {
	if r.client == nil {
		return discord.Role{}, false
	}
	return r.client.Caches.Role(guildID, roleID)
}

// ChannelCreatePlan is the preflight result for provisioning a board channel.
type ChannelCreatePlan struct {
	UseForum       bool
	FallbackReason string
	BotPermissions discord.Permissions
	ClampedName    string
	ClampedTopic   string
}

// PreflightChecker verifies guild limits, features, permissions, and role hierarchies before REST calls.
type PreflightChecker struct {
	state GuildStateReader
}

// NewPreflightChecker constructs a PreflightChecker backed by GuildStateReader.
func NewPreflightChecker(state GuildStateReader) *PreflightChecker {
	return &PreflightChecker{state: state}
}

var requiredBoardPermissions = []struct {
	perm discord.Permissions
	name string
}{
	{discord.PermissionManageChannels, "ManageChannels"},
	{discord.PermissionViewChannel, "ViewChannel"},
	{discord.PermissionSendMessages, "SendMessages"},
	{discord.PermissionSendMessagesInThreads, "SendMessagesInThreads"},
	{discord.PermissionManageThreads, "ManageThreads"},
	{discord.PermissionManageMessages, "ManageMessages"},
	{discord.PermissionReadMessageHistory, "ReadMessageHistory"},
}

type PreflightErrorKind string

const (
	PreflightErrEmptyChannelName     PreflightErrorKind = "empty_channel_name"
	PreflightErrGuildChannelLimit    PreflightErrorKind = "guild_channel_limit"
	PreflightErrCategoryChannelLimit PreflightErrorKind = "category_channel_limit"
	PreflightErrMissingBotPerms      PreflightErrorKind = "missing_bot_perms"
	PreflightErrForumRequiresTag     PreflightErrorKind = "forum_requires_tag"
	PreflightErrInvalidRoleID        PreflightErrorKind = "invalid_role_id"
	PreflightErrMissingManageRoles   PreflightErrorKind = "missing_manage_roles"
	PreflightErrManagedRole          PreflightErrorKind = "managed_role"
	PreflightErrRoleHierarchy        PreflightErrorKind = "role_hierarchy"
)

// PreflightError represents a localized preflight check failure before calling Discord REST APIs.
type PreflightError struct {
	Kind         PreflightErrorKind
	Current      int
	Max          int
	MissingPerms []string
	RoleName     string
	RolePos      int
	BotPos       int
}

func (e *PreflightError) Error() string {
	switch e.Kind {
	case PreflightErrEmptyChannelName:
		return "channel name cannot be empty"
	case PreflightErrGuildChannelLimit:
		return fmt.Sprintf("guild channel limit reached (%d/%d)", e.Current, e.Max)
	case PreflightErrCategoryChannelLimit:
		return fmt.Sprintf("category channel limit reached (%d/%d)", e.Current, e.Max)
	case PreflightErrMissingBotPerms:
		return fmt.Sprintf("missing required bot permissions: %s", strings.Join(e.MissingPerms, ", "))
	case PreflightErrForumRequiresTag:
		return "forum channel requires a tag but has no available tags"
	case PreflightErrInvalidRoleID:
		return "invalid viewer role id"
	case PreflightErrMissingManageRoles:
		return "bot lacks ManageRoles permission"
	case PreflightErrManagedRole:
		return "cannot assign managed integration role"
	case PreflightErrRoleHierarchy:
		return fmt.Sprintf("target role %q (pos %d) is not below bot highest role (pos %d)", e.RoleName, e.RolePos, e.BotPos)
	default:
		return string(e.Kind)
	}
}

// FormatPreflightError translates a PreflightError (or returns false if err is not a PreflightError).
func FormatPreflightError(locale discord.Locale, err error) (string, bool) {
	var pe *PreflightError
	if !errors.As(err, &pe) {
		return "", false
	}
	switch pe.Kind {
	case PreflightErrEmptyChannelName:
		return i18n.TranslateText(locale, "components.bbs.preflight.empty_channel_name"), true
	case PreflightErrGuildChannelLimit:
		return i18n.TranslateText(locale, "components.bbs.preflight.guild_channel_limit", map[string]any{
			"current": pe.Current,
			"max":     pe.Max,
		}), true
	case PreflightErrCategoryChannelLimit:
		return i18n.TranslateText(locale, "components.bbs.preflight.category_channel_limit", map[string]any{
			"current": pe.Current,
			"max":     pe.Max,
		}), true
	case PreflightErrMissingBotPerms:
		return i18n.TranslateText(locale, "components.bbs.preflight.missing_bot_perms", map[string]any{
			"perms": strings.Join(pe.MissingPerms, ", "),
		}), true
	case PreflightErrForumRequiresTag:
		return i18n.TranslateText(locale, "components.bbs.preflight.forum_requires_tag"), true
	case PreflightErrInvalidRoleID:
		return i18n.TranslateText(locale, "components.bbs.preflight.invalid_role_id"), true
	case PreflightErrMissingManageRoles:
		return i18n.TranslateText(locale, "components.bbs.preflight.missing_manage_roles"), true
	case PreflightErrManagedRole:
		return i18n.TranslateText(locale, "components.bbs.preflight.managed_role"), true
	case PreflightErrRoleHierarchy:
		return i18n.TranslateText(locale, "components.bbs.preflight.role_hierarchy", map[string]any{
			"role":     pe.RoleName,
			"role_pos": pe.RolePos,
			"bot_pos":  pe.BotPos,
		}), true
	default:
		return pe.Error(), true
	}
}

// CheckChannelCreate verifies guild/category channel caps, bot permissions, and Community requirement for Forum channels.
func (p *PreflightChecker) CheckChannelCreate(
	guildID snowflake.ID,
	name string,
	topic string,
	wantForum bool,
	parentCategoryID *snowflake.ID,
) (ChannelCreatePlan, error) {
	plan := ChannelCreatePlan{
		UseForum:    wantForum,
		ClampedName: TruncateRunes(strings.TrimSpace(name), MaxChannelNameRunes),
	}
	if plan.ClampedName == "" {
		return plan, &PreflightError{Kind: PreflightErrEmptyChannelName}
	}

	if p.state != nil {
		if total := p.state.GuildChannelCount(guildID); total >= MaxGuildChannels {
			return plan, &PreflightError{Kind: PreflightErrGuildChannelLimit, Current: total, Max: MaxGuildChannels}
		}
		if parentCategoryID != nil && *parentCategoryID != 0 {
			if children := p.state.CategoryChildCount(guildID, *parentCategoryID); children >= MaxCategoryChannels {
				return plan, &PreflightError{Kind: PreflightErrCategoryChannelLimit, Current: children, Max: MaxCategoryChannels}
			}
		}

		if wantForum {
			if features, ok := p.state.GuildFeatures(guildID); ok {
				hasCommunity := false
				for _, f := range features {
					if f == discord.GuildFeatureCommunity {
						hasCommunity = true
						break
					}
				}
				if !hasCommunity {
					plan.UseForum = false
					plan.FallbackReason = "community_required"
				}
			}
		}

		if perms, ok := p.state.BotGuildPermissions(guildID); ok {
			plan.BotPermissions = perms
			if !perms.Has(discord.PermissionAdministrator) {
				var missing []string
				for _, req := range requiredBoardPermissions {
					if !perms.Has(req.perm) {
						missing = append(missing, req.name)
					}
				}
				if !plan.UseForum && !perms.Has(discord.PermissionCreatePublicThreads) {
					missing = append(missing, "CreatePublicThreads")
				}
				if len(missing) > 0 {
					return plan, &PreflightError{Kind: PreflightErrMissingBotPerms, MissingPerms: missing}
				}
			}
		}
	}

	topicLimit := MaxTextTopicRunes
	if plan.UseForum {
		topicLimit = MaxForumTopicRunes
	}
	plan.ClampedTopic = TruncateRunes(strings.TrimSpace(topic), topicLimit)
	return plan, nil
}

// MaskPermissionOverwrites restricts overwrite Allow/Deny bits to permissions held by the bot
// unless the bot holds Administrator, preventing Discord 403 (50013) on channel creation.
func MaskPermissionOverwrites(botPerms discord.Permissions, overwrites []discord.PermissionOverwrite) []discord.PermissionOverwrite {
	if botPerms == 0 || botPerms.Has(discord.PermissionAdministrator) {
		return overwrites
	}
	out := make([]discord.PermissionOverwrite, 0, len(overwrites))
	for _, ow := range overwrites {
		switch o := ow.(type) {
		case discord.RolePermissionOverwrite:
			o.Allow &= botPerms
			o.Deny &= botPerms
			out = append(out, o)
		case discord.MemberPermissionOverwrite:
			o.Allow &= botPerms
			o.Deny &= botPerms
			out = append(out, o)
		default:
			out = append(out, ow)
		}
	}
	return out
}

// ResolveForumTags returns an applicable tag ID slice when the forum requires tags (40067/40066 prevention).
func ResolveForumTags(forumCh discord.GuildForumChannel) ([]snowflake.ID, error) {
	if !forumCh.Flags().Has(discord.ChannelFlagRequireTag) {
		return nil, nil
	}
	for _, tag := range forumCh.AvailableTags {
		if !tag.Moderated {
			return []snowflake.ID{tag.ID}, nil
		}
	}
	if len(forumCh.AvailableTags) > 0 {
		return []snowflake.ID{forumCh.AvailableTags[0].ID}, nil
	}
	return nil, &PreflightError{Kind: PreflightErrForumRequiresTag}
}

// CheckRoleAssign verifies bot ManageRoles permission and role hierarchy before AddMemberRole.
func (p *PreflightChecker) CheckRoleAssign(guildID, roleID snowflake.ID) error {
	if roleID == 0 || roleID == guildID {
		return &PreflightError{Kind: PreflightErrInvalidRoleID}
	}
	if p.state == nil {
		return nil
	}
	if perms, ok := p.state.BotGuildPermissions(guildID); ok {
		if !perms.Has(discord.PermissionAdministrator) && !perms.Has(discord.PermissionManageRoles) {
			return &PreflightError{Kind: PreflightErrMissingManageRoles}
		}
	}
	role, ok := p.state.Role(guildID, roleID)
	if !ok {
		return nil
	}
	if role.Managed {
		return &PreflightError{Kind: PreflightErrManagedRole}
	}
	if botPos, ok := p.state.BotHighestRolePosition(guildID); ok {
		if role.Position >= botPos {
			return &PreflightError{
				Kind:     PreflightErrRoleHierarchy,
				RoleName: role.Name,
				RolePos:  role.Position,
				BotPos:   botPos,
			}
		}
	}
	return nil
}

// IsJSONError returns true if err wraps a *rest.Error matching the Discord JSON error code.
func IsJSONError(err error, code rest.JSONErrorCode) bool {
	var re *rest.Error
	if errors.As(err, &re) {
		return re.Code == code
	}
	return false
}

// HTTPStatusCode extracts the HTTP status code from a *rest.Error, or 0 if not a REST error.
func HTTPStatusCode(err error) int {
	var re *rest.Error
	if errors.As(err, &re) && re.Response != nil {
		return re.Response.StatusCode
	}
	return 0
}

// IsSharedRateLimit returns true if the REST error is a 429 with X-RateLimit-Scope: shared.
func IsSharedRateLimit(err error) bool {
	var re *rest.Error
	if errors.As(err, &re) && re.Response != nil {
		return strings.EqualFold(re.Response.Header.Get("X-RateLimit-Scope"), "shared")
	}
	return false
}

// EnsureThreadUnarchived unarchives an auto-archived thread before attempting edits or deletes (50083 guard).
func EnsureThreadUnarchived(client *bot.Client, threadID snowflake.ID) error {
	_, err := ensureThreadUnarchivedState(client, threadID)
	return err
}

func ensureThreadUnarchivedState(client *bot.Client, threadID snowflake.ID) (bool, error) {
	if client == nil {
		return false, nil
	}
	if th, ok := client.Caches.GuildThread(threadID); ok {
		if !th.ThreadMetadata.Archived {
			return false, nil
		}
	}
	return forceUnarchiveThreadState(client, threadID)
}

func forceUnarchiveThreadState(client *bot.Client, threadID snowflake.ID) (bool, error) {
	ch, err := client.Rest.GetChannel(threadID)
	if err != nil {
		return false, err
	}
	th, ok := ch.(discord.GuildThread)
	if ok && !th.ThreadMetadata.Archived {
		return false, nil
	}
	wasArchived := ok && th.ThreadMetadata.Archived
	archivedFalse := false
	if gc, ok := ch.(discord.GuildChannel); ok && gc.ParentID() != nil {
		if parent, ok := client.Caches.Channel(*gc.ParentID()); ok && parent.Type() == discord.ChannelTypeGuildForum {
			_, err = client.Rest.UpdateChannel(threadID, discord.GuildPostUpdate{Archived: &archivedFalse})
			return wasArchived, err
		}
	}
	_, err = client.Rest.UpdateChannel(threadID, discord.GuildThreadUpdate{Archived: &archivedFalse})
	return wasArchived, err
}

// WithUnarchiveRetry runs fn and, on Discord 50083 (archived thread), unarchives once and retries.
func WithUnarchiveRetry(client *bot.Client, threadID snowflake.ID, fn func() error) error {
	_, err := WithUnarchiveRetryState(client, threadID, fn)
	return err
}

// WithUnarchiveRetryState runs fn, unarchiving on 50083 if needed, and reports whether the thread was previously archived.
func WithUnarchiveRetryState(client *bot.Client, threadID snowflake.ID, fn func() error) (bool, error) {
	wasArchived, err := ensureThreadUnarchivedState(client, threadID)
	if err != nil {
		slog.Debug("bbs unarchive check failed before action", "thread_id", threadID, "err", err)
	}
	err = fn()
	if err != nil && IsJSONError(err, JSONErrThreadArchived) && client != nil {
		uWasArch, uErr := forceUnarchiveThreadState(client, threadID)
		if uErr == nil {
			wasArchived = true
			_ = uWasArch
			return wasArchived, fn()
		}
	}
	return wasArchived, err
}

// ArchiveThreadOnly re-archives an unlocked Discord thread/forum post without locking it.
func ArchiveThreadOnly(client *bot.Client, threadID snowflake.ID, isForum bool) error {
	if client == nil {
		return nil
	}
	archived := true
	if isForum {
		_, err := client.Rest.UpdateChannel(threadID, discord.GuildPostUpdate{
			Archived: &archived,
		})
		return err
	}
	_, err := client.Rest.UpdateChannel(threadID, discord.GuildThreadUpdate{
		Archived: &archived,
	})
	return err
}

// ArchiveAndLockThread locks and archives a Discord thread/forum post simultaneously to free active thread slots.
func ArchiveAndLockThread(client *bot.Client, threadID snowflake.ID, isForum bool) error {
	if client == nil {
		return nil
	}
	return WithUnarchiveRetry(client, threadID, func() error {
		locked := true
		archived := true
		if isForum {
			_, err := client.Rest.UpdateChannel(threadID, discord.GuildPostUpdate{
				Locked:   &locked,
				Archived: &archived,
			})
			return err
		}
		_, err := client.Rest.UpdateChannel(threadID, discord.GuildThreadUpdate{
			Locked:   &locked,
			Archived: &archived,
		})
		return err
	})
}

// SafeResponder guards interaction replies against the 3s initial window, 15m token expiry, and double acks (40060/10062).
type SafeResponder struct {
	mu           sync.Mutex
	createdAt    time.Time
	acked        bool
	inflight     bool
	ackUncertain bool
	deferred     bool
	isModal      bool
	validator    PayloadValidator
}

// NewSafeResponder creates a SafeResponder anchored at interaction creation time.
func NewSafeResponder(createdAt time.Time) *SafeResponder {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return &SafeResponder{createdAt: createdAt}
}

func (r *SafeResponder) canInitialAck() bool {
	return !r.acked && time.Since(r.createdAt) < 3*time.Second
}

func (r *SafeResponder) tokenValid() bool {
	return time.Since(r.createdAt) < 15*time.Minute
}

func (r *SafeResponder) finishInitialAttempt(err error, markDeferred, markModal bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inflight = false
	if err == nil {
		r.acked = true
		r.deferred = markDeferred
		r.isModal = markModal
		return
	}
	if IsJSONError(err, JSONErrInteractionAcked) {
		r.acked = true
		return
	}
	status := HTTPStatusCode(err)
	if status == 0 || status >= 500 {
		r.ackUncertain = true
	}
}

// RespondModal validates and sends a modal only if not previously acknowledged and within 3 seconds.
func (r *SafeResponder) RespondModal(event interface {
	Modal(modalCreate discord.ModalCreate, opts ...rest.RequestOpt) error
}, modal discord.ModalCreate) error {
	validated, err := r.validator.ValidateModal(modal)
	if err != nil {
		return err
	}
	r.mu.Lock()
	if r.inflight || r.ackUncertain || !r.canInitialAck() {
		r.mu.Unlock()
		return errors.New("interaction already acknowledged, uncertain, or past 3s initial window")
	}
	r.inflight = true
	r.mu.Unlock()
	err = event.Modal(validated)
	r.finishInitialAttempt(err, false, true)
	return err
}

// DeferCreate acknowledges with DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE (without IS_COMPONENTS_V2).
func (r *SafeResponder) DeferCreate(event interface {
	DeferCreateMessage(ephemeral bool, opts ...rest.RequestOpt) error
}, ephemeral bool) error {
	r.mu.Lock()
	if r.inflight || r.ackUncertain || !r.canInitialAck() {
		r.mu.Unlock()
		return nil
	}
	r.inflight = true
	r.mu.Unlock()
	err := event.DeferCreateMessage(ephemeral)
	r.finishInitialAttempt(err, true, false)
	return err
}

// RespondMessageCreate sends an initial or deferred interaction message safely.
func (r *SafeResponder) RespondMessageCreate(
	event interface {
		CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
		UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
		CreateFollowupMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) (*discord.Message, error)
	},
	mc discord.MessageCreate,
) error {
	validated, err := r.validator.ValidateAndClampMessageCreate(mc)
	if err != nil {
		return err
	}
	const (
		modeInitial = iota
		modeEditDeferred
		modeFollowup
	)
	r.mu.Lock()
	if r.inflight {
		r.mu.Unlock()
		return errors.New("interaction response already in flight")
	}
	var mode int
	switch {
	case r.deferred && !r.isModal:
		if !r.tokenValid() {
			r.mu.Unlock()
			return errors.New("interaction token expired (15m limit)")
		}
		mode = modeEditDeferred
	case r.acked:
		if !r.tokenValid() {
			r.mu.Unlock()
			return errors.New("interaction token expired (15m limit)")
		}
		mode = modeFollowup
	default:
		if r.ackUncertain {
			r.mu.Unlock()
			return errors.New("previous interaction acknowledgement outcome is uncertain")
		}
		if !r.canInitialAck() {
			r.mu.Unlock()
			return errors.New("interaction past 3s initial window")
		}
		mode = modeInitial
	}
	r.inflight = true
	r.mu.Unlock()

	switch mode {
	case modeEditDeferred:
		isV2 := validated.Flags.Has(discord.MessageFlagIsComponentsV2)
		mu := discord.MessageUpdate{
			Components:      &validated.Components,
			AllowedMentions: validated.AllowedMentions,
		}
		if !isV2 {
			c := validated.Content
			mu.Content = &c
			mu.Embeds = &validated.Embeds
		} else {
			flags := validated.Flags
			mu.Flags = &flags
		}
		mu, err = r.validator.ValidateAndClampMessageUpdate(mu, isV2)
		if err != nil {
			r.mu.Lock()
			r.inflight = false
			r.mu.Unlock()
			return err
		}
		_, err = event.UpdateInteractionResponse(mu)
		r.mu.Lock()
		r.inflight = false
		if err == nil {
			r.deferred = false
			r.acked = true
		}
		r.mu.Unlock()
		return err

	case modeFollowup:
		_, err = event.CreateFollowupMessage(validated)
		r.mu.Lock()
		r.inflight = false
		r.mu.Unlock()
		return err

	default:
		err = event.CreateMessage(validated)
		r.finishInitialAttempt(err, false, false)
		return err
	}
}

// UpdateComponentMessage updates the source component message or deferred response.
func (r *SafeResponder) UpdateComponentMessage(
	event interface {
		UpdateMessage(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) error
		UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
	},
	mu discord.MessageUpdate,
) error {
	validated, err := r.validator.ValidateAndClampMessageUpdate(mu, true)
	if err != nil {
		return err
	}
	r.mu.Lock()
	if r.inflight {
		r.mu.Unlock()
		return errors.New("interaction response already in flight")
	}
	wasDeferred := r.deferred && !r.isModal
	if wasDeferred {
		if !r.tokenValid() {
			r.mu.Unlock()
			return errors.New("interaction token expired (15m limit)")
		}
	} else {
		if r.ackUncertain || !r.canInitialAck() {
			r.mu.Unlock()
			return errors.New("interaction already acknowledged, uncertain, or past 3s initial window")
		}
	}
	r.inflight = true
	r.mu.Unlock()

	if wasDeferred {
		_, err = event.UpdateInteractionResponse(validated)
		r.mu.Lock()
		r.inflight = false
		if err == nil {
			r.deferred = false
			r.acked = true
		}
		r.mu.Unlock()
		return err
	}
	err = event.UpdateMessage(validated)
	r.finishInitialAttempt(err, false, false)
	return err
}

// SafeRespondComponentModal safely sends a modal from a ComponentInteractionCreate event.
func SafeRespondComponentModal(e *events.ComponentInteractionCreate, modal discord.ModalCreate) error {
	return NewSafeResponder(e.ID().Time()).RespondModal(e, modal)
}

type debounceEntry struct {
	seq   uint64
	timer *time.Timer
}

// ControlPanelDebouncer coalesces rapid floating control-panel moves per thread mirror.
type ControlPanelDebouncer struct {
	delay   time.Duration
	mu      sync.Mutex
	nextSeq uint64
	timers  map[uuid.UUID]debounceEntry
	closed  bool
}

// NewControlPanelDebouncer constructs a debouncer with delay (defaults to 2.5s when <= 0).
func NewControlPanelDebouncer(delay time.Duration) *ControlPanelDebouncer {
	if delay <= 0 {
		delay = DefaultDebouncerDelay
	}
	return &ControlPanelDebouncer{
		delay:  delay,
		timers: make(map[uuid.UUID]debounceEntry),
	}
}

// Schedule schedules or resets the debounced execution for mirrorID.
func (d *ControlPanelDebouncer) Schedule(mirrorID uuid.UUID, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	if existing, ok := d.timers[mirrorID]; ok {
		existing.timer.Stop()
	}
	d.nextSeq++
	seq := d.nextSeq
	t := time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		cur, ok := d.timers[mirrorID]
		if d.closed || !ok || cur.seq != seq {
			d.mu.Unlock()
			return
		}
		delete(d.timers, mirrorID)
		d.mu.Unlock()
		fn()
	})
	d.timers[mirrorID] = debounceEntry{seq: seq, timer: t}
}

// Stop cancels all pending panel reposition timers.
func (d *ControlPanelDebouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	for id, entry := range d.timers {
		entry.timer.Stop()
		delete(d.timers, id)
	}
}

// DMChannelCache caches DM channel IDs and rate-limits DM watch notifications per (thread, user).
type DMChannelCache struct {
	mu       sync.Mutex
	channels map[snowflake.ID]snowflake.ID
	lastSent map[string]time.Time
	cooldown time.Duration
}

// NewDMChannelCache constructs a DMChannelCache with a 3-minute per-thread cooldown by default.
func NewDMChannelCache(cooldown time.Duration) *DMChannelCache {
	if cooldown <= 0 {
		cooldown = DefaultDMCooldown
	}
	return &DMChannelCache{
		channels: make(map[snowflake.ID]snowflake.ID),
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
}

// GetOrCreateDMChannel returns a cached DM channel ID or opens one via REST.
func (c *DMChannelCache) GetOrCreateDMChannel(client *bot.Client, userID snowflake.ID) (snowflake.ID, error) {
	c.mu.Lock()
	chID, ok := c.channels[userID]
	c.mu.Unlock()
	if ok {
		return chID, nil
	}
	ch, err := client.Rest.CreateDMChannel(userID)
	if err != nil {
		return 0, err
	}
	c.mu.Lock()
	c.channels[userID] = ch.ID()
	c.mu.Unlock()
	return ch.ID(), nil
}

// AllowNotify checks and records the per-(threadID, userID) notification debounce window.
func (c *DMChannelCache) AllowNotify(threadID uuid.UUID, userID snowflake.ID, now time.Time) bool {
	key := fmt.Sprintf("%s:%d", threadID, userID)
	c.mu.Lock()
	defer c.mu.Unlock()
	if last, ok := c.lastSent[key]; ok && now.Sub(last) < c.cooldown {
		return false
	}
	c.lastSent[key] = now
	return true
}

// Invalidate removes cached DM channel ID on permanent error.
func (c *DMChannelCache) Invalidate(userID snowflake.ID) {
	c.mu.Lock()
	delete(c.channels, userID)
	c.mu.Unlock()
}

// InvalidRequestGuard tracks 401/403/429 responses (excluding shared 429s) over a 10-minute sliding window
// to stay well below Discord's 10,000 invalid requests / 10 min Cloudflare ban limit, and manages mount circuit breakers.
type InvalidRequestGuard struct {
	mu              sync.Mutex
	window          time.Duration
	threshold       int
	events          []time.Time
	mountFailures   map[uuid.UUID]int
	openMounts      map[uuid.UUID]time.Time
	halfOpenProbe   map[uuid.UUID]bool
	mountFailLimit  int
	circuitCooldown time.Duration
}

// NewInvalidRequestGuard creates a guard with threshold invalid requests per window and mount fail limit.
func NewInvalidRequestGuard(window time.Duration, threshold int, mountFailLimit int) *InvalidRequestGuard {
	if window <= 0 {
		window = DefaultInvalidReqWindow
	}
	if threshold <= 0 {
		threshold = DefaultInvalidReqThreshold
	}
	if mountFailLimit <= 0 {
		mountFailLimit = DefaultMountCircuitFails
	}
	return &InvalidRequestGuard{
		window:          window,
		threshold:       threshold,
		mountFailures:   make(map[uuid.UUID]int),
		openMounts:      make(map[uuid.UUID]time.Time),
		halfOpenProbe:   make(map[uuid.UUID]bool),
		mountFailLimit:  mountFailLimit,
		circuitCooldown: DefaultMountCircuitCooldown,
	}
}

// SetCircuitCooldown overrides the half-open recovery cooldown for mount circuit breakers.
func (g *InvalidRequestGuard) SetCircuitCooldown(d time.Duration) {
	if d <= 0 {
		return
	}
	g.mu.Lock()
	g.circuitCooldown = d
	g.mu.Unlock()
}

// RecordError inspects err and updates global invalid-request tracking and mount circuit breakers.
// Returns true if the mount should be permanently deactivated (e.g. 404 Unknown Channel).
func (g *InvalidRequestGuard) RecordError(mountID uuid.UUID, err error, now time.Time) (deactivateMount bool) {
	if err == nil {
		if mountID != uuid.Nil {
			g.RecordMountSuccess(mountID)
		}
		return false
	}
	status := HTTPStatusCode(err)
	if (status == 401 || status == 403 || status == 429) && !IsSharedRateLimit(err) {
		g.mu.Lock()
		g.pruneLocked(now)
		g.events = append(g.events, now)
		cnt := len(g.events)
		g.mu.Unlock()
		if cnt >= g.threshold {
			slog.Warn("bbs invalid-request safety threshold reached; throttling background sync", "count_10m", cnt, "threshold", g.threshold)
		}
	}

	if mountID != uuid.Nil {
		if status == 404 || IsJSONError(err, JSONErrUnknownChannel) {
			return true
		}
		if status == 403 || IsJSONError(err, JSONErrMissingAccess) || IsJSONError(err, JSONErrMissingPermissions) {
			g.mu.Lock()
			delete(g.halfOpenProbe, mountID)
			if _, alreadyOpen := g.openMounts[mountID]; alreadyOpen {
				g.openMounts[mountID] = now
			} else {
				g.mountFailures[mountID]++
				if g.mountFailures[mountID] >= g.mountFailLimit {
					g.openMounts[mountID] = now
				}
			}
			g.mu.Unlock()
		}
	}
	return false
}

// RecordMountSuccess resets failure counters and closes any open circuit breaker for mountID.
func (g *InvalidRequestGuard) RecordMountSuccess(mountID uuid.UUID) {
	g.mu.Lock()
	delete(g.mountFailures, mountID)
	delete(g.openMounts, mountID)
	delete(g.halfOpenProbe, mountID)
	g.mu.Unlock()
}

// IsMountOpen returns true if the circuit breaker for mountID is currently tripped (and cooldown has not elapsed).
func (g *InvalidRequestGuard) IsMountOpen(mountID uuid.UUID) bool {
	return g.IsMountOpenAt(mountID, time.Now())
}

// IsMountOpenAt evaluates mount circuit breaker state at now, permitting one half-open probe after cooldown.
func (g *InvalidRequestGuard) IsMountOpenAt(mountID uuid.UUID, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	openedAt, open := g.openMounts[mountID]
	if !open {
		return false
	}
	if g.circuitCooldown > 0 && now.Sub(openedAt) >= g.circuitCooldown && !g.halfOpenProbe[mountID] {
		g.halfOpenProbe[mountID] = true
		return false
	}
	return true
}

// IsThrottled returns true when invalid requests in the sliding window reached the safety threshold.
func (g *InvalidRequestGuard) IsThrottled(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pruneLocked(now)
	return len(g.events) >= g.threshold
}

func (g *InvalidRequestGuard) pruneLocked(now time.Time) {
	cutoff := now.Add(-g.window)
	idx := 0
	for idx < len(g.events) && g.events[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		g.events = append([]time.Time(nil), g.events[idx:]...)
	}
}

// HasFlaggedMediaScan inspects message components for Discord ContentScanMetadata policy flags.
func HasFlaggedMediaScan(msg discord.Message, channelNSFW bool) bool {
	for comp := range msg.AllComponents() {
		mg, ok := comp.(discord.MediaGalleryComponent)
		if !ok {
			continue
		}
		for _, item := range mg.Items {
			meta := item.Media.ContentScanMetadata
			if meta == nil {
				continue
			}
			if meta.Flags&discord.UnfurledMediaItemContentScanMetadataFlagIsGoreContent != 0 ||
				meta.Flags&discord.UnfurledMediaItemContentScanMetadataFlagIsSelfHarmContent != 0 {
				return true
			}
			if !channelNSFW && meta.Flags&discord.UnfurledMediaItemContentScanMetadataFlagIsExplicitContent != 0 {
				return true
			}
		}
	}
	return false
}

// SafeExecuteCtx wraps context cancellation checks before making background REST calls.
func SafeExecuteCtx(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}
