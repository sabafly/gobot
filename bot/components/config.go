package components

import "github.com/disgoorg/snowflake/v2"

type Config struct {
	PrivateGuilds []snowflake.ID `yaml:"private_guilds"`
	TranslateDir  string         `yaml:"translate_dir"`
}

func (c *Components) Config() Config { return c.config }
