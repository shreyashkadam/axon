package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kvstore/internal/server"
	"kvstore/internal/store"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8000, "Port to listen on")
	flag.Parse()

	// Initialize the in-memory storage engine
	storage := store.NewInMemoryStore()
	defer storage.Close()

	// Create a server for a single node
	srv := server.New(storage)
	log.Printf("Starting single-node key-value store")

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %s, shutting down...", sig)
		_ = storage.Close()
		os.Exit(0)
	}()

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}