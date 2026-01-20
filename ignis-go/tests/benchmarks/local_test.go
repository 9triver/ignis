package benchmarks

import (
	"fmt"
	"runtime"
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

func TestLocalSave(t *testing.T) {
	opts := utils.WithLogger()

	defer SaveResult("local_save")

	doTest := func(bytes, q int) {
		sys := actor.NewActorSystem(opts)
		defer sys.Shutdown()

		ctx := sys.Root
		r := router.NewLocalRouter(ctx)
		t.Logf("bytes: %d, concurrent: %d", bytes, q)
		wg := sync.WaitGroup{}
		wg.Add(q)

		storeRef := store.Spawn(sys.Root, r, "store")
		defer ctx.Stop(storeRef.PID)

		var tic time.Time
		producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			objs := make([]string, min(10, q))
			for i := range objs {
				objs[i] = generateObject(bytes)
			}

			tic = time.Now()
			for i := range q {
				c.Send(storeRef.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), objs[i%10], object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						wg.Done()
					},
				})
			}
		}))
		defer ctx.Stop(producer)
		wg.Wait()

		totalLatency := time.Since(tic)
		averageLatency := totalLatency / time.Duration(q)
		transmitted := float64(bytes) * float64(q) / 1024 / 1024
		speed := transmitted / totalLatency.Seconds()
		t.Logf("Average save latency: %d ns", averageLatency.Nanoseconds())
		t.Logf("Transmitted: %.2f MB, Speed: %.2f MB/s", transmitted, speed)
		t.Log("========================================")

		WriteResult(
			"bytes", bytes,
			"q", q,
			"lat_ns", averageLatency.Nanoseconds(),
			"speed_mbps", speed,
			"transmitted_mb", transmitted,
		)
	}
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(16*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(64*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 64)
}

func TestLocalLoad(t *testing.T) {
	opts := utils.WithLogger()

	defer SaveResult("local_load")

	doTest := func(bytes, q int) {
		sys := actor.NewActorSystem(opts)
		defer sys.Shutdown()

		ctx := sys.Root
		r := router.NewLocalRouter(ctx)
		t.Logf("bytes: %d, concurrent: %d", bytes, q)
		wg := sync.WaitGroup{}
		wg.Add(q)

		storeRef := store.Spawn(sys.Root, r, "store")
		defer ctx.Stop(storeRef.PID)

		producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			objs := make([]string, min(10, q))
			for i := range objs {
				objs[i] = generateObject(bytes)
			}

			for i := range q {
				c.Send(storeRef.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), objs[i%10], object.LangJson),
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						wg.Done()
					},
				})
			}
		}))
		defer ctx.Stop(producer)
		wg.Wait()

		wg.Add(q)
		var totalLatency time.Duration
		collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			tic := time.Now()
			switch msg := c.Message().(type) {
			case *actor.Started:
				for i := range q {
					objectID := fmt.Sprintf("obj-%d", i)
					c.Send(storeRef.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow: &proto.Flow{
							ID:     objectID,
							Source: storeRef,
						},
					})
				}
			case *store.ObjectResponse:
				_, _ = msg.Value.Value()
				latency := time.Since(tic)
				totalLatency += latency
				wg.Done()
			}
		}))
		defer ctx.Stop(collector)

		wg.Wait()
		averageLatency := totalLatency / time.Duration(q)
		transmitted := float64(bytes) * float64(q) / 1024 / 1024
		speed := transmitted / totalLatency.Seconds()
		t.Logf("Average load latency: %d ns", averageLatency.Nanoseconds())
		t.Logf("Transmitted: %.2f MB, Speed: %.2f MB/s", transmitted, speed)
		t.Log("========================================")
		WriteResult(
			"bytes", bytes,
			"q", q,
			"lat_ns", averageLatency.Nanoseconds(),
			"speed_mbps", speed,
			"transmitted_mb", transmitted,
		)
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(1024*1024, 2)

	runtime.GC()

	doTest(1024*1024, 4)

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 8)

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 16)

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 32)

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(16*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(64*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 64)
}

func TestLocalLoadNoZeroCopy(t *testing.T) {
	opts := utils.WithLogger()

	defer SaveResult("local_load_nozerocopy")

	doTest := func(bytes, q int) {
		sys := actor.NewActorSystem(opts)
		defer sys.Shutdown()

		ctx := sys.Root
		r := router.NewLocalRouter(ctx)
		t.Logf("bytes: %d, concurrent: %d", bytes, q)
		wg := sync.WaitGroup{}
		wg.Add(q)

		storeRef := store.Spawn(sys.Root, r, "store")
		defer ctx.Stop(storeRef.PID)

		producer := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			for i := range q {
				obj := object.LocalWithID(fmt.Sprintf("obj-%d", i), generateObject(bytes), object.LangJson)
				c.Send(storeRef.PID, &store.SaveObject{
					Value: obj,
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						wg.Done()
					},
				})
			}
		}))
		defer ctx.Stop(producer)
		wg.Wait()

		wg.Add(q)
		var totalLatency time.Duration
		collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			tic := time.Now()

			switch msg := c.Message().(type) {
			case *actor.Started:
				for i := range q {
					objectID := fmt.Sprintf("obj-%d", i)
					c.Send(storeRef.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow: &proto.Flow{
							ID:     objectID,
							Source: storeRef,
						},
					})
				}
			case *store.ObjectResponse:
				v, _ := msg.Value.Value()
				tmp := make([]byte, bytes)
				copy(tmp, []byte(v.(string)))

				latency := time.Since(tic)
				totalLatency += latency
				wg.Done()
			}
		}))
		defer ctx.Stop(collector)

		wg.Wait()
		averageLatency := totalLatency / time.Duration(q)
		transmitted := float64(bytes) * float64(q) / 1024 / 1024
		speed := transmitted / totalLatency.Seconds()
		t.Logf("Average load latency: %d ns", averageLatency.Nanoseconds())
		t.Logf("Transmitted: %.2f MB, Speed: %.2f MB/s", transmitted, speed)
		t.Log("========================================")

		WriteResult(
			"bytes", bytes,
			"q", q,
			"lat_ns", averageLatency.Nanoseconds(),
			"speed_mbps", speed,
			"transmitted_mb", transmitted,
		)
	}

	doTest(1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(16*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(16*1024*1024, 64)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	t.Log()

	doTest(64*1024*1024, 2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 4)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 8)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 16)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 32)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	doTest(64*1024*1024, 64)
}
