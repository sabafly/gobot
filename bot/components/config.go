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

	"github.com/disgoorg/snowflake/v2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	TranslateDir string        `yaml:"translate_dir"`
	Debug        ConfigDebug   `yaml:"debug"`
	Message      ConfigMessage `yaml:"message"`

	MySQL string   `yaml:"mysql"`
	Redis []string `yaml:"redis"`

	BumpUserID snowflake.ID `yaml:"bump_user"`
	BumpImage  string       `yaml:"bump_image"`
	UpUserID   snowflake.ID `yaml:"up_user"`
	UpColor    int          `yaml:"up_color"`
}

type ConfigDebug struct {
	DebugUsers  []snowflake.ID `yaml:"users"`
	DebugGuilds []snowflake.ID `yaml:"guilds"`
}

type ConfigMessage struct {
	PinIconImage string `yaml:"pin_icon_image"`
}

func (c *Components) Config() Config { return c.config }

func Load(path string) (config *Config, err error) {
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
