package global_test

import (
	"fmt"
	"strings"
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

func generatorObject(n int) string {
	return strings.Repeat("b", n)
}

func TestGlobalLoad(t *testing.T) {
	opts := utils.WithLogger()

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
			for i := range q {
				obj := generatorObject(bytes)
				c.Send(storeRef1.PID, &store.SaveObject{
					Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), obj, object.LangJson),
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

		var totalLatency time.Duration
		wg.Add(q)

		var tic time.Time
		collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
			tic = time.Now()
			switch msg := c.Message().(type) {
			case *actor.Started:
				for ref := range refs {
					c.Send(storeRef2.PID, &store.RequestObject{
						ReplyTo: c.Self(),
						Flow:    ref,
					})
				}
			case *store.ObjectResponse:
				latency := time.Since(tic)
				_, err := msg.Value.Value()
				if err != nil {
					t.Fatal(err)
				}
				totalLatency += latency
				wg.Done()
			}
		}))
		defer ctx2.Stop(collector)

		wg.Wait()
		averageLatency := totalLatency / time.Duration(q)
		transmitted := float64(bytes) * float64(q) / 1024 / 1024
		speed := transmitted / totalLatency.Seconds()
		t.Logf("Average load latency: %d ns", averageLatency.Nanoseconds())
		t.Logf("Total load latency: %v s", time.Since(tic))
		t.Logf("Transmitted: %.2f MB, Speed: %.2f MB/s", transmitted, speed)
		t.Log("========================================")
	}

	doTest(1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024, 128)
	time.Sleep(5 * time.Second)
	doTest(1024, 256)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 128)
	time.Sleep(5 * time.Second)
	doTest(1024*1024, 256)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(16*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 128)
	time.Sleep(5 * time.Second)
	doTest(16*1024*1024, 256)

	time.Sleep(10 * time.Second)
	t.Log()

	doTest(64*1024*1024, 2)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 8)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 32)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 128)
	time.Sleep(5 * time.Second)
	doTest(64*1024*1024, 256)
}
