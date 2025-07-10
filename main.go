package main

import (
	"websocket-backend/internal/configs"
	"websocket-backend/internal/helpers"
	"websocket-backend/internal/websocket"
	"websocket-backend/routes"

	log "github.com/sirupsen/logrus"
)

func main() {

	// Init log
	helpers.InitLogger()

	r := routes.SetupRouter()

	// Read config
	cf := configs.Config{}
	err := cf.Load("internal/configs/config.json")
	if err != nil {
		panic(err)
	}

	// Load config for websocket
	websocket.LoadAllowedCmds(cf)
	websocket.LoadBlacklistCmds(cf)

	// Run websocket server
	if err := r.Run(":65432"); err != nil {
		log.Printf("WebSocket server failed: %v", err)
		return
	}
}
