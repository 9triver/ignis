package store

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

func TestZeroCopy(t *testing.T) {
	// 初始化Actor运行时
	sys := actor.NewActorSystem()
	ctx := sys.Root
	r := router.NewActorRouter(ctx)
	// 预先创建存储/传输的对象
	var objects []object.Interface
	// 存储对象的内存地址，用于验证零拷贝
	var pointers []unsafe.Pointer

	for i := range 10 {
		obj := object.LocalWithID(fmt.Sprintf("obj-%d", i), i, object.LangJson)
		objects = append(objects, obj)
		pointers = append(pointers, unsafe.Pointer(obj))
	}

	storeRef := store.Spawn(sys.Root, r, "store")

	// 通过本地的Actor运行组件，向Store写入对象：obj-0, obj-1, ..., obj-9
	wg := sync.WaitGroup{}
	wg.Add(10)
	producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for _, obj := range objects {
			c.Send(storeRef.PID, &store.SaveObject{
				Value: obj,
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
		switch c.Message().(type) {
		// 当收集器启动时，发送10个请求
		case *actor.Started:
			for i, pointer := range pointers {
				obj, _ := store.GetObject(c, storeRef.PID, &proto.Flow{
					ID:     fmt.Sprintf("obj-%d", i),
					Source: storeRef,
				}).Result()

				// 获取对象的地址，并和原对象的地址进行比较
				pt := unsafe.Pointer(obj.(*object.Local))
				t.Logf("get: %x, original: %x", pt, pointer)
				if pt != pointer {
					t.Fatal("zero copy failed!")
				}
				wg.Done()
			}
		}
	}))
	defer ctx.Stop(collector)

	// 收集器会收到全部10个对象响应: obj-0, obj-1, ..., obj-9
	wg.Wait()
}
