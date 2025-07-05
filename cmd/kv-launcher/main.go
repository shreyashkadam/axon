package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// NodeConfig holds the configuration for a single kvstore node.
type NodeConfig struct {
	ID      string `json:"id"`
	APIPort int    `json:"api_port"`
}

// NodeProcess holds the state for a running node process.
type NodeProcess struct {
	Config NodeConfig
	Cmd    *exec.Cmd
}

// NodeManager manages all the node processes.
type NodeManager struct {
	mu       sync.RWMutex
	nodes    map[string]*NodeProcess
	kvBinary string
}

func NewNodeManager(kvBinary string) *NodeManager {
	return &NodeManager{
		nodes:    make(map[string]*NodeProcess),
		kvBinary: kvBinary,
	}
}

func (nm *NodeManager) StartNode(config NodeConfig, bootstrap bool, joinPeer string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.nodes[config.ID]; exists {
		return fmt.Errorf("node %s is already running", config.ID)
	}

	dataDir := fmt.Sprintf("./data/%s", config.ID)
	_ = os.RemoveAll(dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory for %s: %w", config.ID, err)
	}

	log.Printf("Starting node %s on port %d...", config.ID, config.APIPort)

	args := []string{
		"--port", fmt.Sprintf("%d", config.APIPort),
		"--node-id", config.ID,
		"--data-dir", dataDir,
		"--cluster-mode",
		"--bind-addr", "127.0.0.1",
	}

	if bootstrap {
		args = append(args, "--bootstrap-leader")
	} else if joinPeer != "" {
		args = append(args, "--join-peer", joinPeer)
	}

	cmd := exec.Command(nm.kvBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start node %s: %w", config.ID, err)
	}

	nm.nodes[config.ID] = &NodeProcess{
		Config: config,
		Cmd:    cmd,
	}

	go func() {
		_ = cmd.Wait()
		nm.mu.Lock()
		delete(nm.nodes, config.ID)
		nm.mu.Unlock()
		log.Printf("Process for node %s has terminated.", config.ID)
	}()

	return nil
}

func (nm *NodeManager) StopNode(nodeID string) error {
	nm.mu.Lock()
	proc, exists := nm.nodes[nodeID]
	nm.mu.Unlock()

	if !exists {
		return nil
	}

	log.Printf("Sending stop signal to node %s...", nodeID)
	// Use SIGTERM for graceful shutdown
	return proc.Cmd.Process.Signal(syscall.SIGTERM)
}

func (nm *NodeManager) StopAll() {
	nm.mu.Lock()
	nodeIDs := make([]string, 0, len(nm.nodes))
	for id := range nm.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	nm.mu.Unlock()

	for _, id := range nodeIDs {
		_ = nm.StopNode(id)
	}
}

func main() {
	kvstoreBinaryName := "kvstore"
	if runtime.GOOS == "windows" {
		kvstoreBinaryName += ".exe"
	}
	_ = os.Remove(kvstoreBinaryName)

	log.Println("Building kvstore binary...")
	buildCmd := exec.Command("go", "build", "-o", kvstoreBinaryName, "./cmd/kvstore")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to build kvstore binary: %v\n%s", err, string(output))
	}

	_ = os.RemoveAll("./data")

	manager := NewNodeManager("./" + kvstoreBinaryName)

	// Start a 3-node cluster
	configs := []NodeConfig{
		{ID: "node1", APIPort: 9000},
		{ID: "node2", APIPort: 9010},
		{ID: "node3", APIPort: 9020},
	}

	// Start the first node as the leader
	if err := manager.StartNode(configs[0], true, ""); err != nil {
		log.Fatalf("Failed to start node1: %v", err)
	}
	time.Sleep(5 * time.Second) // Wait for leader to establish

	// Join the other nodes to the leader
	leaderAPIPeer := fmt.Sprintf("127.0.0.1:%d", configs[0].APIPort)
	for i := 1; i < len(configs); i++ {
		if err := manager.StartNode(configs[i], false, leaderAPIPeer); err != nil {
			log.Fatalf("Failed to start node %s: %v", configs[i].ID, err)
		}
		time.Sleep(2 * time.Second)
	}

	log.Println("Cluster is running. Press Ctrl+C to shut down.")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down launcher...")
	manager.StopAll()
	log.Println("Launcher exited.")
}