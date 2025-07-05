package consistent

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"sort"
	"sync"
)

// ErrEmptyCircle is returned when there are no nodes in the hash ring
var ErrEmptyCircle = errors.New("empty consistent hash circle")

// HashKey generates a hash for a key
func HashKey(key []byte) uint32 {
	hash := md5.Sum(key)
	return binary.LittleEndian.Uint32(hash[:4])
}

// Ring represents a consistent hash ring
type Ring struct {
	mutex    sync.RWMutex
	nodes    map[string]struct{}
	hashRing []uint32
	hashMap  map[uint32]string
	replicas int
}

// NewRing creates a new consistent hash ring
func NewRing(replicas int) *Ring {
	return &Ring{
		nodes:    make(map[string]struct{}),
		hashMap:  make(map[uint32]string),
		replicas: replicas,
	}
}

// AddNode adds a node to the hash ring
func (r *Ring) AddNode(node string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.nodes[node]; ok {
		return
	}

	r.nodes[node] = struct{}{}

	for i := 0; i < r.replicas; i++ {
		key := []byte(node + string(rune(i)))
		hash := HashKey(key)
		r.hashRing = append(r.hashRing, hash)
		r.hashMap[hash] = node
	}

	sort.Slice(r.hashRing, func(i, j int) bool { return r.hashRing[i] < r.hashRing[j] })
}

// RemoveNode removes a node from the hash ring
func (r *Ring) RemoveNode(node string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.nodes[node]; !ok {
		return
	}

	delete(r.nodes, node)

	var newRing []uint32
	for i := 0; i < len(r.hashRing); i++ {
		hash := r.hashRing[i]
		if r.hashMap[hash] != node {
			newRing = append(newRing, hash)
		} else {
			delete(r.hashMap, hash)
		}
	}

	r.hashRing = newRing
}

// GetNode gets the node responsible for the key
func (r *Ring) GetNode(key []byte) (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.hashRing) == 0 {
		return "", ErrEmptyCircle
	}

	hash := HashKey(key)
	idx := r.search(hash)

	return r.hashMap[r.hashRing[idx]], nil
}

// GetNodes gets n nodes responsible for the key, for replication
func (r *Ring) GetNodes(key []byte, n int) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.nodes) < n {
		return nil, errors.New("not enough nodes for requested replication")
	}

	if len(r.hashRing) == 0 {
		return nil, ErrEmptyCircle
	}

	hash := HashKey(key)
	idx := r.search(hash)

	var result []string
	seen := make(map[string]struct{})

	for len(result) < n {
		node := r.hashMap[r.hashRing[idx]]
		if _, ok := seen[node]; !ok {
			seen[node] = struct{}{}
			result = append(result, node)
		}
		idx = (idx + 1) % len(r.hashRing)
	}

	return result, nil
}

// search finds the appropriate index in the hash ring
func (r *Ring) search(hash uint32) int {
	idx := sort.Search(len(r.hashRing), func(i int) bool { return r.hashRing[i] >= hash })

	if idx == len(r.hashRing) {
		idx = 0
	}

	return idx
}

// GetAllNodes returns all node names in the ring
func (r *Ring) GetAllNodes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	nodes := make([]string, 0, len(r.nodes))
	for node := range r.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}