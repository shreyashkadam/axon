package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/hashicorp/raft"
	"kvstore/internal/store"
)

// Operation types
const (
	OpPut    = "PUT"
	OpDelete = "DELETE"
)

// Operation represents a distributed operation
type Operation struct {
	Type  string `json:"type"`
	Key   []byte `json:"key"`
	Value []byte `json:"value,omitempty"`
}

// FSM implements the raft.FSM interface for our key-value store
type FSM struct {
	persistentStore *store.PersistentStore
}

// NewFSM creates a new FSM with the given storage
func NewFSM(s *store.PersistentStore) *FSM {
	if s == nil {
		panic("persistent store must not be nil")
	}
	return &FSM{
		persistentStore: s,
	}
}

// Apply applies a Raft log entry to the key-value store
func (fsm *FSM) Apply(raftLog *raft.Log) interface{} {
	var op Operation
	if err := json.Unmarshal(raftLog.Data, &op); err != nil {
		log.Printf("Failed to unmarshal Raft log entry: %v\n", err)
		return nil
	}
	// Placeholder: Logic to apply the operation will be added later.
	log.Printf("Applying operation: type=%s, key=%s", op.Type, string(op.Key))
	return nil
}

// Snapshot returns a snapshot of the key-value store
func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	// Placeholder: Snapshot logic will be implemented later.
	return &fsmSnapshot{}, nil
}

// Restore restores the key-value store to a previous state
func (fsm *FSM) Restore(rc io.ReadCloser) error {
	// Placeholder: Restore logic will be implemented later.
	defer rc.Close()
	_, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	return nil
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct{}

// Persist persists the snapshot to the given sink
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	// Placeholder: Persist logic will be implemented later.
	sink.Close()
	return nil
}

// Release releases resources associated with the snapshot
func (s *fsmSnapshot) Release() {}