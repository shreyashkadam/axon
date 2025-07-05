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
	Type    string `json:"type"`
	Key     []byte `json:"key"`
	Value   []byte `json:"value,omitempty"`
	Version uint64 `json:"version,omitempty"` // Raft log index where this operation was committed
}

// FSM implements the raft.FSM interface for our key-value store
type FSM struct {
	persistentStore *store.PersistentStore // Renamed from 'store'
	raftStore       *RaftStore             // In-memory store for Raft-replicated data
}

// NewFSM creates a new FSM with the given storage
func NewFSM(s *store.PersistentStore) *FSM {
	if s == nil {
		panic("persistent store must not be nil")
	}
	return &FSM{
		persistentStore: s,
		raftStore:       NewRaftStore(),
	}
}

// Apply applies a Raft log entry to the key-value store
func (fsm *FSM) Apply(raftLog *raft.Log) interface{} {
	if fsm.persistentStore == nil || fsm.raftStore == nil {
		fmt.Printf("FSM has nil store(s), cannot apply operations\n")
		return fmt.Errorf("fsm has nil store(s)")
	}

	var op Operation
	if err := json.Unmarshal(raftLog.Data, &op); err != nil {
		fmt.Printf("Failed to unmarshal Raft log entry: %v\n", err)
		return fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	op.Version = raftLog.Index

	fmt.Printf("FSM applying operation: type=%s, key=%s, index=%d, term=%d, version=%d\n",
		op.Type, string(op.Key), raftLog.Index, raftLog.Term, op.Version)

	switch op.Type {
	case OpPut:
		if err := fsm.raftStore.Put(op.Key, op.Value, op.Version); err != nil {
			fmt.Printf("FSM failed to put value in Raft store: %v\n", err)
			return fmt.Errorf("failed to put value in raft store: %w", err)
		}

		if err := fsm.persistentStore.Put(op.Key, op.Value); err != nil {
			fmt.Printf("FSM failed to put value in persistent store: %v\n", err)
			return fmt.Errorf("failed to put value in persistent store: %w", err)
		}

		fmt.Printf("FSM successfully put key: %s (version %d)\n", string(op.Key), op.Version)
		return nil

	case OpDelete:
		if err := fsm.raftStore.Delete(op.Key, op.Version); err != nil {
			fmt.Printf("FSM failed to delete value from Raft store: %v\n", err)
		}

		if err := fsm.persistentStore.Delete(op.Key); err != nil {
			fmt.Printf("FSM failed to delete value from persistent store: %v\n", err)
			return fmt.Errorf("failed to delete value: %w", err)
		}

		fmt.Printf("FSM successfully deleted key: %s (version %d)\n", string(op.Key), op.Version)
		return nil

	default:
		fmt.Printf("FSM received unknown operation type: %s\n", op.Type)
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// Snapshot returns a snapshot of the key-value store
func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{
		store:     fsm.persistentStore,
		raftStore: fsm.raftStore,
	}, nil
}

// Restore restores the key-value store to a previous state
func (fsm *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	snapshot, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}

	var snapshotData struct {
		RaftStore []struct {
			Key   string
			Value []byte
		}
		RegularStore []struct {
			Key   []byte
			Value []byte
		}
	}

	if err := json.Unmarshal(snapshot, &snapshotData); err != nil {
		return err
	}

	for _, kv := range snapshotData.RaftStore {
		if err := fsm.raftStore.Put([]byte(kv.Key), kv.Value, 1); err != nil {
			fmt.Printf("Error restoring key %s to RaftStore: %v\n", kv.Key, err)
		}
	}

	for _, kv := range snapshotData.RegularStore {
		if err := fsm.persistentStore.Put(kv.Key, kv.Value); err != nil {
			fmt.Printf("Error restoring key %s to regular store: %v\n", string(kv.Key), err)
		}
	}

	return nil
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct {
	store     *store.PersistentStore
	raftStore *RaftStore
}

// Persist persists the snapshot to the given sink
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		snapshot := struct {
			RaftStore []struct {
				Key   string
				Value []byte
			}
			RegularStore []struct {
				Key   []byte
				Value []byte
			}
		}{}

		if s.raftStore != nil {
			raftData := s.raftStore.GetAll()
			for k, v := range raftData {
				snapshot.RaftStore = append(snapshot.RaftStore, struct {
					Key   string
					Value []byte
				}{Key: k, Value: v})
			}
		}

		if s.store != nil {
			storeData, err := s.store.GetAll()
			if err != nil {
				return fmt.Errorf("failed to get data from regular store: %w", err)
			}
			for k, v := range storeData {
				snapshot.RegularStore = append(snapshot.RegularStore, struct {
					Key   []byte
					Value []byte
				}{Key: []byte(k), Value: v})
			}
		}

		if err := json.NewEncoder(sink).Encode(snapshot); err != nil {
			return fmt.Errorf("failed to encode snapshot: %w", err)
		}

		return sink.Close()
	}()

	if err != nil {
		log.Printf("Error persisting snapshot: %v", err)
		sink.Cancel()
		return err
	}

	return nil
}

// Release releases resources associated with the snapshot
func (s *fsmSnapshot) Release() {}