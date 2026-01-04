package benchmarks

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
			tic = time.Now()
			for i := range q {
				obj := generateObject(bytes)

				c.Send(storeRef.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), obj, object.LangJson),
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
		t.Logf("Average save latency: %d us", averageLatency.Microseconds())
		t.Logf("Transmitted: %.2f MB, Speed: %.2f MB/s", transmitted, speed)
		t.Log("========================================")

		WriteResult(
			"bytes", bytes,
			"q", q,
			"lat_us", averageLatency.Microseconds(),
			"speed_mbps", speed,
			"transmitted_mb", transmitted,
		)
	}

	doTest(1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(16*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(64*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 128)
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
			for i := range q {
				obj := generateObject(bytes)
				c.Send(storeRef.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), obj, object.LangJson),
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

	doTest(1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(16*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(64*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 128)
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
				// encoding object disables zero-copy
				enc, _ := obj.Encode()
				c.Send(storeRef.PID, &store.SaveObject{
					Value: enc,
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
			"lat_us", averageLatency.Microseconds(),
			"speed_mbps", speed,
			"transmitted_mb", transmitted,
		)
	}

	doTest(1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(16*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 128)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(64*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 128)
}
