package config

import "github.com/pelletier/go-toml"

var (
	Data *toml.Tree
)

func Load() {
	configFile := "config.toml"
	Data, _ = toml.LoadFile(configFile)
}
