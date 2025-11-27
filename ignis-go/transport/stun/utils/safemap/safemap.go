package safemap

import "sync"

type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	M  map[K]V
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		M: make(map[K]V),
	}
}

func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.M[key] = value
}

func (sm *SafeMap[K, V]) Get(key K) (value V, ok bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	value, ok = sm.M[key]
	return
}

func (sm *SafeMap[K, V]) Del(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.M, key)
}

func SetSafeMap[K comparable, V any](m map[K]V, mu *sync.RWMutex, key K, value V) {
	mu.Lock()
	defer mu.Unlock()
	m[key] = value
}

func GetSafeMap[K comparable, V any](m map[K]V, mu *sync.RWMutex, key K) (value V, ok bool) {
	mu.RLock()
	defer mu.RUnlock()

	value, ok = m[key]
	return
}

func DeleteSafeMap[K comparable, V any](m map[K]V, mu *sync.RWMutex, key K) {
	mu.Lock()
	defer mu.Unlock()
	delete(m, key)
}
