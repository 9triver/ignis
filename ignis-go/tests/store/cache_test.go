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
	"github.com/9triver/ignis/utils/cache"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
)

func TestLRUCache(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem()
	ctx1 := sys1.Root
	ctx2 := sys2.Root
	r1 := router.NewLocalRouter(ctx1)
	r2 := router.NewLocalRouter(ctx2)

	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(ctx1, r1, "store1")

	controller := cache.NewLRU[string, object.Interface](8)
	storeRef2 := store.Spawn(ctx2, r2, "store2", controller)

	r1.Register(storeRef2)
	r2.Register(storeRef1)

	refs := make([]*proto.Flow, 0, 10)

	var wg sync.WaitGroup
	wg.Add(10)
	producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			for i := range 10 {
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						t.Logf("[actor-1@sys1] saved %s: %d", ref.ID, i)
						refs = append(refs, ref)
						wg.Done()
					},
				})
			}
		}
	}))
	defer ctx1.Stop(producer)
	wg.Wait()

	fetchAll := func() {
		wg.Add(10)
		collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			switch msg := c.Message().(type) {
			case *actor.Started:
				for _, flow := range refs {
					c.Send(storeRef2.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow:    flow,
					})
				}
			case *store.ObjectResponse:
				obj := msg.Value
				t.Logf("[sys2] received object: %v", obj)
				wg.Done()
			}
		}))

		defer ctx2.Stop(collector)

		wg.Wait()
	}

	fetchAll()
	fetchAll()
}

func TestTimedCache(t *testing.T) {
	sys1 := actor.NewActorSystem(utils.WithLogger())
	sys2 := actor.NewActorSystem()
	ctx1 := sys1.Root
	ctx2 := sys2.Root
	r1 := router.NewLocalRouter(ctx1)
	r2 := router.NewLocalRouter(ctx2)

	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(ctx1, r1, "store1")

	controller := cache.NewTimed[string, object.Interface](5*time.Second, 300*time.Second)
	storeRef2 := store.Spawn(ctx2, r2, "store2", controller)

	r1.Register(storeRef2)
	r2.Register(storeRef1)

	refs := make([]*proto.Flow, 0, 10)

	var wg sync.WaitGroup
	wg.Add(10)
	producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			for i := range 10 {
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						t.Logf("[actor-1@sys1] saved %s: %d", ref.ID, i)
						refs = append(refs, ref)
						wg.Done()
					},
				})
			}
		}
	}))
	defer ctx1.Stop(producer)
	wg.Wait()

	fetchAll := func() {
		wg.Add(10)
		collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			switch msg := c.Message().(type) {
			case *actor.Started:
				for _, flow := range refs {
					c.Send(storeRef2.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow:    flow,
					})
				}
			case *store.ObjectResponse:
				obj := msg.Value
				t.Logf("[sys2] received object: %v", obj)
				wg.Done()
			}
		}))

		defer ctx2.Stop(collector)

		wg.Wait()
	}

	fetchAll()
	fetchAll()

	// wait for expiration
	time.Sleep(7 * time.Second)
	fetchAll()
}
