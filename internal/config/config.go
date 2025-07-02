package config

import (
	"fmt"
	"os"
	"sync"
	"websocket-backend/models"
	"websocket-backend/pkg/utils"

	"github.com/joho/godotenv"
)

var (
	config     *models.Config
	configOnce sync.Once
)

func LoadConfig() (*models.Config, error) {
	var err error

	configOnce.Do(func() {
		if envErr := godotenv.Load(); envErr != nil {
			err = fmt.Errorf("error loading .env file: %w", envErr)
			return
		}

		config = &models.Config{
			Port: os.Getenv("PORT"),
		}

		if config.Port == "" {
			config.Port = "8080"
		}

		inventoryPath := os.Getenv("INVENTORY_TARGET")
		if inventoryPath == "" {
			err = fmt.Errorf("INVENTORY_TARGET not set in environment")
			return
		}

		hosts, parseErr := utils.ParseInventoryFile(inventoryPath)
		if parseErr != nil {
			err = parseErr
			return
		}

		config.Target = hosts
	})

	return config, err
}
