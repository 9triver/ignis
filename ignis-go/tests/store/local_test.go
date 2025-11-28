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

func TestLocal(t *testing.T) {
	// 初始化Actor运行时
	sys := actor.NewActorSystem()
	ctx := sys.Root
	r := router.NewLocalRouter(ctx)
	storeRef := store.Spawn(sys.Root, r, "store")

	// 通过本地的Actor运行组件，向Store写入对象：obj-0, obj-1, ..., obj-9
	wg := sync.WaitGroup{}
	wg.Add(10)
	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for i := range 10 {
			c.Send(storeRef.PID, &store.SaveObject{
				Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					wg.Done()
				},
			})
		}
	}))
	defer ctx.Stop(producer)

	// 等待所有对象写入完成
	wg.Wait()

	// 创建一个收集器，收集所有对象响应
	wg.Add(10)
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		// 当收集器启动时，发送10个请求
		// 在请求时，只需要提供对象ID即可
		case *actor.Started:
			for i := range 10 {
				c.Send(storeRef.PID, &store.RequestObject{
					ReplyTo: c.Self(),
					Flow: &proto.Flow{
						ID:     fmt.Sprintf("obj-%d", i),
						Source: storeRef,
					},
				})
			}
		// 等待所有对象返回: obj-0, obj-1, ..., obj-9
		case *store.ObjectResponse:
			obj := msg.Value
			t.Logf("received %v", obj)
			wg.Done()
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func TestLocalStream(t *testing.T) {
	// 初始化Actor运行时
	sys := actor.NewActorSystem()
	ctx := sys.Root
	r := router.NewLocalRouter(ctx)
	storeRef := store.Spawn(ctx, r, "store-stream")

	// 创建一个流对象，并将其保存到Store
	streamID := "stream-0"
	source := make(chan int)
	// 创建一个生产者，向流对象中异步写入数据
	go func() {
		defer close(source)
		for i := range 10 {
			source <- i
			time.Sleep(500 * time.Millisecond)
		}
	}()
	stream := object.StreamWithID(streamID, source, object.LangJson)

	// 创建一个生产者，将流对象保存到Store
	wg := sync.WaitGroup{}
	wg.Add(1)
	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		c.Send(storeRef.PID, &store.SaveObject{
			Value: stream,
			Callback: func(ctx actor.Context, ref *proto.Flow) {
				wg.Done()
			},
		})
	}))
	defer ctx.Stop(producer)

	// 等待流对象保存完成
	wg.Wait()

	// 创建一个收集器，从Store请求流对象
	wg.Add(10)
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		// 当收集器启动时，发送请求
		// 在请求时，只需要提供对象ID即可
		case *actor.Started:
			c.Send(storeRef.PID, &store.RequestObject{
				ReplyTo: c.Self(),
				Flow: &proto.Flow{
					ID:     streamID,
					Source: storeRef,
				},
			})
		case *store.ObjectResponse:
			// 获取流对象头
			obj := msg.Value
			t.Logf("received stream header: %v", obj)

			// 将头转换为channel，并从流对象中读取数据
			for val := range obj.(*object.Stream).ToChan() {
				t.Logf("received stream chunk: %v", val)
				wg.Done()
			}
		}
	}))

	defer ctx.Stop(collector)

	// 等待流完全返回
	wg.Wait()
}
