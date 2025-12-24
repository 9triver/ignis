package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
)

func TestGlobal(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem(utils.WithLogger())
	ctx1 := sys1.Root
	ctx2 := sys2.Root

	r1 := router.NewLocalRouter(ctx1)
	r2 := router.NewLocalRouter(ctx2)

	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(ctx1, r1, "store1")
	storeRef2 := store.Spawn(ctx2, r2, "store2")

	r1.Register(storeRef2)
	r2.Register(storeRef1)

	refs := make(chan *proto.Flow, 10)
	defer close(refs)

	producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			for i := range 10 {
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						t.Logf("[actor-1@sys1] saved %s: %d", ref.ID, i)
						refs <- ref
					},
				})
			}
		}
	}))
	defer ctx1.Stop(producer)

	var wg sync.WaitGroup
	wg.Add(10)

	collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for ref := range refs {
			store.GetObject(c, storeRef2.PID, ref).
				OnDone(func(obj object.Interface, duration time.Duration, err error) {
					t.Logf("[actor-2@sys2] discover object: %s (isStream = %v)", obj.GetID(), obj.IsStream())
					v, _ := obj.Value()
					t.Logf("[actor-2@sys2] receive object: %v", v)
					wg.Done()
				})
		}
	}))
	defer ctx2.Stop(collector)

	wg.Wait()
}

func TestGlobalStream(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem(utils.WithLogger())
	ctx1 := sys1.Root
	ctx2 := sys2.Root

	r1 := router.NewLocalRouter(ctx1)
	r2 := router.NewLocalRouter(ctx2)

	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(ctx1, r1, "store1")
	storeRef2 := store.Spawn(ctx2, r2, "store2")

	r1.Register(storeRef2)
	r2.Register(storeRef1)

	refs := make(chan *proto.Flow, 1)
	defer close(refs)

	streamID := "stream-0"
	source := make(chan int)

	stream := object.StreamWithID(streamID, source, object.LangJson)
	go func() {
		defer close(source)
		for i := range 10 {
			time.Sleep(500 * time.Millisecond)
			source <- i
			t.Logf("[actor-1@sys-1] send chunk: %d", i)
		}
	}()

	producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		c.Send(storeRef1.PID, &store.SaveObject{
			Value: stream,
			Callback: func(ctx actor.Context, ref *proto.Flow) {
				t.Logf("[actor-1@sys-1] saved %s", streamID)
				refs <- ref
			},
		})
	}))
	defer ctx1.Stop(producer)

	var wg sync.WaitGroup
	wg.Add(10)

	collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for ref := range refs {
			store.GetObject(c, storeRef2.PID, ref).
				OnDone(func(obj object.Interface, duration time.Duration, err error) {
					t.Logf("[actor-2@sys-2] discover object: %s (isStream = %t)", obj.GetID(), obj.IsStream())
					stream, ok := obj.(*object.Stream)
					if !ok {
						t.Fatal("Not a stream")
					}

					for chunk := range stream.ToChan() {
						t.Logf("[actor-2@sys-2] receive chunk: %v", chunk)
						wg.Done()
					}
				})
		}
	}))

	defer ctx2.Stop(collector)

	wg.Wait()
}
