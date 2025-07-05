package cluster

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/memberlist"
	"kvstore/internal/consistent"
)

// Node represents a node in the distributed key-value store
type Node struct {
	ID           string
	BindAddr     string
	BindPort     int
	hashRing     *consistent.Ring
	memberConfig *memberlist.Config
	memberList   *memberlist.Memberlist
	KnownPeers   []string
	dataDir      string
	replicaCount int
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
func NewNode(opts NodeOptions) (*Node, error) {
	node := &Node{
		ID:           opts.ID,
		BindAddr:     opts.BindAddr,
		BindPort:     opts.BindPort,
		dataDir:      opts.DataDir,
		replicaCount: opts.ReplicaCount,
		hashRing:     consistent.NewRing(10),
		KnownPeers:   opts.KnownPeers,
	}

	if err := os.MkdirAll(node.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := node.setupMemberlist(opts.KnownPeers); err != nil {
		return nil, fmt.Errorf("failed to setup memberlist: %w", err)
	}

	return node, nil
}

func (n *Node) setupMemberlist(knownPeers []string) error {
	config := memberlist.DefaultLocalConfig()
	config.Name = n.ID
	config.BindAddr = n.BindAddr
	config.BindPort = n.BindPort + 1 // Use a different port for memberlist
	config.Events = &eventDelegate{node: n}

	ml, err := memberlist.Create(config)
	if err != nil {
		return err
	}

	n.memberList = ml
	n.memberConfig = config

	// Add self to the hash ring
	selfAddr := fmt.Sprintf("%s:%d", n.BindAddr, n.BindPort)
	n.hashRing.AddNode(selfAddr)
	log.Printf("Added self to hash ring: %s", selfAddr)

	if len(knownPeers) > 0 {
		_, err = ml.Join(knownPeers)
		if err != nil {
			log.Printf("Failed to join cluster: %v", err)
		}
	}

	return nil
}

func (n *Node) Shutdown() error {
	if n.memberList != nil {
		_ = n.memberList.Leave(time.Second)
		_ = n.memberList.Shutdown()
	}
	return nil
}

// eventDelegate handles memberlist events
type eventDelegate struct {
	node *Node
}

func (e *eventDelegate) NotifyJoin(n *memberlist.Node) {
	log.Printf("Node joined: %s", n.Name)
	// The port in memberlist is the gossip port, we need the API port
	nodeAddr := fmt.Sprintf("%s:%d", n.Addr.String(), e.node.BindPort)
	e.node.hashRing.AddNode(nodeAddr)
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	log.Printf("Node left: %s", n.Name)
	nodeAddr := fmt.Sprintf("%s:%d", n.Addr.String(), e.node.BindPort)
	e.node.hashRing.RemoveNode(nodeAddr)
}

func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	// Not implemented
}