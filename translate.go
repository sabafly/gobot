package main

import (
	"fmt"
	"log"
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
var defaultLocalizer *i18n.Localizer

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
	defaultLocalizer = i18n.NewLocalizer(bundle, defaultLang.String())
	log.Printf("%v", defaultLang.String())
	return nil
}

func translate(locale discordgo.Locale, messageId string, templateData interface{}) (res string) {
	defaultLocalizer = i18n.NewLocalizer(translations, locale.String())
	res, err := defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
	})
	if err != nil {
		res = fmt.Sprintf("Translate error: %v", err)
	}
	return
}

func translates(locale discordgo.Locale, messageId string, templateData interface{}, pluralCount int) (res string) {
	defaultLocalizer = i18n.NewLocalizer(translations, locale.String())
	res, err := defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
		PluralCount:  pluralCount,
	})
	if err != nil {
		res = fmt.Sprintf("Translate error: %v", err)
	}
	return
}
