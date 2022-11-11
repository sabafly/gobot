package translate

import (
	"fmt"
	"log"
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

func init() {
	loadTranslations()
}

func loadTranslations() (i18n.Bundle, error) {
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
		return i18n.Bundle{}, err
	}

	return *bundle, nil
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
	translations, err := loadTranslations()
	if err != nil {
		panic(err)
	}
	messageId = strings.ReplaceAll(messageId, ".", "_")
	defaultLocalizer := i18n.NewLocalizer(&translations, string(locale))
	res, err := defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageId,
		TemplateData: templateData,
		PluralCount:  pluralCount,
	})
	if err != nil {
		defaultLocalizer = i18n.NewLocalizer(&translations, "en")
		res, err = defaultLocalizer.Localize(&i18n.LocalizeConfig{
			MessageID:    messageId,
			TemplateData: templateData,
			PluralCount:  pluralCount,
		})
		if err != nil {
			log.Print(err)
			res = fmt.Sprintf("Translate error: %v", err)
		}
	}
	return res
}

func MessageMap(key string, replace bool) *map[discordgo.Locale]string {
	var res *map[discordgo.Locale]string
	if replace {
		res = &map[discordgo.Locale]string{
			discordgo.Bulgarian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Bulgarian, key)), " ", ""),
			discordgo.ChineseCN:    strings.ReplaceAll(strings.ToLower(Message(discordgo.ChineseCN, key)), " ", ""),
			discordgo.ChineseTW:    strings.ReplaceAll(strings.ToLower(Message(discordgo.ChineseTW, key)), " ", ""),
			discordgo.Croatian:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Croatian, key)), " ", ""),
			discordgo.Czech:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Czech, key)), " ", ""),
			discordgo.Danish:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Danish, key)), " ", ""),
			discordgo.Dutch:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Dutch, key)), " ", ""),
			discordgo.EnglishGB:    strings.ReplaceAll(strings.ToLower(Message(discordgo.EnglishGB, key)), " ", ""),
			discordgo.EnglishUS:    strings.ReplaceAll(strings.ToLower(Message(discordgo.EnglishUS, key)), " ", ""),
			discordgo.Finnish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Finnish, key)), " ", ""),
			discordgo.French:       strings.ReplaceAll(strings.ToLower(Message(discordgo.French, key)), " ", ""),
			discordgo.German:       strings.ReplaceAll(strings.ToLower(Message(discordgo.German, key)), " ", ""),
			discordgo.Greek:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Greek, key)), " ", ""),
			discordgo.Hindi:        strings.ReplaceAll(strings.ToLower(Message(discordgo.Hindi, key)), " ", ""),
			discordgo.Hungarian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Hungarian, key)), " ", ""),
			discordgo.Italian:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Italian, key)), " ", ""),
			discordgo.Japanese:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Japanese, key)), " ", ""),
			discordgo.Korean:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Korean, key)), " ", ""),
			discordgo.Lithuanian:   strings.ReplaceAll(strings.ToLower(Message(discordgo.Lithuanian, key)), " ", ""),
			discordgo.Norwegian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Norwegian, key)), " ", ""),
			discordgo.Polish:       strings.ReplaceAll(strings.ToLower(Message(discordgo.Polish, key)), " ", ""),
			discordgo.PortugueseBR: strings.ReplaceAll(strings.ToLower(Message(discordgo.PortugueseBR, key)), " ", ""),
			discordgo.Romanian:     strings.ReplaceAll(strings.ToLower(Message(discordgo.Romanian, key)), " ", ""),
			discordgo.Russian:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Russian, key)), " ", ""),
			discordgo.SpanishES:    strings.ReplaceAll(strings.ToLower(Message(discordgo.SpanishES, key)), " ", ""),
			discordgo.Swedish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Swedish, key)), " ", ""),
			discordgo.Thai:         strings.ReplaceAll(strings.ToLower(Message(discordgo.Thai, key)), " ", ""),
			discordgo.Turkish:      strings.ReplaceAll(strings.ToLower(Message(discordgo.Turkish, key)), " ", ""),
			discordgo.Ukrainian:    strings.ReplaceAll(strings.ToLower(Message(discordgo.Ukrainian, key)), " ", ""),
			discordgo.Vietnamese:   strings.ReplaceAll(strings.ToLower(Message(discordgo.Vietnamese, key)), " ", ""),
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

func ErrorEmbed(locale discordgo.Locale, key string, any ...interface{}) (embed []*discordgo.MessageEmbed) {
	var trs string
	if any[0] != nil {
		trs = Translate(locale, key, any[0])
	} else if key != "" {
		trs = Message(locale, key)
	}
	embed = append(embed, &discordgo.MessageEmbed{
		Title:       Message(locale, "error_message"),
		Description: trs,
	})
	return
}
