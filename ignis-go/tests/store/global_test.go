package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/transport/stub"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
)

func TestGlobal(t *testing.T) {
	sys1 := actor.NewActorSystem()
	sys2 := actor.NewActorSystem()

	// 在:3000和:3001端口启动两个远程Actor系统，模拟多机部署
	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(sys1.Root, stub.NewActorStub(sys1), "store1")
	storeRef2 := store.Spawn(sys2.Root, stub.NewActorStub(sys2), "store2")

	ctx1 := sys1.Root
	ctx2 := sys2.Root

	wg := sync.WaitGroup{}
	wg.Add(10)
	refs := make(chan *proto.Flow, 10)
	// 在sys1中创建10个对象，并保存到store1
	// 为简化测试，这里使用本地对象传输object引用，实际应用中应该使用远程对象
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
			t.Logf("[sys2] received %v", obj)
			wg.Done()
		}
	}))

	wg.Wait()
}

func TestGlobalStream(t *testing.T) {
	sys1 := actor.NewActorSystem()
	sys2 := actor.NewActorSystem()

	remoter1 := remote.NewRemote(sys1, remote.Configure("127.0.0.1", 3000))
	remoter2 := remote.NewRemote(sys2, remote.Configure("127.0.0.1", 3001))
	remoter1.Start()
	remoter2.Start()

	storeRef1 := store.Spawn(sys1.Root, stub.NewActorStub(sys1), "store1")
	storeRef2 := store.Spawn(sys2.Root, stub.NewActorStub(sys2), "store2")

	ctx1 := sys1.Root
	ctx2 := sys2.Root

	wg := sync.WaitGroup{}
	wg.Add(1)
	refs := make(chan *proto.Flow, 10)
	// 在sys1中创建10个对象，并保存到store1
	// 为简化测试，这里使用本地对象传输object引用，实际应用中应该使用远程对象
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
