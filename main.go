package main

import (
	"log"
	"websocket-backend/internal/config"
	"websocket-backend/pkg/utils"
	"websocket-backend/routes"
)

func main() {
	// Init log
	utils.SetupLogger()

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("error in load file dotenv config")
	}

	r := routes.SetupRouter()
	err = r.Run(":" + cfg.Port)
	if err != nil {
		return
	}
}
