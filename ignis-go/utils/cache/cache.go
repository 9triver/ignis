package cache

type Controller[K comparable, V any] interface {
	Contains(key K) bool
	Get(key K) (value V, ok bool)
	Insert(key K, value V)
	Capacity() int
	Len() int
}
