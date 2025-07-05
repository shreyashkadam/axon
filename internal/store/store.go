package store

// Store is the interface for key-value storage.
type Store interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	GetAll() (map[string][]byte, error)
	Close() error
}