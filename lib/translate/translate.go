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

	"github.com/bwmarrin/discordgo"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sabafly/gobot/lib/logging"
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

func Translates(locale discordgo.Locale, messageId string, templateData any, pluralCount int) string {
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
			logging.Error("翻訳に失敗しました %s", err)
			res = fmt.Sprintf("translate error: %v", err)
		}
	}
	return res
}

func MessageMap(key string, replace bool) *map[discordgo.Locale]string {
	res := map[discordgo.Locale]string{
		discordgo.Bulgarian:    Message(discordgo.Bulgarian, key),
		discordgo.ChineseCN:    Message(discordgo.ChineseCN, key),
		discordgo.ChineseTW:    Message(discordgo.ChineseTW, key),
		discordgo.Croatian:     Message(discordgo.Croatian, key),
		discordgo.Czech:        Message(discordgo.Czech, key),
		discordgo.Danish:       Message(discordgo.Danish, key),
		discordgo.Dutch:        Message(discordgo.Dutch, key),
		discordgo.EnglishGB:    Message(discordgo.EnglishGB, key),
		discordgo.EnglishUS:    Message(discordgo.EnglishUS, key),
		discordgo.Finnish:      Message(discordgo.Finnish, key),
		discordgo.French:       Message(discordgo.French, key),
		discordgo.German:       Message(discordgo.German, key),
		discordgo.Greek:        Message(discordgo.Greek, key),
		discordgo.Hindi:        Message(discordgo.Hindi, key),
		discordgo.Hungarian:    Message(discordgo.Hungarian, key),
		discordgo.Italian:      Message(discordgo.Italian, key),
		discordgo.Japanese:     Message(discordgo.Japanese, key),
		discordgo.Korean:       Message(discordgo.Korean, key),
		discordgo.Lithuanian:   Message(discordgo.Lithuanian, key),
		discordgo.Norwegian:    Message(discordgo.Norwegian, key),
		discordgo.Polish:       Message(discordgo.Polish, key),
		discordgo.PortugueseBR: Message(discordgo.PortugueseBR, key),
		discordgo.Romanian:     Message(discordgo.Romanian, key),
		discordgo.Russian:      Message(discordgo.Russian, key),
		discordgo.SpanishES:    Message(discordgo.SpanishES, key),
		discordgo.Swedish:      Message(discordgo.Swedish, key),
		discordgo.Thai:         Message(discordgo.Thai, key),
		discordgo.Turkish:      Message(discordgo.Turkish, key),
		discordgo.Ukrainian:    Message(discordgo.Ukrainian, key),
		discordgo.Vietnamese:   Message(discordgo.Vietnamese, key),
	}
	if replace {
		for l, v := range res {
			res[l] = strings.ReplaceAll(v, " ", "-")
		}
	}
	return &res
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
