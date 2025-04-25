package store

import (
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    messages.Object                          // object or stream
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

// RequestObject is sent to store by **local** actors
type RequestObject struct {
	ReplyTo *actor.PID
	Flow    *proto.Flow
}

// ObjectResponse is sent to request actor
type ObjectResponse struct {
	Value messages.Object
	Error error
}
