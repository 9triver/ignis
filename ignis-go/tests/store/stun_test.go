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

func saveObject(t *testing.T) (refs []*proto.Flow) {
	sys := actor.NewActorSystem(utils.WithLogger())
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
						t.Logf("[actor-1@sys1] saved %s: %d", ref.ID, i)
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

func loadObject(t *testing.T, refs []*proto.Flow) {
	sys := actor.NewActorSystem(utils.WithLogger())
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
			t.Logf("[actor-2@sys2] receive object: %v", obj)
			wg.Done()
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func TestSTUNObject(t *testing.T) {
	refs := saveObject(t)
	loadObject(t, refs)
}

func saveStream(t *testing.T) (ref *proto.Flow) {
	sys := actor.NewActorSystem(utils.WithLogger())
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
			time.Sleep(500 * time.Millisecond)
			source <- i
			t.Logf("[actor-1@sys-1] send chunk: %d", i)
		}
	}()
	stream := object.StreamWithID(streamID, source, object.LangJson)

	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			c.Send(s.PID, &store.SaveObject{
				Value: stream,
				Callback: func(ctx actor.Context, ref_ *proto.Flow) {
					t.Logf("[actor-1@sys-1] saved %s", streamID)
					ref = ref_
					wg.Done()
				},
			})
		}
	}))
	defer ctx.Stop(producer)

	wg.Wait()
	return
}

func loadStream(t *testing.T, ref *proto.Flow) {
	sys := actor.NewActorSystem(utils.WithLogger())
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
			// 将头转换为channel，并从流对象中读取数据
			for chunk := range obj.(*object.Stream).ToChan() {
				t.Logf("[actor-2@sys-2] receive chunk: %v", chunk)
				wg.Done()
			}
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func TestSTUNStream(t *testing.T) {
	ref := saveStream(t)
	loadStream(t, ref)
}
