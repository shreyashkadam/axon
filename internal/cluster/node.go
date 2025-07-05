package cluster

import (
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

	return node, nil
}