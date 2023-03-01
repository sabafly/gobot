package botlib

import (
	"errors"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/lib/logging"
)

var (
	ErrNoFeatureData = errors.New("no feature data")
)

type FeatureManager struct {
	sync.Mutex
	featureMap                 map[FeatureType][]*Feature
	ApplicationCommandSettings FeatureApplicationCommandSettings
}

type FeatureApplicationCommandSettings struct {
	Name                    string
	Description             string
	NameLocalization        *map[discordgo.Locale]string
	DescriptionLocalization *map[discordgo.Locale]string
	Permission              int64
	DMPermission            bool
	Type                    discordgo.ApplicationCommandType
}

func NewFeatureManager() *FeatureManager {
	return &FeatureManager{
		featureMap: map[FeatureType][]*Feature{},
	}
}

func (bm *BotManager) FeatureCreate(f *Feature) (err error) {
	bm.features.Lock()
	defer bm.features.Unlock()
	bm.features.featureMap[f.Type] = append(bm.features.featureMap[f.Type], f)
	return nil
}

type FeatureIDType string

const (
	FeatureChannelID FeatureIDType = "CHANNEL_ID"
	FeatureUserID    FeatureIDType = "USER_ID"
	FeatureGuildID   FeatureIDType = "GUILD_ID"
	FeatureRoleID    FeatureIDType = "ROLE_ID"
)

type Feature struct {
	Name         string
	ID           string
	IDType       FeatureIDType
	ChannelTypes []discordgo.ChannelType
	Type         FeatureType
	Handler      any
}

type FeatureType string

const (
	FeatureMessageCreate FeatureType = "MESSAGE_CREATE"
	FeatureTypingStart               = "TYPING_START"
	FeatureCustom                    = "CUSTOM"
	FeatureUnknown       FeatureType = ""
)

func (f FeatureType) String() string {
	if s, ok := Features[f]; ok {
		return s
	}
	return FeatureUnknown.String()
}

var Features = map[FeatureType]string{
	FeatureMessageCreate: "Message Create",
	FeatureTypingStart:   "Typing Start",
	FeatureCustom:        "Custom",
	FeatureUnknown:       "Unknown",
}

type FeatureData interface {
	Write(string)
	Delete(string)
	IsEnabled(string) bool
}

func (bm *BotManager) FeatureApplicationCommandSettingsSet(settings FeatureApplicationCommandSettings) {
	bm.features.ApplicationCommandSettings = settings
}

func (bm *BotManager) FeatureHandler() func(*discordgo.Session, any) {
	return func(s *discordgo.Session, a any) {
		switch v := a.(type) {
		case *discordgo.MessageCreate:
			for _, f := range bm.features.featureMap[FeatureMessageCreate] {
				fn, ok := f.Handler.(func(*discordgo.Session, *discordgo.MessageCreate))
				if !ok {
					continue
				}
				var equal bool
				switch f.IDType {
				case FeatureChannelID:
					enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.ChannelID)
					if err != nil {
						equal = false
					} else {
						equal = enabled
					}
				case FeatureGuildID:
					enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.GuildID)
					if err != nil {
						equal = false
					} else {
						equal = enabled
					}
				case FeatureUserID:
					if v.Member != nil {
						enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.Member.User.ID)
						if err != nil {
							equal = false
						} else {
							equal = enabled
						}
					}
				case FeatureRoleID:
					m, err := s.GuildMember(v.GuildID, v.GuildID)
					if err != nil {
						continue
					}
					for _, r := range m.Roles {
						enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, r)
						if err != nil {
							equal = false
						} else {
							equal = enabled
						}
					}
				}
				if !equal {
					continue
				}
				fn(s, v)
			}
		case *discordgo.TypingStart:
			for _, f := range bm.features.featureMap[FeatureTypingStart] {
				fn, ok := f.Handler.(func(*discordgo.Session, *discordgo.TypingStart))
				if !ok {
					continue
				}
				var equal bool
				switch f.IDType {
				case FeatureChannelID:
					enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.ChannelID)
					if err != nil {
						equal = false
					} else {
						equal = enabled
					}
				case FeatureGuildID:
					enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.GuildID)
					if err != nil {
						equal = false
					} else {
						equal = enabled
					}
				case FeatureUserID:
					enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, v.UserID)
					if err != nil {
						equal = false
					} else {
						equal = enabled
					}
				case FeatureRoleID:
					m, err := s.GuildMember(v.GuildID, v.GuildID)
					if err != nil {
						continue
					}
					for _, r := range m.Roles {
						enabled, err := bm.FeatureEnabled(v.GuildID, f.ID, r)
						if err != nil {
							equal = false
						} else {
							equal = enabled
							break
						}
					}
				}
				if !equal {
					continue
				}
				fn(s, v)
			}
		}
	}
}

func (bm *BotManager) FeaturesApplicationCommand() *discordgo.ApplicationCommand {
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, v := range bm.features.featureMap {
		for _, f := range v {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  f.Name,
				Value: string(f.Type) + ":" + f.ID,
			})
		}
	}
	options := []*discordgo.ApplicationCommandOption{
		{
			Name:        "enable",
			Description: "enable feature",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "feature",
					Description: "the feature to enable",
					Type:        discordgo.ApplicationCommandOptionString,
					Choices:     choices,
				},
			},
		},
		{
			Name:        "disable",
			Description: "disable feature",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "feature",
					Description: "the feature to enable",
					Type:        discordgo.ApplicationCommandOptionString,
					Choices:     choices,
				},
			},
		},
	}
	return &discordgo.ApplicationCommand{
		Name:                     bm.features.ApplicationCommandSettings.Name,
		Description:              bm.features.ApplicationCommandSettings.Description,
		NameLocalizations:        bm.features.ApplicationCommandSettings.NameLocalization,
		DescriptionLocalizations: bm.features.ApplicationCommandSettings.DescriptionLocalization,
		DefaultMemberPermissions: &bm.features.ApplicationCommandSettings.Permission,
		DMPermission:             &bm.features.ApplicationCommandSettings.DMPermission,
		Type:                     bm.features.ApplicationCommandSettings.Type,
		Options:                  options,
	}
}

func (bm *BotManager) FeatureApplicationCommandHandler() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		acd := i.ApplicationCommandData()
		if acd.Name != bm.features.ApplicationCommandSettings.Name {
			return
		}
		logging.Debug("called")
		option := acd.Options[0]
		switch option.Name {
		case "enable":
			logging.Debug("enable called")
			option = option.Options[0]
			var featureID string
			var featureTypeStr string
			var featureType FeatureType
			switch option.Name {
			case "feature":
				featureID = option.StringValue()
				ids := strings.Split(featureID, ":")
				if len(ids) != 0 {
					featureTypeStr = ids[0]
					featureID = ids[len(ids)-1]
				}
				featureType = FeatureType(featureTypeStr)
			default:
				logging.Warning("よくわかんねぇ引数来てっぞ")
			}
			if featureID == "" {
				// IDが指定されてないエラー
				embeds := ErrorMessageEmbed(i, "error_invalid_command_argument")
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: embeds,
					},
				})
				if err != nil {
					logging.Error("インタラクションに失敗 %s", err)
				}
				return
			}
			for _, f := range bm.features.featureMap[featureType] {
				if f.ID != featureID {
					continue
				}
				var values []string
				i, values = RequestFeatureIDRespond(s, i, f)
				for _, v := range values {
					err := bm.FeatureEnable(i.GuildID, f.ID, v)
					if err != nil {
						logging.Error("有効化に失敗 %s", err)
						embeds := ErrorMessageEmbed(i, "error_create_failed")
						err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Embeds: embeds,
							},
						})
						if err != nil {
							logging.Error("インタラクションに失敗 %s", err)
						}
						return
					}
				}
				// TODO: カスタマイズ可能なメッセージ
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "OK",
					},
				})
				if err != nil {
					logging.Error("インタラクションレスポンスに失敗 %s", err)
				}
			}
		case "disable":
			logging.Debug("disable called")
			option = option.Options[0]
			var featureID string
			var featureTypeStr string
			var featureType FeatureType
			switch option.Name {
			case "feature":
				featureID = option.StringValue()
				ids := strings.Split(featureID, ":")
				if len(ids) != 0 {
					featureTypeStr = ids[0]
					featureID = ids[len(ids)-1]
				}
				featureType = FeatureType(featureTypeStr)
			default:
				logging.Warning("よくわかんねぇ引数来てっぞ")
			}
			if featureID == "" {
				// IDが指定されてないエラー
				embeds := ErrorMessageEmbed(i, "error_invalid_command_argument")
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: embeds,
					},
				})
				if err != nil {
					logging.Error("インタラクションに失敗 %s", err)
				}
				return
			}
			for _, f := range bm.features.featureMap[featureType] {
				if f.ID != featureID {
					continue
				}
				var values []string
				i, values = RequestFeatureIDRespond(s, i, f)
				for _, v := range values {
					err := bm.FeatureDisable(i.GuildID, f.ID, v)
					if err != nil {
						logging.Error("無効化に失敗 %s", err)
						embeds := ErrorMessageEmbed(i, "error_already_deleted")
						err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Embeds: embeds,
							},
						})
						if err != nil {
							logging.Error("インタラクションに失敗 %s", err)
						}
						return
					}
				}
				// TODO: カスタマイズ可能なメッセージ
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "OK",
					},
				})
				if err != nil {
					logging.Error("インタラクションレスポンスに失敗 %s", err)
				}
			}
		}
	}
}

// TODO: 再利用可能に
func RequestFeatureIDRespond(s *discordgo.Session, i *discordgo.InteractionCreate, f *Feature) (ic *discordgo.InteractionCreate, fID []string) {
	var menuType discordgo.SelectMenuType
	channelTypes := []discordgo.ChannelType{}
	switch f.IDType {
	case FeatureChannelID:
		menuType = discordgo.ChannelSelectMenu
		channelTypes = f.ChannelTypes
	case FeatureRoleID:
		menuType = discordgo.RoleSelectMenu
	case FeatureUserID:
		menuType = discordgo.UserSelectMenu
	case FeatureGuildID:
		return i, []string{i.GuildID}
	case FeatureCustom:
		// TODO: 実装する
	}
	// UUID: かぶらないよね
	sessionID := uuid.NewString()
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:     menuType,
					ChannelTypes: channelTypes,
					CustomID:     sessionID,
				},
			},
		},
	}
	logging.Debug("response send")
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
	if err != nil {
		logging.Error("レスポンスに失敗 %s", err)
		return
	}
	var i1 *discordgo.InteractionCreate
	// TODO: タイムアウトを追加
	var c chan struct{} = make(chan struct{})
	var handler func(*discordgo.Session, *discordgo.InteractionCreate)
	handler = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logging.Debug("called")
		if i.Type != discordgo.InteractionMessageComponent {
			logging.Debug("not message component %d", i.Type)
			s.AddHandlerOnce(handler)
			return
		}
		if i.MessageComponentData().CustomID != sessionID {
			logging.Debug("not same custom id")
			s.AddHandlerOnce(handler)
			return
		}
		i1 = i
		logging.Debug("close!")
		close(c)
	}
	s.AddHandlerOnce(handler)
	<-c
	return i1, i1.MessageComponentData().Values
}
