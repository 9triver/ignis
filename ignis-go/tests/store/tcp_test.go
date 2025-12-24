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
)

func TestTCPObject(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem(utils.WithLogger())
	ctx1 := sys1.Root
	ctx2 := sys2.Root

	storeRef1 := store.Spawn(ctx1, router.NewTCPRouter(ctx1, nil, "127.0.0.1", 3000), "store-1")
	pid := actor.NewPID("127.0.0.1:3000", "bootstrap")
	storeRef2 := store.Spawn(ctx2, router.NewTCPRouter(ctx2, pid, "127.0.0.1", 3001), "store-2")

	time.Sleep(100 * time.Millisecond)

	wg := sync.WaitGroup{}
	wg.Add(10)
	refs := make(chan *proto.Flow, 10)

	ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			for i := range 10 {
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						t.Logf("[sys1] saved object %d", i)
						refs <- ref
						wg.Done()
					},
				})
			}
		}
	}))
	wg.Wait()
	close(refs)

	wg.Add(10)
	// 在sys2中请求10个对象，这些对象是sys1中保存的
	ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *actor.Started:
			for flow := range refs {
				c.Send(storeRef2.PID, &store.RequestObject{
					ReplyTo: c.Self(),
					Flow:    flow,
				})
			}
		case *store.ObjectResponse:
			obj := msg.Value
			t.Logf("[sys2] receive %v", obj)
			wg.Done()
		}
	}))

	wg.Wait()
}

func TestTCPStream(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem(utils.WithLogger())
	ctx1 := sys1.Root
	ctx2 := sys2.Root

	storeRef1 := store.Spawn(ctx1, router.NewTCPRouter(ctx1, nil, "127.0.0.1", 3000), "store-1")
	pid := actor.NewPID("127.0.0.1:3000", "bootstrap")
	storeRef2 := store.Spawn(ctx2, router.NewTCPRouter(ctx2, pid, "127.0.0.1", 3001), "store-2")

	time.Sleep(100 * time.Millisecond)

	wg := sync.WaitGroup{}
	wg.Add(1)
	refs := make(chan *proto.Flow, 10)

	streamID := "stream-0"
	source := make(chan int)
	go func() {
		defer close(source)
		for i := range 10 {
			source <- i
			time.Sleep(500 * time.Millisecond)
		}
	}()
	stream := object.StreamWithID(streamID, source, object.LangJson)

	ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			c.Send(storeRef1.PID, &store.SaveObject{
				Value: stream,
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					refs <- ref
					wg.Done()
				},
			})
		}
	}))

	wg.Wait()
	close(refs)

	wg.Add(10)
	ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *actor.Started:
			for flow := range refs {
				c.Send(storeRef2.PID, &store.RequestObject{
					ReplyTo: c.Self(),
					Flow:    flow,
				})
			}
		case *store.ObjectResponse:
			obj := msg.Value
			t.Logf("[sys2] received stream header: %v", obj)
			for v := range obj.(*object.Stream).ToChan() {
				t.Logf("[sys2] received stream chunk: %v", v)
				wg.Done()
			}
		}
	}))

	wg.Wait()
}
