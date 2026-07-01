package i18n

import (
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"
)

func BuildContext() *MapContext {
	return &MapContext{
		texts:          make(map[string]string),
		defaultValues:  make(map[string][]discord.SelectMenuDefaultValue),
		defaultOptions: make(map[string][]string),
		maxValues:      make(map[string]int),
		minValues:      make(map[string]int),
		disabled:       make(map[string]bool),
		urls:           make(map[string]string),
	}
}

type MapContext struct {
	texts          map[string]string
	defaultValues  map[string][]discord.SelectMenuDefaultValue
	defaultOptions map[string][]string
	stringOptions  map[string][]discord.StringSelectMenuOption
	maxValues      map[string]int
	minValues      map[string]int
	disabled       map[string]bool
	urls           map[string]string
	customId       map[string]string
}

func (m *MapContext) WithText(key, value string) *MapContext {
	if m.texts == nil {
		m.texts = make(map[string]string)
	}
	m.texts[key] = value
	return m
}

func (m *MapContext) WithDefaultValues(customID string, values []discord.SelectMenuDefaultValue) *MapContext {
	if m.defaultValues == nil {
		m.defaultValues = make(map[string][]discord.SelectMenuDefaultValue)
	}
	if len(values) == 0 {
		delete(m.defaultValues, customID)
		return m
	}
	m.defaultValues[customID] = values
	return m
}

func (m *MapContext) WithDefaultOptions(customID string, options []string) *MapContext {
	if m.defaultOptions == nil {
		m.defaultOptions = make(map[string][]string)
	}
	if len(options) == 0 {
		delete(m.defaultOptions, customID)
		return m
	}
	m.defaultOptions[customID] = options
	return m
}

func (m *MapContext) WithStringOptions(customID string, options []discord.StringSelectMenuOption) *MapContext {
	if m.stringOptions == nil {
		m.stringOptions = make(map[string][]discord.StringSelectMenuOption)
	}
	if len(options) == 0 {
		delete(m.stringOptions, customID)
		return m
	}
	m.stringOptions[customID] = options
	return m
}

func (m *MapContext) WithMaxValues(customID string, value int) *MapContext {
	if m.maxValues == nil {
		m.maxValues = make(map[string]int)
	}
	if value <= 0 {
		delete(m.maxValues, customID)
		return m
	}
	m.maxValues[customID] = value
	return m
}

func (m *MapContext) WithMinValues(customID string, value int) *MapContext {
	if m.minValues == nil {
		m.minValues = make(map[string]int)
	}
	if value < 0 {
		delete(m.minValues, customID)
		return m
	}
	m.minValues[customID] = value
	return m
}

func (m *MapContext) WithDisabled(customID string, disabled bool) *MapContext {
	if m.disabled == nil {
		m.disabled = make(map[string]bool)
	}
	if !disabled {
		delete(m.disabled, customID)
		return m
	}
	m.disabled[customID] = disabled
	return m
}

func (m *MapContext) WithURL(key, url string) *MapContext {
	if m.urls == nil {
		m.urls = make(map[string]string)
	}
	if url == "" {
		delete(m.urls, key)
		return m
	}
	m.urls[key] = url
	return m
}

func (m *MapContext) WithCustomID(key, customID string) *MapContext {
	if m.customId == nil {
		m.customId = make(map[string]string)
	}
	if customID == "" {
		delete(m.customId, key)
		return m
	}
	m.customId[key] = customID
	return m
}

func (m MapContext) Translate(layouts []LayoutComponent) []discord.LayoutComponent {
	if len(layouts) == 0 {
		return nil
	}
	deserialized, err := deserializeLayoutComponent(m, layouts)
	if err != nil {
		return nil
	}
	return deserialized
}

func (m MapContext) GetText(key string) (string, bool) {
	value, ok := m.texts[key]
	return value, ok
}

func (m MapContext) ReplaceText(text string) string {
	if len(m.texts) == 0 {
		return text
	}
	for key, value := range m.texts {
		if value == "" {
			continue
		}
		text = strings.ReplaceAll(text, "{"+key+"}", value)
	}
	return text
}

func (m MapContext) GetDefaultValues(customID string, t discord.SelectMenuDefaultValueType) []discord.SelectMenuDefaultValue {
	if len(m.defaultValues) == 0 {
		return nil
	}
	values, ok := m.defaultValues[customID]
	if !ok {
		return nil
	}
	for i, v := range slices.All(values) {
		if v.Type == "" {
			v.Type = t
		}
		if v.Type == "" {
			continue
		}
		values[i] = v
	}
	return values
}

func (m MapContext) GetDefaultOptions(customID string, options []discord.StringSelectMenuOption) []discord.StringSelectMenuOption {
	if len(m.defaultOptions) == 0 {
		return options
	}
	defaults, ok := m.defaultOptions[customID]
	if !ok {
		return options
	}
	for i, option := range slices.All(options) {
		if slices.Contains(defaults, option.Value) {
			option.Default = true
		} else {
			option.Default = false
		}
		options[i] = option
	}
	return options
}

func (m MapContext) GetStringOptions(customID string) []discord.StringSelectMenuOption {
	if len(m.stringOptions) == 0 {
		return nil
	}
	options, ok := m.stringOptions[customID]
	if !ok {
		return nil
	}
	return options
}

func (m MapContext) GetMaxValues(customID string) int {
	if len(m.maxValues) == 0 {
		return 1
	}
	value, ok := m.maxValues[customID]
	if !ok {
		return 1
	}
	return value
}

func (m MapContext) GetMinValues(customID string) *int {
	if len(m.minValues) == 0 {
		return nil
	}
	value, ok := m.minValues[customID]
	if !ok {
		return nil
	}
	return &value
}

func (m MapContext) IsDisabled(customID string) bool {
	if len(m.disabled) == 0 {
		return false
	}
	disabled, ok := m.disabled[customID]
	if !ok {
		return false
	}
	return disabled
}

func (m MapContext) GetURL(key string) string {
	if len(m.urls) == 0 {
		return ""
	}
	url := m.urls[key]
	return url
}

func (m MapContext) ReplaceCustomID(id string) string {
	if len(m.customId) == 0 {
		return id
	}
	for key, value := range m.customId {
		if value == "" {
			continue
		}
		id = strings.ReplaceAll(id, "{"+key+"}", value)
	}
	return id
}
