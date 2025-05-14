package remote

import (
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/cluster"
)

type Stub interface {
	SendTo(remoteRef *proto.StoreRef, msg pb.Message)
	RecvChan() <-chan *cluster.Envelope
	Close() error
}

type StubSpawnFunc func(ctx *actor.RootContext) Stub
