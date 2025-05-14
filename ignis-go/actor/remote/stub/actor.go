package stub

import (
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/cluster"
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"
)

const stubActorName = "__stub"

func stubFromStore(ref *proto.StoreRef) *actor.PID {
	return actor.NewPID(ref.PID.Address, stubActorName)
}

type ActorStub struct {
	ctx  actor.Context
	recv chan *cluster.Envelope
}

func (s *ActorStub) Receive(ctx actor.Context) {
	switch e := ctx.Message().(type) {
	case *cluster.Envelope: // receive envelope from remote stores
		select {
		case s.recv <- e:
		default:
			ctx.Logger().Warn("stub: dead message", "msg", e)
		}
	}
}

func (s *ActorStub) SendTo(remoteRef *proto.StoreRef, msg pb.Message) {
	envelope := cluster.NewEnvelope(remoteRef, msg)
	s.ctx.Send(stubFromStore(remoteRef), envelope)
}

func (s *ActorStub) RecvChan() <-chan *cluster.Envelope {
	return s.recv
}

func (s *ActorStub) Close() error {
	close(s.recv)
	s.ctx.Stop(s.ctx.Self())
	return nil
}

func NewActorStub(ctx *actor.RootContext) remote.Stub {
	recv := make(chan *cluster.Envelope, configs.ChannelBufferSize)
	stub := &ActorStub{
		recv: recv,
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return stub
	}, actor.WithOnInit(func(ctx actor.Context) {
		stub.ctx = ctx
	}))

	ctx.SpawnNamed(props, stubActorName)

	return stub
}
