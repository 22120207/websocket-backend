package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"websocket-backend/internal/config"
	"websocket-backend/pkg/utils"
	"websocket-backend/routes"
)

func main() {
	utils.SetupLogger()
	utils.Info("Starting WebSocket backend application...")

	cfg, err := config.LoadConfig()
	if err != nil {
		utils.Error("Failed to load configuration:", err)
		log.Fatalf("Failed to load configuration: %v", err)
	}
	utils.Info("Configuration loaded. Port:", cfg.Port, "Inventory Targets:", len(cfg.Target))

	// Create a new Routes instance with the loaded configuration
	appRoutes := routes.NewRoutes(cfg)
	r := appRoutes.Setup()

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	utils.Info("HTTP server starting on", serverAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		utils.Error("HTTP server failed to start:", err)
		log.Fatalf("HTTP server failed to start: %v", err)
	}
	utils.Info("HTTP server shut down.")
}
