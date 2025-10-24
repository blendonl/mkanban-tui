package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mkanban/internal/daemon"
	"mkanban/internal/infrastructure/config"
)

func main() {
	// Load configuration
	loader, err := config.NewLoader()
	if err != nil {
		log.Fatalf("Failed to create config loader: %v", err)
	}

	cfg, err := loader.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create daemon server
	server, err := daemon.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	fmt.Println("mkanban daemon started")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
	case err := <-errChan:
		fmt.Printf("Server error: %v\n", err)
	}

	// Cleanup
	fmt.Println("Shutting down...")
	if err := server.Stop(); err != nil {
		fmt.Printf("Error stopping server: %v\n", err)
	}
}
