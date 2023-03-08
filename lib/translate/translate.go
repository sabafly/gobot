/*
	Copyright (C) 2022-2023  ikafly144

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
package translate

import (
	"embed"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

//go:embed lang/*
//go:embed ja.yaml
var f embed.FS

var (
	defaultLang  = language.Japanese
	translations = i18n.Bundle{}
)

func init() {
	translations, _ = loadTranslations()
}

func loadTranslations() (i18n.Bundle, error) {
	bundle := i18n.NewBundle(defaultLang)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	buf, err := f.ReadFile("ja.yaml")
	if err != nil {
		panic(err)
	}
	ln, err := i18n.ParseMessageFileBytes(buf, "ja.yaml", map[string]i18n.UnmarshalFunc{
		"yaml": yaml.Unmarshal,
	})
	if err != nil {
		panic(err)
	}
	err = bundle.AddMessages(ln.Tag, ln.Messages...)
	if err != nil {
		panic(err)
	}
	fd, err := f.ReadDir("lang")
	if err != nil {
		panic(err)
	}
	for _, de := range fd {
		_, err := bundle.LoadMessageFileFS(f, "lang/"+de.Name())
		if err != nil {
			panic(err)
		}
	}
	return *bundle, nil
}

func Message(locale discord.Locale, messageId string) (res string) {
	res = Translate(locale, messageId, map[string]any{})
	return
}

func Translate(locale discord.Locale, messageId string, templateData any) (res string) {
	res = Translates(locale, messageId, templateData, 2)
	return
}

func Translates(locale discord.Locale, messageId string, templateData any, pluralCount int) string {
	messageId = strings.ReplaceAll(messageId, ".", "_")
	Localizer := i18n.NewLocalizer(&translations, string(locale))
	res, err := Localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
		PluralCount:  pluralCount,
	})
	if err != nil {
		Localizer = i18n.NewLocalizer(&translations, "ja")
		res, err = Localizer.Localize(&i18n.LocalizeConfig{
			MessageID:    messageId,
			TemplateData: templateData,
			PluralCount:  pluralCount,
		})
		if err != nil {
			res = fmt.Sprintf("translate error: %v", err)
		}
	}
	return res
}

func MessageMap(key string, replace bool) *map[discord.Locale]string {
	res := map[discord.Locale]string{
		discord.LocaleEnglishUS:    Message(discord.LocaleEnglishUS, ""),
		discord.LocaleEnglishGB:    Message(discord.LocaleEnglishGB, ""),
		discord.LocaleBulgarian:    Message(discord.LocaleBulgarian, ""),
		discord.LocaleChineseCN:    Message(discord.LocaleChineseCN, ""),
		discord.LocaleChineseTW:    Message(discord.LocaleChineseTW, ""),
		discord.LocaleCroatian:     Message(discord.LocaleCroatian, ""),
		discord.LocaleCzech:        Message(discord.LocaleCzech, ""),
		discord.LocaleDanish:       Message(discord.LocaleDanish, ""),
		discord.LocaleDutch:        Message(discord.LocaleDutch, ""),
		discord.LocaleFinnish:      Message(discord.LocaleFinnish, ""),
		discord.LocaleFrench:       Message(discord.LocaleFrench, ""),
		discord.LocaleGerman:       Message(discord.LocaleGerman, ""),
		discord.LocaleGreek:        Message(discord.LocaleGreek, ""),
		discord.LocaleHindi:        Message(discord.LocaleHindi, ""),
		discord.LocaleHungarian:    Message(discord.LocaleHungarian, ""),
		discord.LocaleIndonesian:   Message(discord.LocaleIndonesian, ""),
		discord.LocaleItalian:      Message(discord.LocaleItalian, ""),
		discord.LocaleJapanese:     Message(discord.LocaleJapanese, ""),
		discord.LocaleKorean:       Message(discord.LocaleKorean, ""),
		discord.LocaleLithuanian:   Message(discord.LocaleLithuanian, ""),
		discord.LocaleNorwegian:    Message(discord.LocaleNorwegian, ""),
		discord.LocalePolish:       Message(discord.LocalePolish, ""),
		discord.LocalePortugueseBR: Message(discord.LocalePortugueseBR, ""),
		discord.LocaleRomanian:     Message(discord.LocaleRomanian, ""),
		discord.LocaleRussian:      Message(discord.LocaleRussian, ""),
		discord.LocaleSpanishES:    Message(discord.LocaleSpanishES, ""),
		discord.LocaleSwedish:      Message(discord.LocaleSwedish, ""),
		discord.LocaleThai:         Message(discord.LocaleThai, ""),
		discord.LocaleTurkish:      Message(discord.LocaleTurkish, ""),
		discord.LocaleUkrainian:    Message(discord.LocaleUkrainian, ""),
		discord.LocaleVietnamese:   Message(discord.LocaleVietnamese, ""),
		discord.LocaleUnknown:      Message(discord.LocaleUnknown, ""),
	}
	if replace {
		for l, v := range res {
			res[l] = strings.ReplaceAll(v, " ", "-")
		}
	}
	return &res
}

func ErrorEmbed(locale discord.Locale, key string, any ...any) (embed []*discord.Embed) {
	var trs string
	if len(any) != 0 {
		trs = Translate(locale, key, any[0])
	} else if key != "" {
		trs = Message(locale, key)
	}
	embed = append(embed, &discord.Embed{
		Title:       Message(locale, "error_message"),
		Description: trs,
		Color:       0xff0000,
	})
	return
}
