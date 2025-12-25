package cache

import (
	"sync"
	"time"
)

type timedNode[T any] struct {
	Value     T
	Timestamp time.Time
}

type Timed[K comparable, V any] struct {
	mu      sync.RWMutex
	nodes   map[K]timedNode[V]
	timeout time.Duration
}

func (cache *Timed[K, V]) Contains(key K) bool {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	node, ok := cache.nodes[key]
	if !ok {
		return false
	}

	if time.Since(node.Timestamp) > cache.timeout {
		delete(cache.nodes, key)
		return false
	}

	return true
}

func (cache *Timed[K, V]) Get(key K) (value V, ok bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	node, ok := cache.nodes[key]
	if !ok {
		return
	}

	if time.Since(node.Timestamp) > cache.timeout {
		delete(cache.nodes, key)
		ok = false
		return
	}

	value = node.Value
	return
}

func (cache *Timed[K, V]) Insert(key K, value V) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	node, ok := cache.nodes[key]
	if !ok {
		cache.nodes[key] = timedNode[V]{
			Value:     value,
			Timestamp: time.Now(),
		}
	} else {
		node.Value = value
	}
}

func (cache *Timed[K, V]) Capacity() int {
	return -1 // unbound
}

func (cache *Timed[K, V]) Len() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return len(cache.nodes)
}

func (cache *Timed[K, V]) expireAll() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	var deletes []K
	for key, node := range cache.nodes {
		if time.Since(node.Timestamp) > cache.timeout {
			deletes = append(deletes, key)
		}
	}

	for _, key := range deletes {
		delete(cache.nodes, key)
	}
}

func NewTimed[K comparable, V any](timeout, checkInterval time.Duration) *Timed[K, V] {
	t := &Timed[K, V]{
		nodes:   map[K]timedNode[V]{},
		timeout: timeout,
	}

	go func() {
		for range time.Tick(checkInterval) {
			t.expireAll()
		}
	}()

	return t
}
