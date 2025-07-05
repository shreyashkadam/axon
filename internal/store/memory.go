package store

import (
	"fmt"
	"sync"
)

// InMemoryStore is a simple in-memory key-value store.
type InMemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string][]byte),
	}
}

// Put stores a key-value pair.
func (s *InMemoryStore) Put(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[string(key)] = value
	return nil
}

// Get retrieves a value by its key.
func (s *InMemoryStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

// Delete removes a key-value pair.
func (s *InMemoryStore) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, string(key))
	return nil
}

// GetAll retrieves all key-value pairs.
func (s *InMemoryStore) GetAll() (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent race conditions on the caller's side
	dataCopy := make(map[string][]byte, len(s.data))
	for k, v := range s.data {
		dataCopy[k] = v
	}
	return dataCopy, nil
}

// Close is a no-op for the in-memory store.
func (s *InMemoryStore) Close() error {
	return nil
}