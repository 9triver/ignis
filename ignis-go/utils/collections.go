package utils

import (
	"container/heap"
	"fmt"
	"strings"
)

type Set[T comparable] map[T]struct{}
type LessFunc[T any] func(i, j T) bool

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

func MakeSetFromSlice[T comparable](slice []T) Set[T] {
	set := make(Set[T])
	for _, value := range slice {
		set[value] = struct{}{}
	}
	return set
}

type heapImpl[T any] struct {
	data []T
	less LessFunc[T]
}

func (h *heapImpl[T]) String() string {
	sb := strings.Builder{}
	sb.WriteByte('[')
	for i, v := range h.data {
		sb.WriteString(fmt.Sprintf("%v", v))
		if i != len(h.data)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

var _ heap.Interface = (*heapImpl[int])(nil)

func (h *heapImpl[T]) Len() int {
	return len(h.data)
}

func (h *heapImpl[T]) Less(i int, j int) bool {
	return h.less(h.data[i], h.data[j])
}

func (h *heapImpl[T]) Pop() (ret any) {
	idx := h.Len() - 1
	ret, h.data = h.data[idx], h.data[:idx]
	return
}

func (h *heapImpl[T]) Push(x any) {
	h.data = append(h.data, x.(T))
}

func (h *heapImpl[T]) Swap(i int, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
}

type PQueue[T any] struct {
	impl *heapImpl[T]
}

func (pq PQueue[T]) Push(x T) {
	heap.Push(pq.impl, x)
}

func (pq PQueue[T]) Pop() T {
	return heap.Pop(pq.impl).(T)
}

func (pq PQueue[T]) Remove(i int) T {
	return heap.Remove(pq.impl, i).(T)
}

func (pq PQueue[T]) Len() int {
	return pq.impl.Len()
}

func (pq PQueue[T]) String() string {
	return pq.impl.String()
}

func MakePriorityQueue[T any](less LessFunc[T], elems ...T) PQueue[T] {
	impl := &heapImpl[T]{
		less: less,
		data: elems,
	}
	heap.Init(impl)
	return PQueue[T]{impl}
}
