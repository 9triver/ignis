package utils

import (
	"strconv"
	"sync"
)

type IDGenerator struct {
	key uint64
	mu  sync.Mutex
}

func (gen *IDGenerator) Next() uint64 {
	gen.mu.Lock()
	defer gen.mu.Unlock()

	gen.key++
	return gen.key
}

func (gen *IDGenerator) NextWithPrefix(prefix string) string {
	return prefix + strconv.FormatUint(gen.Next(), 16)
}

var objIDGen = IDGenerator{}
var globalIdGen = IDGenerator{}

func GenNextID() uint64 {
	return globalIdGen.Next()
}

func GenObjectID() string {
	return objIDGen.NextWithPrefix("obj")
}

func GenExecutionID() string {
	return globalIdGen.NextWithPrefix("exec")
}
