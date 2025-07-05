package cluster

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	dataDir       string
	replicaCount  int
}

// NodeOptions contains options for creating a new node
type NodeOptions struct {
	ID           string
	BindAddr     string
	BindPort     int
	DataDir      string
	ReplicaCount int
	KnownPeers   []string
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
		raftStore:    fsm.raftStore,
		KnownPeers:   opts.KnownPeers,
		raftDir:      filepath.Join(opts.DataDir, fmt.Sprintf("node-%s", opts.ID)),
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

	log.Printf("Raft setup complete for node %s", n.ID)
	return nil
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