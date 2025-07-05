package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"kvstore/internal/server"
	"kvstore/internal/store"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8000, "Port to listen on")
	dbPath := flag.String("db-path", "", "Path to database file (defaults to ./data/data_<port>.db)")
	flag.Parse()

	// Set default values
	if *dbPath == "" {
		*dbPath = fmt.Sprintf("./data/data_%d.db", *port)
	}

	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize the persistent storage engine
	persistentStore, err := store.NewPersistentStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer persistentStore.Close()

	// Create a server for a single node
	srv := server.New(persistentStore)
	log.Printf("Starting single-node key-value store")

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %s, shutting down...", sig)
		_ = persistentStore.Close()
		os.Exit(0)
	}()

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}