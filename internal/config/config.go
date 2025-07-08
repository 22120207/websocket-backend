package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
}

var (
	config     *Config
	configOnce sync.Once
)

func LoadConfig() (*Config, error) {
	var err error

	configOnce.Do(func() {
		if envErr := godotenv.Load(); envErr != nil {
			err = fmt.Errorf("error loading .env file: %w", envErr)
			return
		}

		config = &Config{
			Port: os.Getenv("PORT"),
		}

		if config.Port == "" {
			config.Port = "8080"
		}
	})

	return config, err
}
