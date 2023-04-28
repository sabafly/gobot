package botlib

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/sabafly/gobot/lib/db"
	"gopkg.in/yaml.v2"
)

func LoadConfig(config_filepath string) (*Config, error) {
	file, err := os.Open(config_filepath)
	if os.IsNotExist(err) {
		if file, err = os.Create(config_filepath); err != nil {
			return nil, err
		}
		var data []byte
		switch filepath.Ext(config_filepath) {
		case ".json":
			data, err = json.MarshalIndent(Config{}, "", "\t")
		case ".yml", ".yaml":
			data, err = yaml.Marshal(Config{})
		case ".toml":
			data, err = toml.Marshal(Config{})
		default:
			panic("unknown config file type " + filepath.Ext(config_filepath))
		}
		if err != nil {
			return nil, err
		}
		if _, err = file.Write(data); err != nil {
			return nil, err
		}
		return nil, errors.New("config file not found, created new one")
	} else if err != nil {
		return nil, err
	}

	var cfg Config
	switch filepath.Ext(config_filepath) {
	case ".json":
		err = json.NewDecoder(file).Decode(&cfg)
	case ".yml", ".yaml":
		err = yaml.NewDecoder(file).Decode(&cfg)
	case ".toml":
		err = toml.NewDecoder(file).Decode(&cfg)
	default:
		panic("unknown config file type" + filepath.Ext(config_filepath))
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	DevMode            bool           `json:"dev_mode"`
	DevOnly            bool           `json:"dev_only"`
	DevGuildIDs        []snowflake.ID `json:"dev_guild_id"`
	DevUserIDs         []snowflake.ID `json:"dev_user_id"`
	LogLevel           string         `json:"log_level"`
	Token              string         `json:"token"`
	DMPermission       bool           `json:"dm_permission"`
	ShouldSyncCommands bool           `json:"sync_commands"`
	DBConfig           db.DBConfig    `json:"db_config"`
	Dislog             DislogConfig   `json:"dislog"`
}

type DislogConfig struct {
	WebhookChannel snowflake.ID `json:"webhook_channel"`
	WebhookID      snowflake.ID `json:"webhook_id"`
	WebhookToken   string       `json:"webhook_token"`
}
