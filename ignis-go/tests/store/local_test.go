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

func TestLocal(t *testing.T) {
	// 初始化Actor运行时
	sys := actor.NewActorSystem(utils.WithLogger())
	ctx := sys.Root
	r := router.NewLocalRouter(ctx)
	storeRef := store.Spawn(sys.Root, r, "store")

	// 通过本地的Actor运行组件，向Store写入对象：obj-0, obj-1, ..., obj-9
	refs := make(chan *proto.Flow, 10)
	defer close(refs)

	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for i := range 10 {
			c.Send(storeRef.PID, &store.SaveObject{
				Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson),
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					t.Logf("[actor-1] saved %s: %d", ref.ID, i)
					refs <- ref
				},
			})
		}
	}))
	defer ctx.Stop(producer)

	var wg sync.WaitGroup
	// 创建一个收集器，收集所有对象响应
	wg.Add(10)
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for ref := range refs {
			store.GetObject(c, storeRef.PID, ref).
				OnDone(func(obj object.Interface, duration time.Duration, err error) {
					t.Logf("[actor-2] discovered object: %s (isStream = %v)", obj.GetID(), obj.IsStream())
					v, _ := obj.Value()
					t.Logf("[actor-2] receive object: %v", v)
					wg.Done()
				})
		}
	}))

	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}

func TestLocalStream(t *testing.T) {
	// 初始化Actor运行时
	sys := actor.NewActorSystem(utils.WithLogger())
	ctx := sys.Root
	r := router.NewLocalRouter(ctx)
	storeRef := store.Spawn(ctx, r, "store-stream")

	refs := make(chan *proto.Flow, 1)
	defer close(refs)

	// 创建一个流对象，并将其保存到Store
	streamID := "stream-0"
	source := make(chan int)
	// 创建一个生产者，向流对象中异步写入数据
	go func() {
		defer close(source)
		for i := range 10 {
			time.Sleep(500 * time.Millisecond)
			source <- i
			t.Logf("[actor-1] send chunk: %d", i)
		}
	}()
	stream := object.StreamWithID(streamID, source, object.LangJson)

	// 创建一个生产者，将流对象保存到Store

	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		c.Send(storeRef.PID, &store.SaveObject{
			Value: stream,
			Callback: func(ctx actor.Context, ref *proto.Flow) {
				t.Logf("[actor-1] saved %s", ref.ID)
				refs <- ref
			},
		})
	}))
	defer ctx.Stop(producer)

	var wg sync.WaitGroup
	wg.Add(10)

	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for ref := range refs {
			store.GetObject(c, storeRef.PID, ref).
				OnDone(func(obj object.Interface, duration time.Duration, err error) {
					t.Logf("[actor-2] discover object: %s (isStream = %t)", obj.GetID(), obj.IsStream())
					stream, ok := obj.(*object.Stream)
					if !ok {
						t.Fatal("Not a stream")
					}

					for chunk := range stream.ToChan() {
						t.Logf("[actor-2] receive chunk: %v", chunk)
						wg.Done()
					}
				})
		}
	}))

	defer ctx.Stop(collector)

	wg.Wait()
}
