package components

import (
	"os"

	"github.com/disgoorg/snowflake/v2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	PrivateGuilds []snowflake.ID `yaml:"private_guilds"`
	TranslateDir  string         `yaml:"translate_dir"`
	Message       ConfigMessage  `yaml:"message"`

	MySQL string   `yaml:"mysql"`
	Redis []string `yaml:"redis"`
}

type ConfigMessage struct {
	PinIconImage string `yaml:"pin_icon_image"`
}

func (c *Components) Config() Config { return c.config }

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
