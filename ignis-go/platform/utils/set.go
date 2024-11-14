package utils

type Set[T comparable] map[T]struct{}

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

func (s Set[T]) Values() []T {
	values := make([]T, 0, len(s))
	for value := range s {
		values = append(values, value)
	}
	return values
}

func (s Set[T]) Len() int {
	return len(s)
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
