package i18n

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/disgoorg/disgo/discord"
	"gopkg.in/yaml.v3"
)

var (
	defaultLocale discord.Locale
	globalLocales = make(map[discord.Locale]*LocalizedValues, 0)
)

// DefaultLocale returns the configured default locale, falling back to Japanese.
func DefaultLocale() discord.Locale {
	if defaultLocale != "" && defaultLocale != discord.LocaleUnknown {
		return defaultLocale
	}
	return discord.LocaleJapanese
}

// AvailableLocales returns all loaded locales with DefaultLocale first followed by sorted locale codes.
func AvailableLocales() []discord.Locale {
	if len(globalLocales) == 0 {
		return nil
	}
	def := DefaultLocale()
	var out []discord.Locale
	if _, ok := globalLocales[def]; ok {
		out = append(out, def)
	}
	var rest []discord.Locale
	for loc := range globalLocales {
		if loc == def {
			continue
		}
		rest = append(rest, loc)
	}
	slices.SortFunc(rest, func(a, b discord.Locale) int {
		return strings.Compare(string(a), string(b))
	})
	return append(out, rest...)
}

// IsSupportedLocale reports whether locale can be selected as a gobot override locale.
func IsSupportedLocale(locale discord.Locale) bool {
	if locale == "" || locale == discord.LocaleUnknown {
		return false
	}
	if len(globalLocales) > 0 {
		_, ok := globalLocales[locale]
		return ok
	}
	return locale == discord.LocaleJapanese || locale == discord.LocaleEnglishUS
}

// LocaleDisplayName returns a human-readable display string for locale.
func LocaleDisplayName(locale discord.Locale) string {
	switch locale {
	case discord.LocaleUnknown:
		return LocaleDisplayName(DefaultLocale())
	case discord.LocaleJapanese:
		return "日本語 (ja)"
	case discord.LocaleEnglishUS:
		return "English (US) (en-US)"
	case discord.LocaleEnglishGB:
		return "English (UK) (en-GB)"
	default:
		if locale.String() != discord.LocaleUnknown.String() {
			return fmt.Sprintf("%s (%s)", locale.String(), locale.Code())
		}
		return locale.Code()
	}
}

func newTemplate(name string) *template.Template {
	return template.New(name).
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"div": func(n float64, d float64) float64 {
				return n / d
			},
			"mul": func(n float64, d float64) float64 {
				return n * d
			},
			"add": func(n float64, d float64) float64 {
				return n + d
			},
			"sub": func(n float64, d float64) float64 {
				return n - d
			},
		})
}

func TranslateText(locale discord.Locale, key string, args ...any) string {
	if locale == discord.LocaleUnknown {
		locale = defaultLocale
	}

	var parseString string
	found := false
	if localeValues, exists := globalLocales[locale]; exists {
		if value, exists := localeValues.Strings[key]; exists {
			parseString = value
			found = true
		}
	}
	if !found && locale == discord.LocaleEnglishGB {
		if localeValues, exists := globalLocales[discord.LocaleEnglishUS]; exists {
			if value, exists := localeValues.Strings[key]; exists {
				parseString = value
				found = true
			}
		}
	}
	if !found {
		if defaultValues, exists := globalLocales[defaultLocale]; exists {
			if value, exists := defaultValues.Strings[key]; exists {
				parseString = value
				found = true
			}
		}
	}
	if !found {
		parseString = key // Fallback to the key itself if no translation is found
		slog.Warn("TranslateText: no translation found", "key", key, "locale", locale)
	}

	tmpl := newTemplate("translate")
	if _, err := tmpl.Parse(parseString); err != nil {
		slog.Error("TranslateText: failed to parse template", "key", key, "locale", locale, "error", err)
		return key // Fallback to the key itself if template parsing fails
	}

	var sb strings.Builder
	var data any
	if len(args) == 1 {
		data = args[0]
	} else if len(args) > 1 {
		data = args
	}
	if err := tmpl.Execute(&sb, data); err != nil {
		slog.Error("TranslateText: failed to execute template", "key", key, "locale", locale, "error", err)
		return key // Fallback to the key itself if template execution fails
	}

	return sb.String()
}

func TranslateTextMap(key string) map[discord.Locale]string {
	translations := make(map[discord.Locale]string, len(globalLocales))
	for locale, values := range globalLocales {
		if value, exists := values.Strings[key]; exists {
			translations[locale] = value
		} else if defaultValues, exists := globalLocales[defaultLocale]; exists {
			if value, exists := defaultValues.Strings[key]; exists {
				translations[locale] = value
			} else {
				slog.Warn("TranslateTextMap: no translation found", "key", key, "locale", locale)
				translations[locale] = key
			}
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

	if locale == discord.LocaleEnglishGB {
		if localeValues, exists := globalLocales[discord.LocaleEnglishUS]; exists {
			if values, exists := localeValues.Components[key]; exists {
				return values
			}
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

		ext := filepath.Ext(file.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		f, err := os.Open(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}

		locale, err := loadLocale(f)
		_ = f.Close()
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
