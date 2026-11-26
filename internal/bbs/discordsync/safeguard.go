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

	MaxModalTitleRunes         = 45
	MaxModalLabels             = 5
	MaxLabelTextRunes          = 45
	MaxLabelDescriptionRunes   = 100
	MaxTextInputPlaceholder    = 100
	MaxTextInputLength         = 4000
	MaxCheckboxGroupOptions    = 10
	MaxCheckboxOptionTextRunes = 100

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

// ValidateMessageCreate strictly verifies a MessageCreate payload against Discord limits without mutating it.
func (v PayloadValidator) ValidateMessageCreate(mc discord.MessageCreate) error {
	if mc.Flags.Has(discord.MessageFlagIsComponentsV2) {
		if mc.Content != "" || len(mc.Embeds) > 0 || len(mc.StickerIDs) > 0 || mc.Poll != nil {
			return errors.New("components v2 message cannot include legacy content, embeds, stickers, or poll")
		}
		totalCount, totalTextRunes, err := validateLayoutComponents(mc.Components, make(map[string]struct{}))
		if err != nil {
			return err
		}
		if totalCount == 0 {
			return errors.New("components v2 message must include at least 1 component")
		}
		if totalCount > MaxComponentsPerMessage {
			return fmt.Errorf("components v2 total component count %d exceeds %d", totalCount, MaxComponentsPerMessage)
		}
		if totalTextRunes > MaxTextDisplayRunesTotal {
			return fmt.Errorf("components v2 text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
		}
		return nil
	}
	if utf8.RuneCountInString(mc.Content) > 2000 {
		return fmt.Errorf("message content length %d exceeds 2000", utf8.RuneCountInString(mc.Content))
	}
	return nil
}

// ValidateMessageUpdate strictly verifies a MessageUpdate payload against Discord limits without mutating it.
func (v PayloadValidator) ValidateMessageUpdate(mu discord.MessageUpdate, isV2 bool) error {
	if isV2 {
		if mu.Flags == nil || !mu.Flags.Has(discord.MessageFlagIsComponentsV2) {
			return errors.New("components v2 message update must include IS_COMPONENTS_V2 flag")
		}
		if mu.Content != nil || mu.Embeds != nil {
			return errors.New("components v2 message update cannot include legacy content or embeds")
		}
		if mu.Components != nil {
			totalCount, totalTextRunes, err := validateLayoutComponents(*mu.Components, make(map[string]struct{}))
			if err != nil {
				return err
			}
			if totalCount == 0 {
				return errors.New("components v2 message must include at least 1 component")
			}
			if totalCount > MaxComponentsPerMessage {
				return fmt.Errorf("components v2 total component count %d exceeds %d", totalCount, MaxComponentsPerMessage)
			}
			if totalTextRunes > MaxTextDisplayRunesTotal {
				return fmt.Errorf("components v2 text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
			}
		}
		return nil
	}
	if mu.Content != nil && utf8.RuneCountInString(*mu.Content) > 2000 {
		return fmt.Errorf("message content length %d exceeds 2000", utf8.RuneCountInString(*mu.Content))
	}
	return nil
}

func registerCustomID(seen map[string]struct{}, customID string) error {
	if customID == "" || seen == nil {
		return nil
	}
	if _, dup := seen[customID]; dup {
		return fmt.Errorf("duplicate component custom_id %q", customID)
	}
	seen[customID] = struct{}{}
	return nil
}

func validateLayoutComponents(list []discord.LayoutComponent, seen map[string]struct{}) (int, int, error) {
	totalCount := 0
	totalTextRunes := 0

	for _, comp := range list {
		if comp == nil {
			return 0, 0, errors.New("nil layout component is not allowed")
		}
		switch c := comp.(type) {
		case discord.TextDisplayComponent:
			if strings.TrimSpace(c.Content) == "" {
				return 0, 0, errors.New("empty text display component is not allowed")
			}
			totalCount++
			totalTextRunes += utf8.RuneCountInString(c.Content)

		case discord.SeparatorComponent:
			totalCount++

		case discord.MediaGalleryComponent:
			if len(c.Items) == 0 {
				return 0, 0, errors.New("empty media gallery component is not allowed")
			}
			if len(c.Items) > MaxMediaGalleryItems {
				return 0, 0, fmt.Errorf("media gallery items %d exceeds %d", len(c.Items), MaxMediaGalleryItems)
			}
			for _, it := range c.Items {
				u, err := url.Parse(it.Media.URL)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					return 0, 0, fmt.Errorf("invalid media gallery URL: %q", it.Media.URL)
				}
				if utf8.RuneCountInString(it.Description) > MaxMediaDescriptionRunes {
					return 0, 0, fmt.Errorf("media gallery description exceeds %d chars", MaxMediaDescriptionRunes)
				}
			}
			totalCount++

		case discord.ActionRowComponent:
			rowCount, err := validateActionRow(c, seen)
			if err != nil {
				return 0, 0, err
			}
			totalCount += rowCount

		case discord.SectionComponent:
			cnt, runes, err := validateSection(c, seen)
			if err != nil {
				return 0, 0, err
			}
			totalCount += cnt
			totalTextRunes += runes

		case discord.ContainerComponent:
			cnt, runes, err := validateContainer(c, seen)
			if err != nil {
				return 0, 0, err
			}
			totalCount += cnt
			totalTextRunes += runes

		default:
			totalCount++
		}
	}
	return totalCount, totalTextRunes, nil
}

func validateContainer(c discord.ContainerComponent, seen map[string]struct{}) (int, int, error) {
	if len(c.Components) == 0 {
		return 0, 0, errors.New("empty container component is not allowed")
	}
	subLayout := make([]discord.LayoutComponent, 0, len(c.Components))
	for _, sub := range c.Components {
		lc, ok := sub.(discord.LayoutComponent)
		if !ok {
			return 0, 0, errors.New("unsupported container subcomponent")
		}
		subLayout = append(subLayout, lc)
	}
	subCount, subRunes, err := validateLayoutComponents(subLayout, seen)
	if err != nil {
		return 0, 0, err
	}
	if subCount == 0 {
		return 0, 0, errors.New("empty container component is not allowed")
	}
	return 1 + subCount, subRunes, nil
}

func validateSection(s discord.SectionComponent, seen map[string]struct{}) (int, int, error) {
	if len(s.Components) == 0 || len(s.Components) > 3 {
		return 0, 0, fmt.Errorf("section must have 1-3 child components, got %d", len(s.Components))
	}
	if s.Accessory == nil {
		return 0, 0, errors.New("section accessory cannot be nil")
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
			return 0, 0, errors.New("section text display cannot be empty")
		}
		cnt++
		runes += utf8.RuneCountInString(td.Content)
	}
	if btn, ok := s.Accessory.(discord.ButtonComponent); ok {
		if err := validateButton(btn); err != nil {
			return 0, 0, err
		}
		if btn.Style != discord.ButtonStyleLink && btn.Style != discord.ButtonStylePremium {
			if err := registerCustomID(seen, btn.CustomID); err != nil {
				return 0, 0, err
			}
		}
	}
	cnt++
	return cnt, runes, nil
}

func validateActionRow(row discord.ActionRowComponent, seen map[string]struct{}) (int, error) {
	if len(row.Components) == 0 {
		return 0, errors.New("empty action row component is not allowed")
	}
	if len(row.Components) > MaxActionRowButtons {
		return 0, fmt.Errorf("action row components %d exceeds max %d", len(row.Components), MaxActionRowButtons)
	}
	for _, ic := range row.Components {
		switch item := ic.(type) {
		case discord.ButtonComponent:
			if err := validateButton(item); err != nil {
				return 0, err
			}
			if item.Style != discord.ButtonStyleLink && item.Style != discord.ButtonStylePremium {
				if err := registerCustomID(seen, item.CustomID); err != nil {
					return 0, err
				}
			}

		case discord.StringSelectMenuComponent:
			if len(row.Components) > 1 {
				return 0, errors.New("select menu must be the only component in an action row")
			}
			if err := validateStringSelect(item, false); err != nil {
				return 0, err
			}
			if err := registerCustomID(seen, item.CustomID); err != nil {
				return 0, err
			}

		default:
			if len(row.Components) > 1 {
				return 0, errors.New("select menu must be the only component in an action row")
			}
			if idComp, ok := ic.(interface{ GetCustomID() string }); ok {
				if err := registerCustomID(seen, idComp.GetCustomID()); err != nil {
					return 0, err
				}
			}
		}
	}
	return 1 + len(row.Components), nil
}

func validateButton(b discord.ButtonComponent) error {
	if b.Style != discord.ButtonStylePremium {
		if strings.TrimSpace(b.Label) == "" && b.Emoji == nil {
			return errors.New("button must have a non-empty label")
		}
		if utf8.RuneCountInString(b.Label) > MaxButtonLabelRunes {
			return fmt.Errorf("button label length exceeds %d chars", MaxButtonLabelRunes)
		}
	}
	if b.Style == discord.ButtonStyleLink {
		if b.CustomID != "" {
			return errors.New("link button cannot specify custom_id")
		}
		u, err := url.Parse(b.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid link button URL: %q", b.URL)
		}
		if utf8.RuneCountInString(b.URL) > MaxButtonURLRunes {
			return fmt.Errorf("link button URL exceeds %d chars", MaxButtonURLRunes)
		}
	} else if b.Style != discord.ButtonStylePremium {
		if b.URL != "" {
			return errors.New("non-link button cannot specify url")
		}
		if b.CustomID == "" || utf8.RuneCountInString(b.CustomID) > MaxCustomIDRunes {
			return fmt.Errorf("button custom_id length must be 1-%d chars: %q", MaxCustomIDRunes, b.CustomID)
		}
	}
	return nil
}

func validateStringSelect(sel discord.StringSelectMenuComponent, inModal bool) error {
	if sel.CustomID == "" || utf8.RuneCountInString(sel.CustomID) > MaxCustomIDRunes {
		return fmt.Errorf("select custom_id must be 1-%d chars", MaxCustomIDRunes)
	}
	if inModal && sel.Disabled {
		return errors.New("disabled components are not allowed in modals")
	}
	if len(sel.Options) == 0 || len(sel.Options) > MaxSelectOptions {
		return fmt.Errorf("string select menu must have 1-%d options, got %d", MaxSelectOptions, len(sel.Options))
	}
	seenVal := make(map[string]struct{}, len(sel.Options))
	for _, opt := range sel.Options {
		if strings.TrimSpace(opt.Label) == "" || strings.TrimSpace(opt.Value) == "" {
			return errors.New("string select option label and value cannot be empty")
		}
		if utf8.RuneCountInString(opt.Label) > MaxSelectTextRunes {
			return fmt.Errorf("string select option label exceeds %d chars", MaxSelectTextRunes)
		}
		if utf8.RuneCountInString(opt.Value) > MaxSelectTextRunes {
			return fmt.Errorf("string select option value exceeds %d chars", MaxSelectTextRunes)
		}
		if utf8.RuneCountInString(opt.Description) > MaxSelectTextRunes {
			return fmt.Errorf("string select option description exceeds %d chars", MaxSelectTextRunes)
		}
		if _, dup := seenVal[opt.Value]; dup {
			return fmt.Errorf("duplicate string select option value: %q", opt.Value)
		}
		seenVal[opt.Value] = struct{}{}
	}
	if utf8.RuneCountInString(sel.Placeholder) > MaxSelectPlaceholderRunes {
		return fmt.Errorf("string select placeholder exceeds %d chars", MaxSelectPlaceholderRunes)
	}
	if sel.MaxValues < 0 || sel.MaxValues > len(sel.Options) {
		return fmt.Errorf("string select max_values %d exceeds options len %d", sel.MaxValues, len(sel.Options))
	}
	if sel.MinValues != nil {
		if *sel.MinValues < 0 || *sel.MinValues > len(sel.Options) {
			return fmt.Errorf("string select min_values %d out of range", *sel.MinValues)
		}
		effectiveMax := sel.MaxValues
		if effectiveMax == 0 {
			effectiveMax = 1
		}
		if *sel.MinValues > effectiveMax {
			return fmt.Errorf("string select min_values %d exceeds max_values %d", *sel.MinValues, effectiveMax)
		}
	}
	return nil
}

func validateCheckboxGroup(cb discord.CheckboxGroupComponent) error {
	if cb.CustomID == "" || utf8.RuneCountInString(cb.CustomID) > MaxCustomIDRunes {
		return fmt.Errorf("checkbox group custom_id must be 1-%d chars", MaxCustomIDRunes)
	}
	if len(cb.Options) == 0 || len(cb.Options) > MaxCheckboxGroupOptions {
		return fmt.Errorf("checkbox group must have 1-%d options, got %d", MaxCheckboxGroupOptions, len(cb.Options))
	}
	seenVal := make(map[string]struct{}, len(cb.Options))
	for _, opt := range cb.Options {
		if strings.TrimSpace(opt.Label) == "" || strings.TrimSpace(opt.Value) == "" {
			return errors.New("checkbox group option label and value cannot be empty")
		}
		if utf8.RuneCountInString(opt.Label) > MaxCheckboxOptionTextRunes {
			return fmt.Errorf("checkbox group option label exceeds %d chars", MaxCheckboxOptionTextRunes)
		}
		if utf8.RuneCountInString(opt.Value) > MaxCheckboxOptionTextRunes {
			return fmt.Errorf("checkbox group option value exceeds %d chars", MaxCheckboxOptionTextRunes)
		}
		if utf8.RuneCountInString(opt.Description) > MaxCheckboxOptionTextRunes {
			return fmt.Errorf("checkbox group option description exceeds %d chars", MaxCheckboxOptionTextRunes)
		}
		if _, dup := seenVal[opt.Value]; dup {
			return fmt.Errorf("duplicate checkbox group option value: %q", opt.Value)
		}
		seenVal[opt.Value] = struct{}{}
	}
	if cb.MaxValues < 0 || cb.MaxValues > len(cb.Options) {
		return fmt.Errorf("checkbox group max_values %d exceeds options len %d", cb.MaxValues, len(cb.Options))
	}
	if cb.MinValues != nil {
		if *cb.MinValues < 0 || *cb.MinValues > len(cb.Options) {
			return fmt.Errorf("checkbox group min_values %d out of range", *cb.MinValues)
		}
		if cb.Required && *cb.MinValues < 1 {
			return errors.New("required checkbox group min_values must be at least 1")
		}
		effectiveMax := cb.MaxValues
		if effectiveMax == 0 {
			effectiveMax = len(cb.Options)
		}
		if *cb.MinValues > effectiveMax {
			return fmt.Errorf("checkbox group min_values %d exceeds max_values %d", *cb.MinValues, effectiveMax)
		}
	}
	return nil
}

// ValidateModal strictly verifies Discord Modal constraints without mutating the payload.
func (v PayloadValidator) ValidateModal(m discord.ModalCreate) error {
	if strings.TrimSpace(m.Title) == "" || utf8.RuneCountInString(m.Title) > MaxModalTitleRunes {
		return fmt.Errorf("modal title must be 1-%d chars", MaxModalTitleRunes)
	}
	if m.CustomID == "" || utf8.RuneCountInString(m.CustomID) > MaxCustomIDRunes {
		return fmt.Errorf("modal custom_id must be 1-%d chars", MaxCustomIDRunes)
	}
	if len(m.Components) == 0 || len(m.Components) > MaxModalLabels {
		return fmt.Errorf("modal must have 1-%d components, got %d", MaxModalLabels, len(m.Components))
	}

	totalTextRunes := 0
	seenCustomIDs := make(map[string]struct{})
	for _, comp := range m.Components {
		switch c := comp.(type) {
		case discord.LabelComponent:
			if strings.TrimSpace(c.Label) == "" || utf8.RuneCountInString(c.Label) > MaxLabelTextRunes {
				return fmt.Errorf("modal label text must be 1-%d chars", MaxLabelTextRunes)
			}
			if utf8.RuneCountInString(c.Description) > MaxLabelDescriptionRunes {
				return fmt.Errorf("modal label description exceeds %d chars", MaxLabelDescriptionRunes)
			}
			switch sub := c.Component.(type) {
			case discord.TextInputComponent:
				if sub.CustomID == "" || utf8.RuneCountInString(sub.CustomID) > MaxCustomIDRunes {
					return fmt.Errorf("text input custom_id must be 1-%d chars", MaxCustomIDRunes)
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
				if utf8.RuneCountInString(sub.Placeholder) > MaxTextInputPlaceholder {
					return fmt.Errorf("text input placeholder exceeds %d chars", MaxTextInputPlaceholder)
				}
				if sub.MaxLength < 0 || sub.MaxLength > MaxTextInputLength {
					return fmt.Errorf("text input max_length %d out of range", sub.MaxLength)
				}
				effectiveMax := sub.MaxLength
				if effectiveMax == 0 {
					effectiveMax = MaxTextInputLength
				}
				if sub.MinLength != nil {
					if *sub.MinLength < 0 || *sub.MinLength > effectiveMax {
						return fmt.Errorf("text input min_length %d invalid for max_length %d", *sub.MinLength, effectiveMax)
					}
				}
				if sub.Value != "" {
					valRunes := utf8.RuneCountInString(sub.Value)
					if valRunes > effectiveMax {
						return fmt.Errorf("text input pre-filled value %d exceeds max_length %d", valRunes, effectiveMax)
					}
					if sub.MinLength != nil && valRunes < *sub.MinLength {
						return errors.New("text input pre-filled value is shorter than min_length")
					}
				}
			case discord.StringSelectMenuComponent:
				if err := validateStringSelect(sub, true); err != nil {
					return err
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.UserSelectMenuComponent:
				if sub.Disabled {
					return errors.New("disabled components are not allowed in modals")
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.RoleSelectMenuComponent:
				if sub.Disabled {
					return errors.New("disabled components are not allowed in modals")
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.MentionableSelectMenuComponent:
				if sub.Disabled {
					return errors.New("disabled components are not allowed in modals")
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.ChannelSelectMenuComponent:
				if sub.Disabled {
					return errors.New("disabled components are not allowed in modals")
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.CheckboxGroupComponent:
				if err := validateCheckboxGroup(sub); err != nil {
					return err
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			case discord.CheckboxComponent:
				if sub.CustomID == "" || utf8.RuneCountInString(sub.CustomID) > MaxCustomIDRunes {
					return fmt.Errorf("checkbox custom_id must be 1-%d chars", MaxCustomIDRunes)
				}
				if err := registerCustomID(seenCustomIDs, sub.CustomID); err != nil {
					return err
				}
			}
		case discord.TextDisplayComponent:
			if strings.TrimSpace(c.Content) == "" {
				return errors.New("modal text display cannot be empty")
			}
			totalTextRunes += utf8.RuneCountInString(c.Content)
			if totalTextRunes > MaxTextDisplayRunesTotal {
				return fmt.Errorf("modal text display total runes %d exceeds %d", totalTextRunes, MaxTextDisplayRunesTotal)
			}
		}
	}
	return nil
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

// ValidateAutocompleteChoices rejects choices exceeding Discord's 25-item or 100-char bounds without clamping.
func ValidateAutocompleteChoices(choices []discord.AutocompleteChoice) error {
	if len(choices) > MaxAutocompleteChoices {
		return fmt.Errorf("autocomplete choices count %d exceeds %d", len(choices), MaxAutocompleteChoices)
	}
	for _, c := range choices {
		if ch, ok := c.(discord.AutocompleteChoiceString); ok {
			if strings.TrimSpace(ch.Name) == "" || utf8.RuneCountInString(ch.Name) > MaxSelectTextRunes {
				return fmt.Errorf("autocomplete choice name must be 1-%d chars: %q", MaxSelectTextRunes, ch.Name)
			}
			if ch.Value == "" || utf8.RuneCountInString(ch.Value) > MaxSelectTextRunes {
				return fmt.Errorf("autocomplete choice value must be 1-%d chars: %q", MaxSelectTextRunes, ch.Value)
			}
		}
	}
	return nil
}

// ValidateAutocompleteChoices strictly validates choices against Discord's bounds without clamping.
func (v PayloadValidator) ValidateAutocompleteChoices(choices []discord.AutocompleteChoice) error {
	return ValidateAutocompleteChoices(choices)
}

// GuildStateReader abstracts cached Discord guild/channel lookups for preflight checks and unit testing.
type GuildStateReader interface {
	GuildFeatures(guildID snowflake.ID) ([]discord.GuildFeature, bool)
	GuildChannelCount(guildID snowflake.ID) int
	CategoryChildCount(guildID, categoryID snowflake.ID) int
	BotGuildPermissions(guildID snowflake.ID) (discord.Permissions, bool)
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

// ChannelCreatePlan is the validated preflight result for provisioning a board channel.
type ChannelCreatePlan struct {
	UseForum       bool
	BotPermissions discord.Permissions
	Name           string
	Topic          string
}

// PreflightChecker verifies guild limits, permissions, and role hierarchies before REST calls.
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
	{discord.PermissionCreatePublicThreads, "CreatePublicThreads"},
	{discord.PermissionCreatePrivateThreads, "CreatePrivateThreads"},
	{discord.PermissionManageThreads, "ManageThreads"},
	{discord.PermissionManageMessages, "ManageMessages"},
	{discord.PermissionReadMessageHistory, "ReadMessageHistory"},
}

type PreflightErrorKind string

const (
	PreflightErrEmptyChannelName     PreflightErrorKind = "empty_channel_name"
	PreflightErrChannelNameTooLong   PreflightErrorKind = "channel_name_too_long"
	PreflightErrChannelTopicTooLong  PreflightErrorKind = "channel_topic_too_long"
	PreflightErrGuildChannelLimit    PreflightErrorKind = "guild_channel_limit"
	PreflightErrCategoryChannelLimit PreflightErrorKind = "category_channel_limit"
	PreflightErrMissingBotPerms      PreflightErrorKind = "missing_bot_perms"
	PreflightErrForumRequiresTag     PreflightErrorKind = "forum_requires_tag"
)

// PreflightError represents a localized preflight check failure before calling Discord REST APIs.
type PreflightError struct {
	Kind         PreflightErrorKind
	Current      int
	Max          int
	MissingPerms []string
}

func (e *PreflightError) Error() string {
	switch e.Kind {
	case PreflightErrEmptyChannelName:
		return "channel name cannot be empty"
	case PreflightErrChannelNameTooLong:
		return fmt.Sprintf("channel name exceeds max length (%d/%d)", e.Current, e.Max)
	case PreflightErrChannelTopicTooLong:
		return fmt.Sprintf("channel topic exceeds max length (%d/%d)", e.Current, e.Max)
	case PreflightErrGuildChannelLimit:
		return fmt.Sprintf("guild channel limit reached (%d/%d)", e.Current, e.Max)
	case PreflightErrCategoryChannelLimit:
		return fmt.Sprintf("category channel limit reached (%d/%d)", e.Current, e.Max)
	case PreflightErrMissingBotPerms:
		return fmt.Sprintf("missing required bot permissions: %s", strings.Join(e.MissingPerms, ", "))
	case PreflightErrForumRequiresTag:
		return "forum channel requires a tag but has no available tags"
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
	case PreflightErrChannelNameTooLong:
		return i18n.TranslateText(locale, "components.bbs.preflight.channel_name_too_long", map[string]any{
			"current": pe.Current,
			"max":     pe.Max,
		}), true
	case PreflightErrChannelTopicTooLong:
		return i18n.TranslateText(locale, "components.bbs.preflight.channel_topic_too_long", map[string]any{
			"current": pe.Current,
			"max":     pe.Max,
		}), true
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
	default:
		return pe.Error(), true
	}
}

// CheckChannelCreate strictly verifies channel bounds, guild/category caps, and required bot permissions.
func (p *PreflightChecker) CheckChannelCreate(
	guildID snowflake.ID,
	name string,
	topic string,
	wantForum bool,
	parentCategoryID *snowflake.ID,
) (ChannelCreatePlan, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ChannelCreatePlan{UseForum: wantForum}, &PreflightError{Kind: PreflightErrEmptyChannelName}
	}
	if nameRunes := utf8.RuneCountInString(trimmedName); nameRunes > MaxChannelNameRunes {
		return ChannelCreatePlan{UseForum: wantForum}, &PreflightError{
			Kind:    PreflightErrChannelNameTooLong,
			Current: nameRunes,
			Max:     MaxChannelNameRunes,
		}
	}

	trimmedTopic := strings.TrimSpace(topic)
	topicLimit := MaxTextTopicRunes
	if wantForum {
		topicLimit = MaxForumTopicRunes
	}
	if topicRunes := utf8.RuneCountInString(trimmedTopic); topicRunes > topicLimit {
		return ChannelCreatePlan{UseForum: wantForum}, &PreflightError{
			Kind:    PreflightErrChannelTopicTooLong,
			Current: topicRunes,
			Max:     topicLimit,
		}
	}

	plan := ChannelCreatePlan{
		UseForum: wantForum,
		Name:     trimmedName,
		Topic:    trimmedTopic,
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

		if perms, ok := p.state.BotGuildPermissions(guildID); ok {
			plan.BotPermissions = perms
			if !perms.Has(discord.PermissionAdministrator) {
				var missing []string
				for _, req := range requiredBoardPermissions {
					if !perms.Has(req.perm) {
						missing = append(missing, req.name)
					}
				}
				if len(missing) > 0 {
					return plan, &PreflightError{Kind: PreflightErrMissingBotPerms, MissingPerms: missing}
				}
			}
		}
	}

	return plan, nil
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
	if err := r.validator.ValidateModal(modal); err != nil {
		return err
	}
	r.mu.Lock()
	if r.inflight || r.ackUncertain || !r.canInitialAck() {
		r.mu.Unlock()
		return errors.New("interaction already acknowledged, uncertain, or past 3s initial window")
	}
	r.inflight = true
	r.mu.Unlock()
	err := event.Modal(modal)
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
	if err := r.validator.ValidateMessageCreate(mc); err != nil {
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
		isV2 := mc.Flags.Has(discord.MessageFlagIsComponentsV2)
		mu := discord.MessageUpdate{
			Components:      &mc.Components,
			AllowedMentions: mc.AllowedMentions,
		}
		if !isV2 {
			c := mc.Content
			mu.Content = &c
			mu.Embeds = &mc.Embeds
		} else {
			flags := mc.Flags
			mu.Flags = &flags
		}
		if err := r.validator.ValidateMessageUpdate(mu, isV2); err != nil {
			r.mu.Lock()
			r.inflight = false
			r.mu.Unlock()
			return err
		}
		_, err := event.UpdateInteractionResponse(mu)
		r.mu.Lock()
		r.inflight = false
		if err == nil {
			r.deferred = false
			r.acked = true
		}
		r.mu.Unlock()
		return err

	case modeFollowup:
		_, err := event.CreateFollowupMessage(mc)
		r.mu.Lock()
		r.inflight = false
		r.mu.Unlock()
		return err

	default:
		err := event.CreateMessage(mc)
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
	if err := r.validator.ValidateMessageUpdate(mu, true); err != nil {
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
		_, err := event.UpdateInteractionResponse(mu)
		r.mu.Lock()
		r.inflight = false
		if err == nil {
			r.deferred = false
			r.acked = true
		}
		r.mu.Unlock()
		return err
	}
	err := event.UpdateMessage(mu)
	r.finishInitialAttempt(err, false, false)
	return err
}

// SafeRespondComponentModal safely sends a modal from a ComponentInteractionCreate event.
func SafeRespondComponentModal(e *events.ComponentInteractionCreate, modal discord.ModalCreate) error {
	return NewSafeResponder(e.ID().Time()).RespondModal(e, modal)
}

// DebounceTimer abstracts *time.Timer for deterministic debouncer unit tests.
type DebounceTimer interface {
	Stop() bool
}

// AfterFunc schedules f after d and returns a DebounceTimer.
type AfterFunc func(d time.Duration, f func()) DebounceTimer

type debounceEntry struct {
	seq   uint64
	timer DebounceTimer
}

// ControlPanelDebouncer coalesces rapid floating control-panel moves per thread mirror.
type ControlPanelDebouncer struct {
	delay     time.Duration
	afterFunc AfterFunc
	mu        sync.Mutex
	nextSeq   uint64
	timers    map[uuid.UUID]debounceEntry
	closed    bool
}

// NewControlPanelDebouncer constructs a debouncer with delay (defaults to 2.5s when <= 0).
func NewControlPanelDebouncer(delay time.Duration) *ControlPanelDebouncer {
	return NewControlPanelDebouncerWithTimer(delay, nil)
}

// NewControlPanelDebouncerWithTimer constructs a debouncer using an injectable timer factory.
func NewControlPanelDebouncerWithTimer(delay time.Duration, afterFn AfterFunc) *ControlPanelDebouncer {
	if delay <= 0 {
		delay = DefaultDebouncerDelay
	}
	if afterFn == nil {
		afterFn = func(d time.Duration, f func()) DebounceTimer {
			return time.AfterFunc(d, f)
		}
	}
	return &ControlPanelDebouncer{
		delay:     delay,
		afterFunc: afterFn,
		timers:    make(map[uuid.UUID]debounceEntry),
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
	t := d.afterFunc(d.delay, func() {
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
	halfOpenProbe   map[uuid.UUID]uint64
	nextProbeSeq    uint64
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
		halfOpenProbe:   make(map[uuid.UUID]uint64),
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

// AcquireMount checks global throttling and mount circuit breaker state atomically at now.
// When allowed, callers MUST defer the returned release func so any half-open probe lock is cleaned up.
func (g *InvalidRequestGuard) AcquireMount(mountID uuid.UUID, now time.Time) (allowed bool, release func()) {
	allowed, _, release = g.AcquireMountWithProbe(mountID, now)
	return allowed, release
}

// AcquireMountWithProbe checks global throttling and mount circuit breaker state atomically at now,
// returning a unique non-zero probeSeq token when a half-open probe slot was acquired.
func (g *InvalidRequestGuard) AcquireMountWithProbe(mountID uuid.UUID, now time.Time) (allowed bool, probeSeq uint64, release func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pruneLocked(now)
	if len(g.events) >= g.threshold {
		return false, 0, func() {}
	}
	if mountID == uuid.Nil {
		return true, 0, func() {}
	}
	openedAt, open := g.openMounts[mountID]
	if !open {
		return true, 0, func() {}
	}
	if g.circuitCooldown > 0 && now.Sub(openedAt) >= g.circuitCooldown && g.halfOpenProbe[mountID] == 0 {
		g.nextProbeSeq++
		seq := g.nextProbeSeq
		g.halfOpenProbe[mountID] = seq
		return true, seq, func() { g.ReleaseMountProbeSeq(mountID, seq) }
	}
	return false, 0, func() {}
}

// ReleaseMountProbeSeq clears the half-open probe marker for mountID only if its token matches seq.
func (g *InvalidRequestGuard) ReleaseMountProbeSeq(mountID uuid.UUID, seq uint64) {
	if mountID == uuid.Nil || seq == 0 {
		return
	}
	g.mu.Lock()
	if g.halfOpenProbe[mountID] == seq {
		delete(g.halfOpenProbe, mountID)
	}
	g.mu.Unlock()
}

// ReleaseMountProbe clears an in-flight half-open probe marker for mountID unconditionally.
func (g *InvalidRequestGuard) ReleaseMountProbe(mountID uuid.UUID) {
	if mountID == uuid.Nil {
		return
	}
	g.mu.Lock()
	delete(g.halfOpenProbe, mountID)
	g.mu.Unlock()
}

// RecordError inspects err and updates global invalid-request tracking and mount circuit breakers.
// Returns true if the mount should be permanently deactivated (e.g. 404 Unknown Channel).
func (g *InvalidRequestGuard) RecordError(mountID uuid.UUID, err error, now time.Time) (deactivateMount bool) {
	return g.RecordMountError(mountID, 0, err, now)
}

// RecordMountError updates global invalid-request tracking and mount circuit breakers for a REST outcome.
// Only a matching probeSeq releases an active half-open probe marker on non-403/404 errors.
func (g *InvalidRequestGuard) RecordMountError(mountID uuid.UUID, probeSeq uint64, err error, now time.Time) (deactivateMount bool) {
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
			g.mu.Lock()
			delete(g.halfOpenProbe, mountID)
			delete(g.openMounts, mountID)
			delete(g.mountFailures, mountID)
			g.mu.Unlock()
			return true
		}
		if status == 403 || IsJSONError(err, JSONErrMissingAccess) || IsJSONError(err, JSONErrMissingPermissions) {
			g.mu.Lock()
			if probeSeq != 0 && g.halfOpenProbe[mountID] == probeSeq {
				delete(g.halfOpenProbe, mountID)
			}
			if _, alreadyOpen := g.openMounts[mountID]; alreadyOpen {
				if g.halfOpenProbe[mountID] == 0 {
					g.openMounts[mountID] = now
				}
			} else {
				g.mountFailures[mountID]++
				if g.mountFailures[mountID] >= g.mountFailLimit {
					g.openMounts[mountID] = now
				}
			}
			g.mu.Unlock()
			return false
		}
		g.ReleaseMountProbeSeq(mountID, probeSeq)
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

// IsMountOpenAt reports whether mountID is currently blocked by an open circuit breaker without mutating probe state.
func (g *InvalidRequestGuard) IsMountOpenAt(mountID uuid.UUID, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	openedAt, open := g.openMounts[mountID]
	if !open {
		return false
	}
	if g.circuitCooldown > 0 && now.Sub(openedAt) >= g.circuitCooldown && g.halfOpenProbe[mountID] == 0 {
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
