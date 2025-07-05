package cluster

import (
	"fmt"
	"log"
	"sync"
)

// VersionedValue represents a value with its associated version in the Raft log
type VersionedValue struct {
	Value   []byte // The actual data
	Version uint64 // The Raft log index when this value was last updated
}

// RaftStore provides a simple in-memory store for Raft-replicated data
type RaftStore struct {
	mu          sync.RWMutex
	data        map[string]VersionedValue
	latestIndex uint64 // Tracks the latest applied Raft log index
}

// NewRaftStore creates a new RaftStore
func NewRaftStore() *RaftStore {
	return &RaftStore{
		data:        make(map[string]VersionedValue),
		latestIndex: 0,
	}
}

// Put stores a key-value pair in the Raft store with its version
func (rs *RaftStore) Put(key, value []byte, version uint64) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	log.Printf("RaftStore: storing key %s with version %d", string(key), version)
	rs.data[string(key)] = VersionedValue{
		Value:   value,
		Version: version,
	}

	if version > rs.latestIndex {
		rs.latestIndex = version
	}

	return nil
}

// Get retrieves a value from the Raft store without version information
func (rs *RaftStore) Get(key []byte) ([]byte, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if versionedValue, ok := rs.data[string(key)]; ok {
		log.Printf("RaftStore: found key %s (version %d)", string(key), versionedValue.Version)
		return versionedValue.Value, nil
	}

	log.Printf("RaftStore: key %s not found", string(key))
	return nil, fmt.Errorf("key not found in raft store")
}

// GetVersioned retrieves a value and its version from the Raft store
func (rs *RaftStore) GetVersioned(key []byte) ([]byte, uint64, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if versionedValue, ok := rs.data[string(key)]; ok {
		log.Printf("RaftStore: found key %s with version %d", string(key), versionedValue.Version)
		return versionedValue.Value, versionedValue.Version, nil
	}

	log.Printf("RaftStore: key %s not found", string(key))
	return nil, 0, fmt.Errorf("key not found in raft store")
}

// Delete removes a key-value pair from the Raft store
func (rs *RaftStore) Delete(key []byte, version uint64) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, ok := rs.data[string(key)]; ok {
		log.Printf("RaftStore: deleting key %s at version %d", string(key), version)
		delete(rs.data, string(key))

		if version > rs.latestIndex {
			rs.latestIndex = version
		}

		return nil
	}

	log.Printf("RaftStore: key %s not found for deletion", string(key))
	return fmt.Errorf("key not found in raft store")
}

// GetAll returns all key-value pairs in the Raft store
func (rs *RaftStore) GetAll() map[string][]byte {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	result := make(map[string][]byte, len(rs.data))
	for k, v := range rs.data {
		result[k] = v.Value
	}

	return result
}