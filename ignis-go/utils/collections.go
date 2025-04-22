package utils

type Set[T comparable] map[T]struct{}

func (s Set[T]) Values() []T {
	keys := make([]T, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}
	return keys
}

func (s Set[T]) Add(value T) bool {
	if _, ok := s[value]; ok {
		return false
	}
	s[value] = struct{}{}
	return true
}

func (s Set[T]) Remove(value T) bool {
	if _, ok := s[value]; !ok {
		return false
	}
	delete(s, value)
	return true
}

func (s Set[T]) Contains(value T) bool {
	_, ok := s[value]
	return ok
}

func (s Set[T]) Len() int {
	return len(s)
}

func (s Set[T]) Empty() bool {
	return len(s) == 0
}

func (s Set[T]) Copy() Set[T] {
	return MakeSetFromSlice(s.Values())
}

func MakeSet[T comparable]() Set[T] {
	return make(Set[T])
}

func MakeSetFromSlice[T comparable](slice []T) Set[T] {
	set := make(Set[T])
	for _, value := range slice {
		set[value] = struct{}{}
	}
	return set
}

type Map[K comparable, V any] map[K]V

func (s Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}
	return keys
}

func (s Map[K, V]) KeySet() Set[K] {
	return MakeSetFromSlice(s.Keys())
}

func (s Map[K, V]) Put(key K, value V) bool {
	if _, ok := s[key]; ok {
		return false
	}

	s[key] = value
	return true
}

func (s Map[K, V]) Get(key K) (v V, ok bool) {
	v, ok = s[key]
	return
}

func (s Map[K, V]) Remove(value K) bool {
	if _, ok := s[value]; !ok {
		return false
	}
	delete(s, value)
	return true
}

func (s Map[K, V]) Contains(value K) bool {
	_, ok := s[value]
	return ok
}

func (s Map[K, V]) ComputeIfAbsent(key K, producer func() V) V {
	if v, ok := s[key]; ok {
		return v
	}
	s[key] = producer()
	return s[key]
}

func (s Map[K, V]) Values() []V {
	values := make([]V, 0, len(s))
	for _, value := range s {
		values = append(values, value)
	}
	return values
}

func (s Map[K, V]) Len() int {
	return len(s)
}

func (s Map[K, V]) Empty() bool {
	return len(s) == 0
}

func MakeMap[K comparable, V any]() Map[K, V] {
	return make(Map[K, V])
}
