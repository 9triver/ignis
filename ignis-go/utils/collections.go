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
