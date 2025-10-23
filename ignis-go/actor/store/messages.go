package store

import (
	"github.com/9triver/ignis/objects"
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type RequiresReplyMessage interface {
	pb.Message
	GetReplyTo() string
}

type ForwardMessage interface {
	pb.Message
	GetTarget() string
}

type AddStore struct {
	Ref *proto.StoreRef
}

type RemoveStore struct {
	ID string
}

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    objects.Interface                        // object or stream
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

// RequestObject is sent to store by **local** actors
type RequestObject struct {
	ReplyTo *actor.PID
	Flow    *proto.Flow
}

// ObjectResponse is sent to request actor
type ObjectResponse struct {
	Value objects.Interface
	Error error
}
