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
