package main

import (
	"flag"
	"fmt"
	"log"

	"kvstore/internal/server"
	"kvstore/internal/store"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8000, "Port to listen on")
	flag.Parse()

	// Initialize the in-memory storage engine
	inMemoryStore := store.NewInMemoryStore()

	// Create a server for a single node
	srv := server.New(inMemoryStore)
	log.Printf("Starting single-node key-value store")

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}