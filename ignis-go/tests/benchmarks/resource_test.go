package benchmarks

import (
	"runtime"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/utils"
	"github.com/asynkron/protoactor-go/actor"
)

func TestLocalResource(t *testing.T) {
	defer SaveResult("local_resource")

	doTest := func(nActors int) {
		var m1, m2, m3 runtime.MemStats
		runtime.ReadMemStats(&m1)

		opts := utils.WithLogger()
		sys := actor.NewActorSystem(opts)
		defer sys.Shutdown()

		ctx := sys.Root
		r := router.NewLocalRouter(ctx)
		defer r.Shutdown()

		storeRef := store.Spawn(sys.Root, r, "store")
		defer ctx.Stop(storeRef.PID)

		time.Sleep(300 * time.Millisecond)
		runtime.ReadMemStats(&m2)

		startup := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
		t.Logf("store startup: allocated %.3f MB", startup)

		props := compute.NewActor(
			"test",
			functions.NewGo("test-fn", func(input struct{ A, B int }) (int, error) {
				return input.A + input.B, nil
			}, object.LangJson),
			storeRef.PID,
		)

		for range nActors {
			pid := ctx.Spawn(props)
			defer ctx.Stop(pid)
		}

		time.Sleep(300 * time.Millisecond)
		runtime.ReadMemStats(&m3)

		actors := float64(m3.Alloc-m2.Alloc) / 1024 / 1024
		total := float64(m3.Alloc-m1.Alloc) / 1024 / 1024
		t.Logf("%d actors: allocated %.3f MB", nActors, actors)
		t.Logf("total: %.3f MB", total)
		WriteResult(
			"nActors", nActors,
			"startup_mb", startup,
			"actors_mb", actors,
			"total_mb", total,
		)
	}

	doTest(10)
	time.Sleep(3 * time.Second)
	runtime.GC()

	doTest(100)
	time.Sleep(3 * time.Second)
	runtime.GC()

	doTest(1000)
}

func TestClusterResource(t *testing.T) {
	defer SaveResult("cluster_resource")

	doTest := func(nRemoters int) {
		var m1, m2, m3 runtime.MemStats
		runtime.ReadMemStats(&m1)

		opts := utils.WithLogger()

		sys1 := actor.NewActorSystem()
		defer sys1.Shutdown()

		ctx1 := sys1.Root
		r1 := router.NewTCPRouter(ctx1, nil, "127.0.0.1", 3000)
		defer r1.Shutdown()

		storeRef1 := store.Spawn(sys1.Root, r1, "store-1")
		defer ctx1.Stop(storeRef1.PID)

		time.Sleep(300 * time.Millisecond)
		runtime.ReadMemStats(&m2)

		single := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
		t.Logf("spawn single: allocated %.3f MB", single)

		// create remote connections
		pid := actor.NewPID("127.0.0.1:3000", "bootstrap")
		for i := range nRemoters {
			sysi := actor.NewActorSystem(opts)
			defer sysi.Shutdown()

			ri := router.NewTCPRouter(sysi.Root, pid, "127.0.0.1", 3001+i)
			defer ri.Shutdown()
		}

		time.Sleep(1000 * time.Millisecond)
		runtime.ReadMemStats(&m3)

		total := float64(m3.Alloc-m2.Alloc) / 1024 / 1024
		t.Logf("spawn %d remoters: allocated %.3f MB (average %.3f MB)", nRemoters, total, total/float64(nRemoters))
		WriteResult("nRemoters", nRemoters, "single_mb", single, "total_mb", total)
	}

	doTest(10)
	time.Sleep(3 * time.Second)
	runtime.GC()

	doTest(50)
	time.Sleep(3 * time.Second)
	runtime.GC()

	doTest(100)
}
