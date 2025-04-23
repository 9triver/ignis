package messages

import (
	"github.com/asynkron/protoactor-go/actor"
)

type Successor struct {
	ID    string
	Param string
	PID   *actor.PID
}

type CreateSession struct {
	SessionID  string
	Successors []*Successor
}
