package store

import (
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	// defaultBucketName is the default bucket name for our key-value pairs
	defaultBucketName = "kvstore"
)

// PersistentStore represents our storage engine using BoltDB
type PersistentStore struct {
	db *bolt.DB
}

// NewPersistentStore creates a new instance of PersistentStore
func NewPersistentStore(dbPath string) (*PersistentStore, error) {
	// Open the BoltDB database
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create the default bucket if it doesn't exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(defaultBucketName))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &PersistentStore{db: db}, nil
}

// Close closes the database
func (s *PersistentStore) Close() error {
	return s.db.Close()
}

// Put stores a key-value pair
func (s *PersistentStore) Put(key, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(defaultBucketName))
		return b.Put(key, value)
	})
}

// Get retrieves a value by its key
func (s *PersistentStore) Get(key []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(defaultBucketName))
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("key not found")
		}

		// Copy the value as it will be invalid outside the transaction
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Delete removes a key-value pair
func (s *PersistentStore) Delete(key []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(defaultBucketName))
		return b.Delete(key)
	})
}

// GetAll retrieves all key-value pairs
func (s *PersistentStore) GetAll() (map[string][]byte, error) {
	result := make(map[string][]byte)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(defaultBucketName))

		return b.ForEach(func(k, v []byte) error {
			key := make([]byte, len(k))
			value := make([]byte, len(v))

			copy(key, k)
			copy(value, v)

			result[string(key)] = value
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}