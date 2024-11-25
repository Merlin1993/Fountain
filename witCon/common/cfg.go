package common

import (
	"github.com/naoina/toml"
	"os"
	"witCon/log"
)

func GetConfig(p string) *Config {
	_, err := os.Stat(p)
	config := new(Config)
	if err == nil {
		file, err := os.Open(p)
		if err != nil {
			log.Error("Failed to read config file: %v", "err", err)
		}
		defer file.Close()
		if err := toml.NewDecoder(file).Decode(&config); err != nil {
			log.Error("invalid config file: %v", "err", err)
		}
		return config
	} else {
		log.Error("nil config file: %v", "err", err)
	}
	return nil
}
