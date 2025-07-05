package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:build
var uiAssets embed.FS

const basePort = 9000
const portStep = 10

// NodeConfig holds the configuration for a single kvstore node.
type NodeConfig struct {
	ID      string `json:"id"`
	APIPort int    `json:"api_port"`
	RaftPort int    `json:"-"`
	BindAddr string `json:"-"`
	DataDir string `json:"-"`
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
	httpClient   *http.Client
	nextBasePort int
}

func NewNodeManager(kvBinary string) *NodeManager {
	return &NodeManager{
		nodes:        make(map[string]*NodeProcess),
		allConfigs:   make(map[string]NodeConfig),
		kvBinary:     kvBinary,
		httpClient:   &http.Client{Timeout: 2 * time.Second},
		nextBasePort: basePort,
	}
}

func (nm *NodeManager) findFreePorts() (int, int, int, error) {
	for i := 0; i < 100; i++ {
		apiPort := nm.nextBasePort
		memberlistPort := apiPort + 1
		raftPort := apiPort + 2
		nm.nextBasePort += portStep

		l1, err1 := net.Listen("tcp", fmt.Sprintf(":%d", apiPort))
		if err1 != nil {
			continue
		}
		l2, err2 := net.Listen("tcp", fmt.Sprintf(":%d", memberlistPort))
		if err2 != nil {
			l1.Close()
			continue
		}
		l3, err3 := net.Listen("tcp", fmt.Sprintf(":%d", raftPort))
		if err3 != nil {
			l1.Close()
			l2.Close()
			continue
		}

		l1.Close()
		l2.Close()
		l3.Close()
		return apiPort, memberlistPort, raftPort, nil
	}
	return 0, 0, 0, fmt.Errorf("could not find 3 consecutive free ports")
}

func (nm *NodeManager) AddNewNode(nodeID string) (NodeConfig, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.allConfigs[nodeID]; exists {
		return NodeConfig{}, fmt.Errorf("node with ID %s already exists", nodeID)
	}

	apiPort, _, raftPort, err := nm.findFreePorts()
	if err != nil {
		return NodeConfig{}, fmt.Errorf("failed to allocate ports: %w", err)
	}

	config := NodeConfig{
		ID:       nodeID,
		APIPort:  apiPort,
		RaftPort: raftPort,
		BindAddr: "127.0.0.1",
		DataDir:  fmt.Sprintf("./data/%s", nodeID),
	}

	nm.allConfigs[config.ID] = config

	go nm.startNodeWithRecovery(config)

	return config, nil
}

func (nm *NodeManager) startNode(config NodeConfig, forceBootstrap bool, joinPeer string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.nodes[config.ID]; exists {
		return fmt.Errorf("node %s is already running", config.ID)
	}

	_ = os.RemoveAll(config.DataDir)
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory for %s: %w", config.ID, err)
	}

	log.Printf("Starting node %s on port %d...", config.ID, config.APIPort)

	args := []string{
		"--port", fmt.Sprintf("%d", config.APIPort),
		"--node-id", config.ID,
		"--data-dir", config.DataDir,
		"--cluster-mode",
		"--consistency", "strong",
		"--bind-addr", config.BindAddr,
	}

	if forceBootstrap {
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
			_ = proc.Cmd.Process.Kill()
		}
	}
	return nil
}

func (nm *NodeManager) DecommissionNode(nodeID string) error {
	nm.mu.Lock()
	log.Printf("Decommissioning node %s...", nodeID)

	delete(nm.allConfigs, nodeID)

	_, isRunning := nm.nodes[nodeID]
	nm.mu.Unlock()

	if isRunning {
		leaderAPIAddr, err := nm.FindCurrentLeader()
		if err == nil && leaderAPIAddr != "" {
			log.Printf("Found leader at %s. Requesting removal of node %s.", leaderAPIAddr, nodeID)

			reqBody, _ := json.Marshal(map[string]string{"node_id": nodeID})
			removeURL := fmt.Sprintf("http://%s/internal/raft/remove", leaderAPIAddr)
			_, _ = nm.httpClient.Post(removeURL, "application/json", bytes.NewBuffer(reqBody))
			time.Sleep(1 * time.Second)
		}
		_ = nm.StopNode(nodeID)
	}

	dataDir := fmt.Sprintf("./data/%s", nodeID)
	_ = os.RemoveAll(dataDir)

	return nil
}

func (nm *NodeManager) FindCurrentLeader() (string, error) {
	nm.mu.RLock()
	nodesToCheck := make([]NodeConfig, 0, len(nm.nodes))
	for _, proc := range nm.nodes {
		nodesToCheck = append(nodesToCheck, proc.Config)
	}
	nm.mu.RUnlock()

	if len(nodesToCheck) == 0 {
		return "", fmt.Errorf("no running nodes")
	}

	for _, cfg := range nodesToCheck {
		url := fmt.Sprintf("http://%s:%d/internal/status", cfg.BindAddr, cfg.APIPort)
		resp, err := nm.httpClient.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		var status struct {
			LeaderAddr string `json:"leader_addr"`
			IsLeader   bool   `json:"is_leader"`
		}
		if json.NewDecoder(resp.Body).Decode(&status) == nil {
			if status.IsLeader {
				return fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.APIPort), nil
			}
			if status.LeaderAddr != "" {
				return status.LeaderAddr, nil
			}
		}
	}
	return "", fmt.Errorf("could not determine leader")
}

func (nm *NodeManager) startNodeWithRecovery(config NodeConfig) {
	joinPeer, err := nm.FindCurrentLeader()
	forceBootstrap := false

	// If there is no active leader...
	if err != nil {
		log.Printf("No active leader found for node %s. Checking existing configuration...", config.ID)
		nm.mu.RLock()
		
		// Check if any other nodes are part of the overall configuration.
		// We use `len(nm.allConfigs) > 1` because allConfigs includes the node we are currently starting.
		// If it's greater than 1, it means other nodes exist or have existed.
		isExistingCluster := len(nm.allConfigs) > 1
		nm.mu.RUnlock()

		// If other nodes are part of the configuration, this node should NOT bootstrap.
		// It must try to join an existing peer. This prevents a split-brain.
		// The cluster must be recovered by restarting one of the *original* nodes.
		if isExistingCluster {
			log.Printf("An existing cluster configuration was found but has no leader (lost quorum).")
			log.Printf("Node %s will NOT bootstrap. It will attempt to join the old cluster.", config.ID)
			
			// Find an original peer to try and join.
			nm.mu.RLock()
			for _, existingConfig := range nm.allConfigs {
				if existingConfig.ID != config.ID {
					joinPeer = fmt.Sprintf("%s:%d", existingConfig.BindAddr, existingConfig.APIPort)
					log.Printf("Setting join peer for %s to %s", config.ID, joinPeer)
					break
				}
			}
			nm.mu.RUnlock()

            // If for some reason we couldn't find a peer to join, log an error.
            if joinPeer == "" {
                log.Printf("CRITICAL: Could not find a peer for node %s to join in an existing cluster.", config.ID)
            }

		} else {
			// This is the very first node being added to an empty system. It's safe to bootstrap.
			log.Printf("This appears to be the first node in the system. Node %s will bootstrap as leader.", config.ID)
			forceBootstrap = true
		}
	}

	// This logic remains the same. If a leader was found, joinPeer is already set and this block is skipped.
	if err := nm.startNode(config, forceBootstrap, joinPeer); err != nil {
		log.Printf("ERROR starting node %s: %v", config.ID, err)
		nm.mu.Lock()
		delete(nm.allConfigs, config.ID)
		nm.mu.Unlock()
	}
}

func (nm *NodeManager) GetAllNodeStatuses() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	statuses := make(map[string]interface{})
	for id, cfg := range nm.allConfigs {
		baseStatus := map[string]interface{}{
			"id":        id,
			"api_port":  cfg.APIPort,
			"status":    "offline",
			"is_leader": false,
		}

		if _, running := nm.nodes[id]; running {
			url := fmt.Sprintf("http://%s:%d/internal/status", cfg.BindAddr, cfg.APIPort)
			resp, err := nm.httpClient.Get(url)
			if err == nil {
				defer resp.Body.Close()
				var raftStatus struct {
					IsLeader bool `json:"is_leader"`
				}
				if json.NewDecoder(resp.Body).Decode(&raftStatus) == nil {
					baseStatus["status"] = "online"
					baseStatus["is_leader"] = raftStatus.IsLeader
				}
			}
		}
		statuses[id] = baseStatus
	}
	return statuses
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

// spaHandler serves the single-page application.
type spaHandler struct {
	staticFS fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if _, err := h.staticFS.Open(strings.TrimPrefix(path, "/")); err == nil {
		http.FileServer(http.FS(h.staticFS)).ServeHTTP(w, r)
		return
	}

	r.URL.Path = "/"
	http.FileServer(http.FS(h.staticFS)).ServeHTTP(w, r)
}

func main() {
	kvstoreBinaryName := "kvstore"
	if runtime.GOOS == "windows" {
		kvstoreBinaryName += ".exe"
	}
	_ = os.Remove("kvstore.exe")
	_ = os.Remove("kvstore")

	log.Println("Building kvstore binary...")
	// *** UPDATED BUILD COMMAND ***
	buildCmd := exec.Command("go", "build", "-o", kvstoreBinaryName, "./cmd/kvstore")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to build kvstore binary: %v\n%s", err, string(output))
	}

	_ = os.RemoveAll("./data")

	manager := NewNodeManager("./" + kvstoreBinaryName)

	for i := 1; i <= 3; i++ {
		_, err := manager.AddNewNode(fmt.Sprintf("node%d", i))
		if err != nil {
			log.Fatalf("Failed to create initial node: %v", err)
		}
		if i == 1 {
			time.Sleep(5 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	apiRouter := gin.New()
	api := apiRouter.Group("/api/control")
	{
		api.GET("/nodes", func(c *gin.Context) {
			c.JSON(http.StatusOK, manager.GetAllNodeStatuses())
		})
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
			go manager.startNodeWithRecovery(config)
			c.JSON(http.StatusOK, gin.H{"status": "start signal sent"})
		})
		api.POST("/delete/:nodeID", func(c *gin.Context) {
			if err := manager.DecommissionNode(c.Param("nodeID")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "decommissioned"})
		})
	}

	svelteBuildFS, err := fs.Sub(uiAssets, "build")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for UI: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/control/", apiRouter)
	mux.Handle("/", spaHandler{staticFS: svelteBuildFS})

	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		log.Println("Launcher UI running at http://localhost:8080")
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