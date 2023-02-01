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
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sabafly/gobot/pkg/lib/logging"
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
	logging.Info("翻訳ファイルを読み込みます")
	translations, _ = loadTranslations()
}

func loadTranslations() (i18n.Bundle, error) {
	bundle := i18n.NewBundle(defaultLang)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	logging.Info("ja.yaml を読み込み中...")
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
	logging.Info("完了")
	fd, err := f.ReadDir("lang")
	if err != nil {
		panic(err)
	}
	for _, de := range fd {
		logging.Info("%v を読み込み中...", de.Name())
		_, err := bundle.LoadMessageFileFS(f, "lang/"+de.Name())
		if err != nil {
			logging.Error("%v の読み込みに失敗 %s", de.Name(), err)
		}
		logging.Info("完了")
	}
	logging.Info("翻訳ファイルの読み込み完了")
	return *bundle, nil
}

func Message(locale discordgo.Locale, messageId string) (res string) {
	res = Translate(locale, messageId, map[string]any{})
	return
}

func Translate(locale discordgo.Locale, messageId string, templateData any) (res string) {
	res = Translates(locale, messageId, templateData, 2)
	return
}

var reg = regexp.MustCompile("[,\\.;:\\]\\[@\\\\\\^\\/\\-!\"#\\$%&'\\(\\)=~\\|<>\\?_\\+\\*\\}\\{`]")

func Translates(locale discordgo.Locale, messageId string, templateData any, pluralCount int) string {
	messageId = strings.ReplaceAll(messageId, ".", "_")
	defaultLocalizer := i18n.NewLocalizer(&translations, string(locale))
	res, err := defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
		PluralCount:  pluralCount,
	})
	if err != nil {
		defaultLocalizer = i18n.NewLocalizer(&translations, "ja")
		res, err = defaultLocalizer.Localize(&i18n.LocalizeConfig{
			MessageID:    messageId,
			TemplateData: templateData,
			PluralCount:  pluralCount,
		})
		if err != nil {
			log.Print(err)
			res = fmt.Sprintf("translate error: %v", err)
		}
	}
	match := reg.FindAllString(res, -1)
	for _, v := range match {
		res = strings.Replace(res, v, "", 1)
	}
	return res
}

func MessageMap(key string, replace bool) *map[discordgo.Locale]string {
	var res *map[discordgo.Locale]string
	if replace {
		res = &map[discordgo.Locale]string{
			discordgo.Bulgarian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Bulgarian, key)), " ", "_"),
			discordgo.ChineseCN:    strings.ReplaceAll(strings.ToLower(Message(discordgo.ChineseCN, key)), " ", "_"),
			discordgo.ChineseTW:    strings.ReplaceAll(strings.ToLower(Message(discordgo.ChineseTW, key)), " ", "_"),
			discordgo.Croatian:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Croatian, key)), " ", "_"),
			discordgo.Czech:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Czech, key)), " ", "_"),
			discordgo.Danish:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Danish, key)), " ", "_"),
			discordgo.Dutch:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Dutch, key)), " ", "_"),
			discordgo.EnglishGB:    strings.ReplaceAll(strings.ToLower(Message(discordgo.EnglishGB, key)), " ", "_"),
			discordgo.EnglishUS:    strings.ReplaceAll(strings.ToLower(Message(discordgo.EnglishUS, key)), " ", "_"),
			discordgo.Finnish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Finnish, key)), " ", "_"),
			discordgo.French:       strings.ReplaceAll(strings.ToLower(Message(discordgo.French, key)), " ", "_"),
			discordgo.German:       strings.ReplaceAll(strings.ToLower(Message(discordgo.German, key)), " ", "_"),
			discordgo.Greek:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Greek, key)), " ", "_"),
			discordgo.Hindi:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Hindi, key)), " ", "_"),
			discordgo.Hungarian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Hungarian, key)), " ", "_"),
			discordgo.Italian:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Italian, key)), " ", "_"),
			discordgo.Japanese:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Japanese, key)), " ", "_"),
			discordgo.Korean:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Korean, key)), " ", "_"),
			discordgo.Lithuanian:   strings.ReplaceAll(strings.ToLower(Message(discordgo.Lithuanian, key)), " ", "_"),
			discordgo.Norwegian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Norwegian, key)), " ", "_"),
			discordgo.Polish:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Polish, key)), " ", "_"),
			discordgo.PortugueseBR: strings.ReplaceAll(strings.ToLower(Message(discordgo.PortugueseBR, key)), " ", "_"),
			discordgo.Romanian:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Romanian, key)), " ", "_"),
			discordgo.Russian:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Russian, key)), " ", "_"),
			discordgo.SpanishES:    strings.ReplaceAll(strings.ToLower(Message(discordgo.SpanishES, key)), " ", "_"),
			discordgo.Swedish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Swedish, key)), " ", "_"),
			discordgo.Thai:         strings.ReplaceAll(strings.ToLower(Message(discordgo.Thai, key)), " ", "_"),
			discordgo.Turkish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Turkish, key)), " ", "_"),
			discordgo.Ukrainian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Ukrainian, key)), " ", "_"),
			discordgo.Vietnamese:   strings.ReplaceAll(strings.ToLower(Message(discordgo.Vietnamese, key)), " ", "_"),
		}
	} else {
		res = &map[discordgo.Locale]string{
			discordgo.Bulgarian:    strings.ToLower(Message(discordgo.Bulgarian, key)),
			discordgo.ChineseCN:    strings.ToLower(Message(discordgo.ChineseCN, key)),
			discordgo.ChineseTW:    strings.ToLower(Message(discordgo.ChineseTW, key)),
			discordgo.Croatian:     strings.ToLower(Message(discordgo.Croatian, key)),
			discordgo.Czech:        strings.ToLower(Message(discordgo.Czech, key)),
			discordgo.Danish:       strings.ToLower(Message(discordgo.Danish, key)),
			discordgo.Dutch:        strings.ToLower(Message(discordgo.Dutch, key)),
			discordgo.EnglishGB:    strings.ToLower(Message(discordgo.EnglishGB, key)),
			discordgo.EnglishUS:    strings.ToLower(Message(discordgo.EnglishUS, key)),
			discordgo.Finnish:      strings.ToLower(Message(discordgo.Finnish, key)),
			discordgo.French:       strings.ToLower(Message(discordgo.French, key)),
			discordgo.German:       strings.ToLower(Message(discordgo.German, key)),
			discordgo.Greek:        strings.ToLower(Message(discordgo.Greek, key)),
			discordgo.Hindi:        strings.ToLower(Message(discordgo.Hindi, key)),
			discordgo.Hungarian:    strings.ToLower(Message(discordgo.Hungarian, key)),
			discordgo.Italian:      strings.ToLower(Message(discordgo.Italian, key)),
			discordgo.Japanese:     strings.ToLower(Message(discordgo.Japanese, key)),
			discordgo.Korean:       strings.ToLower(Message(discordgo.Korean, key)),
			discordgo.Lithuanian:   strings.ToLower(Message(discordgo.Lithuanian, key)),
			discordgo.Norwegian:    strings.ToLower(Message(discordgo.Norwegian, key)),
			discordgo.Polish:       strings.ToLower(Message(discordgo.Polish, key)),
			discordgo.PortugueseBR: strings.ToLower(Message(discordgo.PortugueseBR, key)),
			discordgo.Romanian:     strings.ToLower(Message(discordgo.Romanian, key)),
			discordgo.Russian:      strings.ToLower(Message(discordgo.Russian, key)),
			discordgo.SpanishES:    strings.ToLower(Message(discordgo.SpanishES, key)),
			discordgo.Swedish:      strings.ToLower(Message(discordgo.Swedish, key)),
			discordgo.Thai:         strings.ToLower(Message(discordgo.Thai, key)),
			discordgo.Turkish:      strings.ToLower(Message(discordgo.Turkish, key)),
			discordgo.Ukrainian:    strings.ToLower(Message(discordgo.Ukrainian, key)),
			discordgo.Vietnamese:   strings.ToLower(Message(discordgo.Vietnamese, key)),
		}
	}
	return res
}

func ErrorEmbed(locale discordgo.Locale, key string, any ...any) (embed []*discordgo.MessageEmbed) {
	var trs string
	if len(any) != 0 {
		trs = Translate(locale, key, any[0])
	} else if key != "" {
		trs = Message(locale, key)
	}
	embed = append(embed, &discordgo.MessageEmbed{
		Title:       Message(locale, "error_message"),
		Description: trs,
		Color:       0xff0000,
	})
	return
}
