package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	ar "github.com/asynkron/protoactor-go/remote"

	"github.com/9triver/ignis/actor/remote/stub"
	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/proto"
)

func callback(ctx actor.Context, obj objects.Interface, err error) {
	go func() {
		if err != nil {
			fmt.Printf("failed to fetch %v: %v\n", "stream-1", err)
			return
		}
		if stream, ok := obj.(*objects.Stream); ok {
			for i := range stream.ToChan() {
				fmt.Printf("received chunk %v\n", i)
			}
		} else {
			fmt.Printf("received %v\n", obj)
		}
	}()
}

func TestSingleStore(t *testing.T) {
	sys := actor.NewActorSystem()
	store := Spawn(sys.Root, stub.NewActorStub, "store")

	ctx := sys.Root

	ctx.Send(store.PID, &SaveObject{
		Value: objects.LocalWithID("obj-1", 10, objects.LangJson),
	})

	ctx.Send(store.PID, &RequestObject{
		Flow: &proto.Flow{
			ID:     "obj-1",
			Source: store,
		},
		Callback: callback,
	})

	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := range 10 {
			ints <- i
		}
	}()
	ctx.Send(store.PID, &SaveObject{
		Value: objects.StreamWithID("stream-1", ints, objects.LangJson),
	})

	ctx.Send(store.PID, &RequestObject{
		Flow: &proto.Flow{
			ID:     "stream-1",
			Source: store,
		},
		Callback: callback,
	})

	<-time.After(120 * time.Second)
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

	store1 := Spawn(ctx1, stub.NewActorStub, "store1")
	store2 := Spawn(ctx2, stub.NewActorStub, "store2")

	defer ctx1.Stop(store1.PID)
	defer ctx2.Stop(store2.PID)

	ctx1.Send(store1.PID, &SaveObject{
		Value: objects.LocalWithID("obj-1", 10, objects.LangJson),
	})

	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := range 10 {
			ints <- i
		}
	}()
	ctx1.Send(store1.PID, &SaveObject{
		Value: objects.StreamWithID("stream-1", ints, objects.LangJson),
	})

	ctx2.Send(store2.PID, &RequestObject{
		Flow: &proto.Flow{
			ID:     "obj-1",
			Source: store1,
		},
		Callback: callback,
	})

	ctx2.Send(store2.PID, &RequestObject{
		Flow: &proto.Flow{
			ID:     "stream-1",
			Source: store1,
		},
		Callback: callback,
	})

	<-time.After(120 * time.Second)
}
