package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const basePort = 9000
const portStep = 10

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
	mu           sync.RWMutex
	nodes        map[string]*NodeProcess
	allConfigs   map[string]NodeConfig
	kvBinary     string
	nextBasePort int
}

func NewNodeManager(kvBinary string) *NodeManager {
	return &NodeManager{
		nodes:        make(map[string]*NodeProcess),
		allConfigs:   make(map[string]NodeConfig),
		kvBinary:     kvBinary,
		nextBasePort: basePort,
	}
}

func (nm *NodeManager) findFreePort() (int, error) {
	for i := 0; i < 100; i++ {
		port := nm.nextBasePort
		nm.nextBasePort += portStep

		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			l.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("could not find a free port")
}

func (nm *NodeManager) AddNewNode(nodeID string) (NodeConfig, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.allConfigs[nodeID]; exists {
		return NodeConfig{}, fmt.Errorf("node with ID %s already exists", nodeID)
	}

	apiPort, err := nm.findFreePort()
	if err != nil {
		return NodeConfig{}, fmt.Errorf("failed to allocate port: %w", err)
	}

	config := NodeConfig{
		ID:      nodeID,
		APIPort: apiPort,
	}

	nm.allConfigs[config.ID] = config

	// The actual start is now handled via API, but we'll prime the first few
	// in main for simplicity.
	// go nm.startNode(config, false, "127.0.0.1:9000")

	return config, nil
}

func (nm *NodeManager) startNode(config NodeConfig, bootstrap bool, joinPeer string) error {
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
	if err := proc.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		if runtime.GOOS == "windows" {
			// Fallback for Windows
			_ = proc.Cmd.Process.Kill()
		}
	}
	return nil
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

	// --- Initial Cluster Setup ---
	initialConfigs := []NodeConfig{}
	for i := 1; i <= 3; i++ {
		config, err := manager.AddNewNode(fmt.Sprintf("node%d", i))
		if err != nil {
			log.Fatalf("Failed to create initial node config: %v", err)
		}
		initialConfigs = append(initialConfigs, config)
	}

	if err := manager.startNode(initialConfigs[0], true, ""); err != nil {
		log.Fatalf("Failed to start node1: %v", err)
	}
	time.Sleep(5 * time.Second)

	leaderAPIPeer := fmt.Sprintf("127.0.0.1:%d", initialConfigs[0].APIPort)
	for i := 1; i < len(initialConfigs); i++ {
		if err := manager.startNode(initialConfigs[i], false, leaderAPIPeer); err != nil {
			log.Fatalf("Failed to start node %s: %v", initialConfigs[i].ID, err)
		}
		time.Sleep(2 * time.Second)
	}
	// --- End Initial Setup ---

	gin.SetMode(gin.ReleaseMode)
	apiRouter := gin.New()
	api := apiRouter.Group("/api/control")
	{
		api.POST("/add", func(c *gin.Context) {
			var req struct {
				NodeID string `json:"node_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}
			if req.NodeID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
				return
			}

			config, err := manager.AddNewNode(req.NodeID)
			if err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, config)
		})
		api.POST("/stop/:nodeID", func(c *gin.Context) {
			_ = manager.StopNode(c.Param("nodeID"))
			c.JSON(http.StatusOK, gin.H{"status": "stop signal sent"})
		})
		api.POST("/start/:nodeID", func(c *gin.Context) {
			manager.mu.RLock()
			config, exists := manager.allConfigs[c.Param("nodeID")]
			manager.mu.RUnlock()
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
				return
			}
			// For now, new nodes always join the first node. This will be improved.
			go manager.startNode(config, false, "127.0.0.1:9000")
			c.JSON(http.StatusOK, gin.H{"status": "start signal sent"})
		})
	}

	server := &http.Server{Addr: ":8080", Handler: apiRouter}

	go func() {
		log.Println("Launcher API running at http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down launcher...")
	manager.StopAll()
	_ = server.Close()
	log.Println("Launcher exited.")
}