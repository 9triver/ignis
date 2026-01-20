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

func TestGlobalLoad(t *testing.T) {
	opts := utils.WithLogger()

	defer SaveResult("global_load")

	doTest := func(bytes, q int) {
		t.Logf("bytes: %d, concurrent: %d", bytes, q)

		sys1 := actor.NewActorSystem(opts)
		defer sys1.Shutdown()
		sys2 := actor.NewActorSystem(opts)
		defer sys2.Shutdown()

		ctx1 := sys1.Root
		r1 := router.NewTCPRouter(ctx1, nil, "127.0.0.1", 3000)
		defer r1.Shutdown()

		storeRef1 := store.Spawn(sys1.Root, r1, "store-1")
		defer ctx1.Stop(storeRef1.PID)
		pid := actor.NewPID("127.0.0.1:3000", "bootstrap")

		ctx2 := sys2.Root
		r2 := router.NewTCPRouter(ctx2, pid, "127.0.0.1", 3001)
		defer r2.Shutdown()

		storeRef2 := store.Spawn(sys2.Root, r2, "store-2")
		defer ctx2.Stop(storeRef2.PID)

		time.Sleep(100 * time.Millisecond)

		wg := sync.WaitGroup{}
		wg.Add(q)

		refs := make(chan *proto.Flow, q)

		producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			objs := make([]string, q)
			for i := range objs {
				objs[i] = generateObject(bytes)
			}

			for i := range q {
				obj := object.LocalWithID(fmt.Sprintf("obj-%d", i), objs[i], object.LangJson)
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: obj,
					Callback: func(ctx actor.Context, ref *proto.Flow) {
						refs <- ref
						wg.Done()
					},
				})
			}
		}))
		defer ctx1.Stop(producer)
		wg.Wait()
		close(refs)

		wg.Add(q)

		totalBytes := 0
		var tic time.Time
		collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			switch msg := c.Message().(type) {
			case *actor.Started:
				tic = time.Now()
				for ref := range refs {
					c.Send(storeRef2.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow:    ref,
					})
				}
			case *store.ObjectResponse:
				obj := msg.Value.(*object.Remote)
				totalBytes += len(obj.Data)
				wg.Done()
				// _, _ = msg.Value.Value()
			}
		}))
		defer ctx2.Stop(collector)

		wg.Wait()
		totalLatency := time.Since(tic)

		averageLatency := totalLatency / time.Duration(q)
		transmitted := float64(totalBytes) / 1024 / 1024
		speed := transmitted / totalLatency.Seconds()
		t.Logf("Average load latency: %dms", averageLatency.Milliseconds())
		t.Logf("Total load latency: %v", time.Since(tic))
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

	// set network rate limits
	// sudo tc qdisc add dev lo root tbf rate 1000mbit burst 128kb latency 10ms
	// sudo tc qdisc del dev lo root

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
