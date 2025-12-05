package main

import (
	"log"
	"virtual-fitcheck/internal/api"
	"virtual-fitcheck/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize API server
	server := api.NewServer(cfg)

	// Start server
	log.Printf("Starting server on port %s", cfg.Port)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
