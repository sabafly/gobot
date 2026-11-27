/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package components

import (
	"os"
	"regexp"

	"github.com/disgoorg/snowflake/v2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LocaleDir         string        `yaml:"locale_dir"`
	EmojiRegistryFile string        `yaml:"emoji_registry_file"`
	Debug             ConfigDebug   `yaml:"debug"`
	Message           ConfigMessage `yaml:"message"`

	MySQL   string `yaml:"mysql"`
	GormDSN string `yaml:"gorm_dsn"`

	BumpUserID snowflake.ID   `yaml:"bump_user"`
	BumpImage  *regexp.Regexp `yaml:"bump_image_regex"`
	UpUserID   snowflake.ID   `yaml:"up_user"`
	UpColor    int            `yaml:"up_color"`

	DisableForeignKeys bool `yaml:"foreign_keys"`

	BBS   ConfigBBS   `yaml:"bbs"`
	Admin ConfigAdmin `yaml:"admin"`
}

type ConfigAdmin struct {
	Enabled         bool                       `yaml:"enabled"`
	ListenAddr      string                     `yaml:"listen_addr"`
	PublicURL       string                     `yaml:"public_url"`
	BasePath        string                     `yaml:"base_path"`
	SessionTTLHours int                        `yaml:"session_ttl_hours"`
	LogBufferSize   int                        `yaml:"log_buffer_size"`
	LogLevel        string                     `yaml:"log_level"`
	TrustedProxies  []string                   `yaml:"trusted_proxies"`
	OAuth           ConfigAdminOAuth           `yaml:"oauth"`
	Roles           map[string]ConfigAdminRole `yaml:"roles"`
	Users           []ConfigAdminUser          `yaml:"users"`
}

type ConfigAdminOAuth struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

type ConfigAdminRole struct {
	Permissions []string `yaml:"permissions"`
}

type ConfigAdminUser struct {
	UserID snowflake.ID `yaml:"user_id"`
	Role   string       `yaml:"role"`
}

type ConfigBBS struct {
	SecretKey          string               `yaml:"secret_key"`
	Operators          []snowflake.ID       `yaml:"operators"`
	OperatorGuilds     []snowflake.ID       `yaml:"operator_guilds"`
	OperatorLogChannel *snowflake.ID        `yaml:"operator_log_channel"`
	PolicyVersion      string               `yaml:"policy_version"`
	TermsURL           string               `yaml:"terms_url"`
	GuidelinesURL      string               `yaml:"guidelines_url"`
	PrivacyPolicyURL   string               `yaml:"privacy_policy_url"`
	Caps               []ConfigBBSCap       `yaml:"caps"`
	SlipEvents         []ConfigBBSSlipEvent `yaml:"slip_events"`
}

type ConfigBBSSlipEvent struct {
	ID        string `yaml:"id"`
	Label     string `yaml:"label"`
	StartDate string `yaml:"start_date"`
	EndDate   string `yaml:"end_date"`
	Mode      string `yaml:"mode"`
}

type ConfigBBSCap struct {
	ID             string         `yaml:"id"`
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description"`
	Users          []snowflake.ID `yaml:"users"`
	GrantOperators bool           `yaml:"grant_operators"`
}

type ConfigDebug struct {
	Trace       bool           `yaml:"trace"`
	DebugUsers  []snowflake.ID `yaml:"users"`
	DebugGuilds []snowflake.ID `yaml:"guilds"`
}

type ConfigMessage struct {
	PinIconImage string `yaml:"pin_icon_image"`
}

func (c *Components) Config() Config { return c.config }

func LoadConfig(path string) (config *Config, err error) {
	config = &Config{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		e := f.Close()
		if e != nil {
			err = e
		}
	}(f)

	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
