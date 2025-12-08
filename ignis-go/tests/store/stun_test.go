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
	"github.com/asynkron/protoactor-go/actor"
)

func saveObject() (refs []*proto.Flow) {
	sys := actor.NewActorSystem()
	ctx := sys.Root

	r := router.NewSTUNRouter(ctx, "s1", "ws://8.153.200.135:28080/ws")
	s := store.Spawn(ctx, r, "s1")

	var wg sync.WaitGroup
	wg.Add(10)

	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			for i := range 10 {
				c.Send(s.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						refs = append(refs, ref)
						wg.Done()
					},
				})
			}
		}
	}))
	defer ctx.Stop(producer)

	wg.Wait()
	return
}

func saveStream() (ref *proto.Flow) {
	sys := actor.NewActorSystem()
	ctx := sys.Root

	r := router.NewSTUNRouter(ctx, "s1", "ws://8.153.200.135:28080/ws")
	s := store.Spawn(ctx, r, "s1")

	var wg sync.WaitGroup
	wg.Add(1)

	streamID := "stream-0"
	source := make(chan int, 10)
	// 创建一个生产者，向流对象中异步写入数据
	go func() {
		defer close(source)
		for i := range 10 {
			source <- i
			time.Sleep(500 * time.Millisecond)
		}
	}()
	stream := object.StreamWithID(streamID, source, object.LangJson)

	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			c.Send(s.PID, &store.SaveObject{
				Value: stream,
				Callback: func(ctx actor.Context, ref_ *proto.Flow) {
					wg.Done()
					ref = ref_
				},
			})
		}
	}))
	defer ctx.Stop(producer)

	wg.Wait()
	return
}

func loadObject(refs []*proto.Flow) {
	sys := actor.NewActorSystem()
	ctx := sys.Root

	r := router.NewSTUNRouter(ctx, "s2", "ws://8.153.200.135:28080/ws")
	s := store.Spawn(sys.Root, r, "s2")

	var wg sync.WaitGroup
	wg.Add(10)
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		// 当收集器启动时，发送10个请求
		// 在请求时，只需要提供对象ID即可
		case *actor.Started:
			for _, ref := range refs {
				c.Send(s.PID, &store.RequestObject{
					ReplyTo: c.Self(),
					Flow:    ref,
				})
				time.Sleep(100 * time.Millisecond)
			}
		// 等待所有对象返回: obj-0, obj-1, ..., obj-9
		case *store.ObjectResponse:
			obj := msg.Value
			ctx.Logger().Info("receive obj", "obj", obj)
			wg.Done()
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func loadStream(ref *proto.Flow) {
	sys := actor.NewActorSystem()
	ctx := sys.Root

	r := router.NewSTUNRouter(ctx, "s2", "ws://8.153.200.135:28080/ws")
	s := store.Spawn(sys.Root, r, "s2")

	var wg sync.WaitGroup
	wg.Add(10)
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		// 当收集器启动时，发送请求
		// 在请求时，只需要提供对象ID即可
		case *actor.Started:
			c.Send(s.PID, &store.RequestObject{
				ReplyTo: c.Self(),
				Flow:    ref,
			})
		case *store.ObjectResponse:
			// 获取流对象头
			obj := msg.Value
			ctx.Logger().Info("received stream header", "stream", obj)

			// 将头转换为channel，并从流对象中读取数据
			for val := range obj.(*object.Stream).ToChan() {
				ctx.Logger().Info("received stream chunk", "chunk", val)
				wg.Done()
			}
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func TestSTUNObject(t *testing.T) {
	refs := saveObject()
	loadObject(refs)
}

func TestSTUNStream(t *testing.T) {
	ref := saveStream()
	loadStream(ref)
}
