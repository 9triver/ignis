package utils

import (
	"sync"

	"github.com/google/uuid"
)

type IDGenerator[T comparable] interface {
	Next() T
	NextWithPrefix(prefix string) T
}

type StringIDGenerator struct {
	producer func() string
}

func (gen *StringIDGenerator) Next() string {
	return gen.producer()
}

func (gen *StringIDGenerator) NextWithPrefix(prefix string) string {
	return prefix + gen.Next()
}

type IntIDGenerator struct {
	mu   sync.Mutex
	curr uint64
}

func (gen *IntIDGenerator) Next() uint64 {
	gen.mu.Lock()
	defer gen.mu.Unlock()
	gen.curr++
	return gen.curr
}
func (gen *IntIDGenerator) NextWithPrefix(prefix string) uint64 {
	// IntIDGenerator does not support prefixing
	return gen.Next()
}

var (
	stringIDGen IDGenerator[string] = &StringIDGenerator{producer: uuid.NewString}
	intIDGen    IDGenerator[uint64] = &IntIDGenerator{}
)

func GenIntID() uint64 {
	return intIDGen.Next()
}

func GenID() string {
	return stringIDGen.Next()
}

func GenObjectID() string {
	return stringIDGen.NextWithPrefix("obj.")
}

func GenSessionID() string {
	return stringIDGen.NextWithPrefix("sessions.")
}
