package store

import (
	"github.com/9triver/ignis/objects"
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type forwardMessage interface {
	pb.Message
	GetTarget() *proto.ActorRef
}

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    objects.Interface                        // object or stream
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

// RequestObject is sent to store by **local** actors
type RequestObject struct {
	Flow     *proto.Flow
	Callback func(ctx actor.Context, obj objects.Interface, err error)
}

// ObjectResponse is sent to request actor
type ObjectResponse struct {
	Value objects.Interface
	Error error
}
