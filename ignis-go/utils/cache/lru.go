package cache

import "sync"

type lruNode[K, V any] struct {
	Key        K
	Val        V
	Prev, Next *lruNode[K, V]
}

type LRU[K comparable, V any] struct {
	mu         sync.RWMutex
	cap        int
	nodes      map[K]*lruNode[K, V]
	head, tail *lruNode[K, V]
}

func (cache *LRU[K, V]) expireNode(node *lruNode[K, V]) (key K) {
	key = node.Key

	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev

	delete(cache.nodes, key)
	return
}

func (cache *LRU[K, V]) addNode(node *lruNode[K, V]) {
	node.Next = cache.head.Next
	node.Prev = cache.head
	cache.head.Next.Prev = node
	cache.head.Next = node
}

func (cache *LRU[K, V]) useNode(node *lruNode[K, V]) {
	cache.expireNode(node)
	cache.addNode(node)
}

func (cache *LRU[K, V]) Insert(key K, value V) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if node, found := cache.nodes[key]; found {
		node.Val = value
		cache.useNode(node)
		return
	}

	if len(cache.nodes) == cache.cap {
		cache.expireNode(cache.tail.Prev)
	}

	node := &lruNode[K, V]{Key: key, Val: value}
	cache.nodes[key] = node
	cache.addNode(node)
}

func (cache *LRU[K, V]) Contains(key K) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	_, ok := cache.nodes[key]
	if !ok {
		return false
	}
	return true
}

func (cache *LRU[K, V]) Get(key K) (value V, ok bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	node, ok := cache.nodes[key]
	if !ok {
		return
	}

	cache.useNode(node)
	value = node.Val
	ok = true
	return
}

func (cache *LRU[K, V]) Capacity() int {
	return cache.cap
}

func (cache *LRU[K, V]) Len() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return len(cache.nodes)
}

func NewLRU[K comparable, V any](cap int) *LRU[K, V] {
	lru := &LRU[K, V]{
		cap:   cap,
		nodes: make(map[K]*lruNode[K, V]),
		head:  new(lruNode[K, V]),
		tail:  new(lruNode[K, V]),
	}

	lru.head.Next = lru.tail
	lru.tail.Prev = lru.head

	return lru
}
