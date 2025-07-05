package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"kvstore/internal/consistent"
)

const (
	raftTimeout      = 10 * time.Second
	raftRetainSnaps  = 2
	raftMaxPool      = 3
	raftTransTimeout = 10 * time.Second
)

// Node represents a node in the distributed key-value store
type Node struct {
	ID            string
	BindAddr      string
	BindPort      int
	hashRing      *consistent.Ring
	memberConfig  *memberlist.Config
	memberList    *memberlist.Memberlist
	KnownPeers    []string
	raft          *raft.Raft
	raftConfig    *raft.Config
	raftTransport *raft.NetworkTransport
	raftLogStore  *raftboltdb.BoltStore
	raftStore     *RaftStore
	raftDir       string
	BootstrapRaft bool
	JoinPeer      string
	httpClient    *http.Client
	dataDir       string
	replicaCount  int
}

// NodeOptions contains options for creating a new node
type NodeOptions struct {
	ID            string
	BindAddr      string
	BindPort      int
	DataDir       string
	ReplicaCount  int
	KnownPeers    []string
	BootstrapRaft bool
	JoinPeer      string
}

// NewNode creates a new node
func NewNode(opts NodeOptions, fsm *FSM) (*Node, error) {
	if fsm == nil {
		return nil, fmt.Errorf("fsm cannot be nil")
	}

	node := &Node{
		ID:           opts.ID,
		BindAddr:     opts.BindAddr,
		BindPort:     opts.BindPort,
		dataDir:      opts.DataDir,
		replicaCount: opts.ReplicaCount,
		hashRing:     consistent.NewRing(10),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		raftStore:     fsm.raftStore,
		KnownPeers:    opts.KnownPeers,
		raftDir:       filepath.Join(opts.DataDir, fmt.Sprintf("node-%s", opts.ID)),
		BootstrapRaft: opts.BootstrapRaft,
		JoinPeer:      opts.JoinPeer,
	}

	if err := os.MkdirAll(node.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := node.setupMemberlist(opts.KnownPeers); err != nil {
		return nil, fmt.Errorf("failed to setup memberlist: %w", err)
	}

	if err := node.setupRaft(fsm); err != nil {
		return nil, fmt.Errorf("failed to setup Raft: %w", err)
	}

	return node, nil
}

func (n *Node) setupMemberlist(knownPeers []string) error {
	config := memberlist.DefaultLocalConfig()
	config.Name = n.ID
	config.BindAddr = n.BindAddr
	config.BindPort = n.BindPort + 1
	config.Events = &eventDelegate{node: n}

	ml, err := memberlist.Create(config)
	if err != nil {
		return err
	}

	n.memberList = ml
	n.memberConfig = config

	nodeAddr := fmt.Sprintf("%s:%d", n.BindAddr, n.BindPort)
	n.hashRing.AddNode(nodeAddr)
	log.Printf("Added self to hash ring: %s", nodeAddr)

	if len(knownPeers) > 0 {
		_, err = ml.Join(knownPeers)
		if err != nil {
			log.Printf("Failed to join cluster: %v", err)
		}
	}

	return nil
}

func (n *Node) setupRaft(fsm *FSM) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(n.ID)
	config.HeartbeatTimeout = 1000 * time.Millisecond
	config.ElectionTimeout = 1000 * time.Millisecond
	config.LeaderLeaseTimeout = 500 * time.Millisecond
	config.CommitTimeout = 200 * time.Millisecond

	log.Printf("Setting up Raft with ID: %s", n.ID)
	n.raftConfig = config

	raftAddr := fmt.Sprintf("%s:%d", n.BindAddr, n.BindPort+2)
	log.Printf("Raft transport listening on %s", raftAddr)

	transport, err := raft.NewTCPTransport(raftAddr, nil, raftMaxPool, raftTransTimeout, nil)
	if err != nil {
		return fmt.Errorf("failed to create Raft transport: %w", err)
	}
	n.raftTransport = transport

	n.raftDir = filepath.Join(n.dataDir, fmt.Sprintf("node-%s", n.ID))
	if err := os.MkdirAll(n.raftDir, 0755); err != nil {
		return fmt.Errorf("failed to create Raft directory: %w", err)
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(n.raftDir, "raft-log.bolt"))
	if err != nil {
		return fmt.Errorf("failed to create Raft log store: %w", err)
	}
	n.raftLogStore = logStore

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(n.raftDir, "raft-stable.bolt"))
	if err != nil {
		return fmt.Errorf("failed to create Raft stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(n.raftDir, raftRetainSnaps, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}

	n.raft, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create Raft instance: %w", err)
	}

	if n.BootstrapRaft {
		log.Printf("Node %s is bootstrapping new Raft cluster as leader", n.ID)
		peerConfig := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       config.LocalID,
					Address:  transport.LocalAddr(),
				},
			},
		}

		bootstrapFuture := n.raft.BootstrapCluster(peerConfig)
		if err := bootstrapFuture.Error(); err != nil {
			log.Printf("Warning: failed to bootstrap Raft cluster: %v", err)
			if !strings.Contains(err.Error(), "already") {
				return fmt.Errorf("failed to bootstrap Raft cluster: %w", err)
			}
		} else {
			log.Printf("Successfully bootstrapped Raft cluster as leader")
		}
	} else if n.JoinPeer != "" {
		go func() {
			log.Printf("Node %s starting join process to peer %s", n.ID, n.JoinPeer)
			ticker := time.NewTicker(5 * time.Second) // Retry every 5 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					// If we have a leader, it means we have successfully joined the cluster.
					if n.raft.Leader() != "" {
						log.Printf("Node %s has detected a leader. Join process complete.", n.ID)
						return // Exit the goroutine
					}

					log.Printf("Node %s attempting to join cluster via peer %s...", n.ID, n.JoinPeer)
					err := n.joinRaftCluster(n.JoinPeer)
					if err != nil {
						log.Printf("Join attempt for node %s failed: %v. Will retry.", n.ID, err)
					} else {
						log.Printf("Node %s successfully SENT join request. Waiting for confirmation...", n.ID)
					}
				}
			}
		}()
	}

	log.Printf("Raft setup complete for node %s", n.ID)
	return nil
}

func (n *Node) joinRaftCluster(peerAddr string) error {
	localRaftAddr := fmt.Sprintf("%s:%d", n.BindAddr, n.BindPort+2)
	joinRequest := struct {
		NodeID   string `json:"node_id"`
		RaftAddr string `json:"raft_addr"`
	}{
		NodeID:   n.ID,
		RaftAddr: localRaftAddr,
	}

	joinData, err := json.Marshal(joinRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal join request: %w", err)
	}

	joinURL := fmt.Sprintf("http://%s/internal/raft/join", peerAddr)
	resp, err := n.httpClient.Post(joinURL, "application/json", bytes.NewBuffer(joinData))
	if err != nil {
		return fmt.Errorf("failed to send join request to %s: %w", peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Success!
	}

	// If we were redirected, we need to update our JoinPeer to the new leader address.
	if resp.StatusCode == http.StatusTemporaryRedirect {
		var redirectInfo struct {
			LeaderAddr string `json:"leader_addr"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&redirectInfo); err == nil && redirectInfo.LeaderAddr != "" {
			log.Printf("Join target %s is not leader. Redirected to leader at %s.", peerAddr, redirectInfo.LeaderAddr)
			n.JoinPeer = redirectInfo.LeaderAddr // Update so the next retry hits the correct leader.
			return fmt.Errorf("redirected to new leader: %s", redirectInfo.LeaderAddr)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return fmt.Errorf("join request to %s failed with status %d: %s", peerAddr, resp.StatusCode, string(body))
}

func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

func (n *Node) LeaderAddr() string {
	if n.raft == nil {
		return ""
	}

	raftLeaderAddr := string(n.raft.Leader())
	if raftLeaderAddr == "" {
		return ""
	}

	parts := strings.Split(raftLeaderAddr, ":")
	if len(parts) != 2 {
		return raftLeaderAddr
	}

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return raftLeaderAddr
	}

	httpPort := port - 2
	httpAddr := fmt.Sprintf("%s:%d", host, httpPort)

	return httpAddr
}

func (n *Node) GetRaft() *raft.Raft {
	return n.raft
}

func (n *Node) GetRaftStore() *RaftStore {
	return n.raftStore
}

func (n *Node) Put(key, value []byte) error {
	log.Printf("Put request for key: %s", string(key))

	if !n.IsLeader() {
		leaderAddr := n.LeaderAddr()
		if leaderAddr == "" {
			return fmt.Errorf("no leader available for put")
		}
		log.Printf("Forwarding put request to leader at %s", leaderAddr)
		return n.sendPutToLeader(leaderAddr, key, value)
	}

	log.Printf("We are the Raft leader, storing key %s via Raft consensus", string(key))
	op := Operation{Type: OpPut, Key: key, Value: value}
	opData, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	applyFuture := n.raft.Apply(opData, raftTimeout)
	if err := applyFuture.Error(); err != nil {
		log.Printf("Failed to apply operation to Raft: %v", err)
		return fmt.Errorf("failed to apply operation to Raft: %w", err)
	}
	log.Printf("Successfully stored key %s", string(key))
	return nil
}

func (n *Node) Get(key []byte) ([]byte, error) {
	if n.IsLeader() {
		return n.raftStore.Get(key)
	}

	leaderAddr := n.LeaderAddr()
	if leaderAddr != "" {
		return n.sendGetToLeader(leaderAddr, key)
	}
	return nil, fmt.Errorf("no leader available for strong consistency read")
}

func (n *Node) Delete(key []byte) error {
	op := Operation{Type: OpDelete, Key: key}
	opBytes, err := json.Marshal(op)
	if err != nil {
		return err
	}

	future := n.raft.Apply(opBytes, raftTimeout)
	if err := future.Error(); err != nil {
		return err
	}
	return nil
}

func (n *Node) sendPutToLeader(leaderAddr string, key, value []byte) error {
	url := fmt.Sprintf("http://%s/kv/%s", leaderAddr, string(key))
	reqBody := map[string]string{"value": string(value)}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to send put: %s", string(body))
	}
	return nil
}

func (n *Node) sendGetToLeader(nodeAddr string, key []byte) ([]byte, error) {
	url := fmt.Sprintf("http://%s/internal/get/%s", nodeAddr, key)
	resp, err := n.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get key from node (status %d): %s", resp.StatusCode, string(body))
	}
	return ioutil.ReadAll(resp.Body)
}


func (n *Node) Shutdown() error {
	if n.raft != nil {
		_ = n.raft.Shutdown().Error()
	}
	if n.memberList != nil {
		_ = n.memberList.Leave(time.Second)
		_ = n.memberList.Shutdown()
	}
	if n.raftLogStore != nil {
		_ = n.raftLogStore.Close()
	}
	return nil
}

type eventDelegate struct {
	node *Node
}

func (e *eventDelegate) NotifyJoin(n *memberlist.Node) {
	log.Printf("Node joined: %s", n.Name)
	nodeAddr := fmt.Sprintf("%s:%d", n.Addr.String(), e.node.BindPort)
	e.node.hashRing.AddNode(nodeAddr)
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	log.Printf("Node left: %s", n.Name)
	nodeAddr := fmt.Sprintf("%s:%d", n.Addr.String(), e.node.BindPort)
	e.node.hashRing.RemoveNode(nodeAddr)
}

func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {}