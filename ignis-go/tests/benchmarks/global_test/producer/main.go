package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
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

func doTest(host string, bytes, q int) {
	opts := utils.WithLogger()

	log.Printf("bytes: %d, concurrent: %d\n", bytes, q)

	sys1 := actor.NewActorSystem(opts)
	defer sys1.Shutdown()

	ctx1 := sys1.Root
	r1 := router.NewTCPRouter(ctx1, nil, host, 3000)
	defer r1.Shutdown()

	storeRef1 := store.Spawn(sys1.Root, r1, "store-1")
	defer ctx1.Stop(storeRef1.PID)
	log.Printf("Store: %v\n", storeRef1)

	time.Sleep(100 * time.Millisecond)

	wg := sync.WaitGroup{}
	wg.Add(q)

	producer := ctx1.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		for i := range q {
			obj := generatorObject(bytes)
			c.Send(storeRef1.PID, &store.SaveObject{
				Value: object.LocalWithID(fmt.Sprintf("obj-%d", i), obj, object.LangJson),
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					wg.Done()
				},
			})
		}
	}))
	defer ctx1.Stop(producer)

	log.Println("Producer done")
	time.Sleep(300 * time.Second)
}

func main() {
	host := flag.String("host", "127.0.0.1", "hostname")
	bytes := flag.Int("bytes", 1024*1024, "bytes")
	q := flag.Int("q", 100, "q")

	flag.Parse()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	go func() {
		<-ch
		os.Exit(0)
	}()

	doTest(*host, *bytes, *q)
}
