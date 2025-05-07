package store

import (
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	ar "github.com/asynkron/protoactor-go/remote"

	"github.com/9triver/ignis/actor/remote/stub"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
)

func TestSingleStore(t *testing.T) {
	sys := actor.NewActorSystem()
	s := stub.NewActorStub(sys)
	props := New(s, "store")

	store := sys.Root.Spawn(props)
	defer sys.Root.Stop(store)

	ctx := sys.Root
	listen := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *messages.ObjectResponse:
			obj := msg.Value
			if stream, ok := obj.(*messages.LocalStream); ok {
				for i := range stream.ToChan() {
					t.Logf("received %v", i)
				}
			} else {
				t.Logf("received %v", obj)
			}
		}
	}))

	ctx.Send(store, &messages.SaveObject{
		Value: messages.NewLocalObjectWithID("obj-1", 10, proto.LangJson),
	})

	ctx.Send(store, &messages.RequestObject{
		ReplyTo: listen,
		Flow: &proto.Flow{
			ObjectID: "obj-1",
			Source: &proto.StoreRef{
				ID:  "store",
				PID: store,
			},
		},
	})

	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := range 10 {
			ints <- i
		}
	}()
	ctx.Send(store, &messages.SaveObject{
		Value:    messages.NewLocalStreamWithID("stream-1", ints, proto.LangJson),
		Callback: nil,
	})

	ctx.Send(store, &messages.RequestObject{
		ReplyTo: listen,
		Flow: &proto.Flow{
			ObjectID: "stream-1",
			Source: &proto.StoreRef{
				ID:  "store",
				PID: store,
			},
		},
	})

	<-time.After(10 * time.Second)
}

func TestMultipleStores(t *testing.T) {
	sys1 := actor.NewActorSystem()
	ctx1 := sys1.Root
	remote1 := ar.NewRemote(sys1, ar.Configure("127.0.0.1", 3000))
	remote1.Start()

	sys2 := actor.NewActorSystem()
	ctx2 := sys2.Root
	remote2 := ar.NewRemote(sys2, ar.Configure("127.0.0.1", 3001))
	remote2.Start()

	stub1 := stub.NewActorStub(sys1)
	stub2 := stub.NewActorStub(sys2)

	store1 := sys1.Root.Spawn(New(stub1, "store1"))
	defer sys1.Root.Stop(store1)

	store2 := sys2.Root.Spawn(New(stub2, "store2"))
	defer sys2.Root.Stop(store2)

	ctx1.Send(store1, &messages.SaveObject{
		Value: messages.NewLocalObjectWithID("obj-1", 10, proto.LangJson),
	})

	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := range 10 {
			ints <- i
		}
	}()
	ctx1.Send(store1, &messages.SaveObject{
		Value:    messages.NewLocalStreamWithID("stream-1", ints, proto.LangJson),
		Callback: nil,
	})

	listen := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *messages.ObjectResponse:
			obj := msg.Value
			if stream, ok := obj.(*messages.LocalStream); ok {
				for i := range stream.ToChan() {
					t.Logf("received chunk %v", i)
				}
			} else {
				t.Logf("received %v", obj)
			}
		}
	}))

	ctx2.Send(store2, &messages.RequestObject{
		ReplyTo: listen,
		Flow: &proto.Flow{
			ObjectID: "obj-1",
			Source: &proto.StoreRef{
				ID:  "store1",
				PID: store1,
			},
		},
	})

	ctx2.Send(store2, &messages.RequestObject{
		ReplyTo: listen,
		Flow: &proto.Flow{
			ObjectID: "stream-1",
			Source: &proto.StoreRef{
				ID:  "store1",
				PID: store1,
			},
		},
	})

	<-time.After(10 * time.Second)
}
