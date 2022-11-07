package translate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	defaultLang = language.English
)

var translations *i18n.Bundle

func init() {
	loadTranslations()
}

func loadTranslations() error {
	dir := filepath.Join("lang")
	bundle := i18n.NewBundle(defaultLang)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".toml" {
			return nil
		}
		_, err = bundle.LoadMessageFile(path)
		return err
	})
	if err != nil {
		return err
	}

	translations = bundle
	return nil
}

func Message(locale discordgo.Locale, messageId string) (res string) {
	res = Translate(locale, messageId, map[string]interface{}{})
	return
}

func Translate(locale discordgo.Locale, messageId string, templateData interface{}) (res string) {
	res = Translates(locale, messageId, templateData, 2)
	return
}

func Translates(locale discordgo.Locale, messageId string, templateData interface{}, pluralCount int) string {
	defaultLocalizer := i18n.NewLocalizer(translations, string(locale))
	res, err := defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
		PluralCount:  pluralCount,
	})
	if err != nil {
		defaultLocalizer = i18n.NewLocalizer(translations, "en")
		res, err = defaultLocalizer.Localize(&i18n.LocalizeConfig{
			MessageID:    messageId,
			TemplateData: templateData,
			PluralCount:  pluralCount,
		})
		if err != nil {
			res = fmt.Sprintf("Translate error: %v", err)
		}
	}
	return strings.ToLower(res)
}

func MessageMap(key string) *map[discordgo.Locale]string {
	res := &map[discordgo.Locale]string{
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
	return res
}
