package i18n

import (
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"gopkg.in/yaml.v3"
)

var (
	defaultLocale discord.Locale
	globalLocales = make(map[discord.Locale]*LocalizedValues, 0)
)

func TranslateText(locale discord.Locale, key string) string {
	if locale == discord.LocaleUnknown {
		locale = defaultLocale
	}

	if localeValues, exists := globalLocales[locale]; exists {
		if value, exists := localeValues.Strings[key]; exists {
			return value
		}
	}

	if defaultValues, exists := globalLocales[defaultLocale]; exists {
		if value, exists := defaultValues.Strings[key]; exists {
			return value
		}
	}

	slog.Warn("TranslateText: no translation found", "key", key, "locale", locale)
	return key // Fallback to the key itself if no translation is found
}

func TranslateTextMap(key string) map[discord.Locale]string {
	translations := make(map[discord.Locale]string, len(globalLocales))
	for locale, values := range globalLocales {
		if value, exists := values.Strings[key]; exists {
			translations[locale] = value
		} else {
			slog.Warn("TranslateTextMap: no translation found", "key", key, "locale", locale)
			translations[locale] = key // Fallback to the key itself if no translation is found
		}
	}
	return translations
}

func TranslateCommandOptionMap(key string) map[discord.Locale]string {
	translations := TranslateTextMap(key)
	for locale, value := range translations {
		translations[locale] = strings.ReplaceAll(value, " ", "-")
	}
	return translations
}

func TranslateComponent(locale discord.Locale, key string) []Component {
	if locale == discord.LocaleUnknown {
		locale = defaultLocale
	}

	if localeValues, exists := globalLocales[locale]; exists {
		if values, exists := localeValues.Components[key]; exists {
			return values
		}
	}

	if defaultValues, exists := globalLocales[defaultLocale]; exists {
		if values, exists := defaultValues.Components[key]; exists {
			return values
		}
	}

	slog.Warn("TranslateComponent: no translation found", "key", key, "locale", locale)
	return []Component{
		TextDisplay("No translation found for key: " + key),
	}
}

func TranslateLayout(locale discord.Locale, key string) []LayoutComponent {
	components := TranslateComponent(locale, key)
	if components == nil {
		return nil // Return nil if no components are found
	}
	layoutComponents := make([]LayoutComponent, len(components))
	for i, component := range components {
		l, ok := component.(LayoutComponent)
		if !ok {
			slog.Warn("TranslateLayout: component is not a LayoutComponent", "component", component, "key", key, "locale", locale)
			continue
		}
		layoutComponents[i] = l
	}

	return layoutComponents
}

// LoadLocales loads all locales from the specified directory.
func LoadLocales(dir string) error {
	globalLocales = make(map[discord.Locale]*LocalizedValues, 0)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		f, err := os.Open(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}
		//nolint:errcheck
		defer f.Close()

		ext := filepath.Ext(file.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		locale, err := loadLocale(f)
		if err != nil {
			return err
		}

		if _, exists := globalLocales[locale.Locale]; exists {
			return ErrDuplicateLocale.Format(locale.Locale)
		}

		slog.Info("Loaded locale", "locale", locale.Locale, "file", file.Name())

		globalLocales[locale.Locale] = locale

		if strings.TrimSuffix(file.Name(), ext) == "default" {
			defaultLocale = locale.Locale
		}
	}

	return nil
}

func loadLocale(file *os.File) (*LocalizedValues, error) {
	yamlDecoder := yaml.NewDecoder(file)
	var locale struct {
		Locale     discord.Locale                  `yaml:"locale"`
		Components map[string][]UnmarshalComponent `yaml:"components"`
		Strings    map[string]string               `yaml:"strings"`
	}

	if err := yamlDecoder.Decode(&locale); err != nil {
		return nil, err
	}

	if locale.Locale == "" || locale.Locale.String() == discord.LocaleUnknown.String() {
		return nil, ErrUnknownLocale.Format(locale.Locale)
	}

	result := &LocalizedValues{
		Locale:     locale.Locale,
		Components: make(map[string][]Component, len(locale.Components)),
		Strings:    make(map[string]string, len(locale.Strings)),
	}

	for key, value := range locale.Components {
		values := make([]Component, len(value))
		for i := range value {
			values[i] = value[i].Component
		}
		result.Components[key] = values
	}

	maps.Copy(result.Strings, locale.Strings)

	return result, nil
}

type LocalizedValues struct {
	Locale     discord.Locale
	Components map[string][]Component
	Strings    map[string]string
}
