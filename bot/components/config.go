package components

import "github.com/disgoorg/snowflake/v2"

type Config struct {
	PrivateGuilds []snowflake.ID `yaml:"private_guilds"`
	TranslateDir  string         `yaml:"translate_dir"`
	Message       ConfigMessage  `yaml:"message"`
}

type ConfigMessage struct {
	PinIconImage string `yaml:"pin_icon_image"`
}

func (c *Components) Config() Config { return c.config }
