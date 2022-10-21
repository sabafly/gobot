package translate

import (
	"fmt"
	"os"
	"path/filepath"

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

func Translates(locale discordgo.Locale, messageId string, templateData interface{}, pluralCount int) (res string) {
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
	return
}
