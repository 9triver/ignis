package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/asynkron/protoactor-go/actor"
)

func doTest(host, remote string, bytes, q int) {
	opts := utils.WithLogger()

	log.Printf("bytes: %d, concurrent: %d\n", bytes, q)

	sys2 := actor.NewActorSystem(opts)
	defer sys2.Shutdown()

	ctx2 := sys2.Root
	bootstrap := actor.NewPID(remote, "bootstrap")
	r2 := router.NewTCPRouter(ctx2, bootstrap, host, 3001)
	defer r2.Shutdown()

	storeRef2 := store.Spawn(sys2.Root, r2, "store-2")
	defer ctx2.Stop(storeRef2.PID)

	storeRef1 := &proto.StoreRef{
		ID:  "store-1",
		PID: actor.NewPID(remote, "store-1"),
	}

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(q)

	var tic time.Time
	collector := ctx2.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *actor.Started:
			tic = time.Now()
			for i := range q {
				c.Send(storeRef2.PID, &store.RequestObject{
					ReplyTo: c.Self(),
					Flow: &proto.Flow{
						ID:     fmt.Sprintf("obj-%d", i),
						Source: storeRef1,
					},
				})
			}
		case *store.ObjectResponse:
			wg.Done()

			_, err := msg.Value.Value()
			if err != nil {
				log.Fatal(err)
			}
		}
	}))
	defer ctx2.Stop(collector)

	wg.Wait()
	totalLatency := time.Since(tic)
	averageLatency := totalLatency / time.Duration(q)
	transmitted := float64(bytes) * float64(q) / 1024 / 1024
	speed := transmitted / totalLatency.Seconds()
	log.Printf("Average load latency: %v\n", averageLatency)
	log.Printf("Total load latency: %v\n", totalLatency)
	log.Printf("Transmitted: %.2f MB, Speed: %.2f MB/s\n", transmitted, speed)

	stat := map[string]any{
		"bytes":          bytes,
		"q":              q,
		"lat_ns":         averageLatency.Nanoseconds(),
		"speed_mbps":     speed,
		"transmitted_mb": transmitted,
	}

	line, _ := json.Marshal(stat)
	fmt.Println(string(line))
}

func main() {
	host := flag.String("host", "127.0.0.1", "hostname")
	remote := flag.String("remote", "127.0.0.1:3000", "remote hostname")
	bytes := flag.Int("bytes", 1024*1024, "bytes")
	q := flag.Int("q", 100, "q")

	flag.Parse()

	doTest(*host, *remote, *bytes, *q)
}
