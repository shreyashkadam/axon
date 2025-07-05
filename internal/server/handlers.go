package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
	"kvstore/internal/cluster"
)

// KeyValue represents the JSON payload for a PUT request.
type KeyValue struct {
	Value string `json:"value" binding:"required"`
}

// putHandler handles storing a key-value pair.
// It delegates to the cluster node if in cluster mode, otherwise writes locally.
func (s *Server) putHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var kv KeyValue
	if err := c.ShouldBindJSON(&kv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if s.isCluster {
		err = s.node.Put([]byte(key), []byte(kv.Value))
	} else {
		err = s.store.Put([]byte(key), []byte(kv.Value))
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// getHandler handles retrieving a value by its key.
func (s *Server) getHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var value []byte
	var err error
	if s.isCluster {
		value, err = s.node.Get([]byte(key))
	} else {
		value, err = s.store.Get([]byte(key))
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key, "value": string(value)})
}

// deleteHandler handles removing a key-value pair.
func (s *Server) deleteHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var err error
	if s.isCluster {
		err = s.node.Delete([]byte(key))
	} else {
		err = s.store.Delete([]byte(key))
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// getAllHandler retrieves all key-value pairs from the local store.
func (s *Server) getAllHandler(c *gin.Context) {
	// Note: This always queries the local node's store.
	values, err := s.store.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for k, v := range values {
		result[k] = string(v)
	}

	c.JSON(http.StatusOK, result)
}

// --- Internal Cluster Handlers ---

// internalOperationHandler handles internal operations from other nodes.
func (s *Server) internalOperationHandler(c *gin.Context) {
	var op cluster.Operation
	if err := c.ShouldBindJSON(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch op.Type {
	case cluster.OpPut:
		if err := s.store.Put(op.Key, op.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case cluster.OpDelete:
		if err := s.store.Delete(op.Key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown operation type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// internalGetHandler handles internal GET requests from other nodes.
func (s *Server) internalGetHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	value, err := s.store.Get([]byte(key))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", value)
}

// internalStatusHandler returns the status of the current node.
func (s *Server) internalStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"node_id":     s.node.ID,
		"is_leader":   s.node.IsLeader(),
		"leader_addr": s.node.LeaderAddr(),
		"consistency": s.consistency,
	})
}

// RaftJoinRequest represents a request to join a Raft cluster.
type RaftJoinRequest struct {
	NodeID   string `json:"node_id"`
	RaftAddr string `json:"raft_addr"`
}

// internalRaftJoinHandler handles requests from nodes to join the Raft cluster.
func (s *Server) internalRaftJoinHandler(c *gin.Context) {
	if !s.node.IsLeader() {
		leaderAddr := s.node.LeaderAddr()
		if leaderAddr == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not leader and no leader available"})
			return
		}
		c.JSON(http.StatusTemporaryRedirect, gin.H{"error": "not the leader", "leader_addr": leaderAddr})
		return
	}

	var req RaftJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	if req.NodeID == "" || req.RaftAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id and raft_addr are required"})
		return
	}

	future := s.node.GetRaft().AddVoter(raft.ServerID(req.NodeID), raft.ServerAddress(req.RaftAddr), 0, 5*time.Second)
	if err := future.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add voter: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "node successfully joined the Raft cluster",
		"consistent": s.consistency,
	})
}

// RaftRemoveRequest represents a request to remove a node from the Raft cluster.
type RaftRemoveRequest struct {
	NodeID string `json:"node_id"`
}

// internalRaftRemoveHandler handles requests to remove a node from the cluster.
func (s *Server) internalRaftRemoveHandler(c *gin.Context) {
	if !s.node.IsLeader() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not the leader"})
		return
	}

	var req RaftRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	if req.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	log.Printf("Leader received request to remove node %s from the cluster", req.NodeID)

	future := s.node.GetRaft().RemoveServer(raft.ServerID(req.NodeID), 0, 0)
	if err := future.Error(); err != nil {
		log.Printf("Warning/Error removing server %s: %v", req.NodeID, err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "removal processed"})
}

// internalVersionHandler returns the version information for a key.
func (s *Server) internalVersionHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	_, version, err := s.node.GetRaftStore().GetVersioned([]byte(key))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"version": version})
}