package store

import (
	"github.com/asynkron/protoactor-go/actor"
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
)

type Stub interface {
	SendTo(remoteRef *proto.StoreRef, msg pb.Message)
	RecvChan() <-chan pb.Message
	Close() error
}

type stubMessage struct {
	Target *proto.StoreRef
	Msg    pb.Message
}

type ActorStub struct {
	send chan *stubMessage
	recv chan pb.Message
}

func (s *ActorStub) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *stubMessage:
		ctx.Send(msg.Target.PID, msg.Msg)
	case pb.Message:
		s.recv <- msg
	}
}

func (s *ActorStub) SendTo(remoteRef *proto.StoreRef, msg pb.Message) {
	s.send <- &stubMessage{
		Target: remoteRef,
		Msg:    msg,
	}
}

func (s *ActorStub) RecvChan() <-chan pb.Message {
	return s.recv
}

func (s *ActorStub) Close() error {
	close(s.send)
	return nil
}

func NewActorStub(sys *actor.ActorSystem) Stub {
	send := make(chan *stubMessage, configs.ChannelBufferSize)

	s := &ActorStub{
		send: send,
		recv: make(chan pb.Message, configs.ChannelBufferSize),
	}
	pid := sys.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return s
	}))

	go func() {
		for msg := range send {
			sys.Root.Send(pid, msg)
		}
	}()
	return s
}
