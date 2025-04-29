package remote

import (
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/cluster"
)

type Stub interface {
	SendTo(remoteRef *proto.StoreRef, msg pb.Message)
	RecvChan() <-chan pb.Message
	Close() error
}

type SendEnvelope struct {
	*cluster.Envelope
}

type RecvEnvelope = *cluster.Envelope

type ActorStub struct {
	pid  *actor.PID
	send chan SendEnvelope
	recv chan pb.Message
}

func (s *ActorStub) SendTo(remoteRef *proto.StoreRef, msg pb.Message) {
	envelope := cluster.NewEnvelope(remoteRef, msg)
	s.send <- SendEnvelope{envelope}
}

func (s *ActorStub) RecvChan() <-chan pb.Message {
	return s.recv
}

func (s *ActorStub) Close() error {
	close(s.send)
	return nil
}

func NewActorStub(sys *actor.ActorSystem) Stub {
	send := make(chan SendEnvelope, configs.ChannelBufferSize)
	recv := make(chan pb.Message, configs.ChannelBufferSize)

	props := actor.PropsFromFunc(func(ctx actor.Context) {
		switch e := ctx.Message().(type) {
		case SendEnvelope:
			ctx.Send(e.Store.PID, e.Unwrap())
		case RecvEnvelope:
			recv <- e.Unwrap()
		}
	})
	pid := sys.Root.Spawn(props)
	go func() {
		for msg := range send {
			sys.Root.Send(pid, msg)
		}
	}()
	return &ActorStub{
		send: send,
		recv: recv,
		pid:  pid,
	}
}
