package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"kvstore/internal/cluster"
	"kvstore/internal/server"
	"kvstore/internal/store"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8000, "Port to listen on")
	dbPath := flag.String("db-path", "", "Path to database file (defaults to ./data/data_<port>.db)")
	dataDir := flag.String("data-dir", "./data", "Directory for node data")
	clusterMode := flag.Bool("cluster-mode", false, "Enable cluster mode")
	nodeID := flag.String("node-id", "", "Node ID (defaults to localhost:<port>)")
	peers := flag.String("peers", "", "Comma-separated list of peer addresses")
	bindAddr := flag.String("bind-addr", "localhost", "Bind address for the node")

	flag.Parse()

	// Set default values
	if *dbPath == "" {
		*dbPath = fmt.Sprintf("./data/data_%d.db", *port)
	}

	if *nodeID == "" {
		*nodeID = fmt.Sprintf("%s:%d", *bindAddr, *port)
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

	var srv *server.Server
	var node *cluster.Node

	if *clusterMode {
		peersList := []string{}
		if *peers != "" {
			peersList = strings.Split(*peers, ",")
		}

		nodeDataDir := filepath.Join(*dataDir, *nodeID)
		if err := os.MkdirAll(nodeDataDir, 0755); err != nil {
			log.Fatalf("Failed to create node data directory: %v", err)
		}

		nodeOpts := cluster.NodeOptions{
			ID:           *nodeID,
			BindAddr:     *bindAddr,
			BindPort:     *port,
			DataDir:      nodeDataDir,
			ReplicaCount: 2, // Default, will be configurable later
			KnownPeers:   peersList,
		}

		node, err = cluster.NewNode(nodeOpts)
		if err != nil {
			log.Fatalf("Failed to initialize cluster node: %v", err)
		}
		defer node.Shutdown()

		// Create a server for a distributed node
		srv = server.New(persistentStore, node)
		log.Printf("Starting distributed key-value store")
	} else {
		// Create a server for a single node
		srv = server.New(persistentStore, nil)
		log.Printf("Starting single-node key-value store")
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %s, shutting down...", sig)
		if node != nil {
			_ = node.Shutdown()
		}
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