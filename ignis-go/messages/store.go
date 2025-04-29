package messages

import (
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type ForwardMessage interface {
	pb.Message
	GetTarget() *proto.ActorRef
}

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    Object                                   // object or stream
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

// RequestObject is sent to store by **local** actors
type RequestObject struct {
	ReplyTo *actor.PID
	Flow    *proto.Flow
}

// ObjectResponse is sent to request actor
type ObjectResponse struct {
	Value Object
	Error error
}
