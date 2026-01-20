package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

func main() {
	port := flag.Int("port", 3001, "port")
	flag.Parse()

	sysi := actor.NewActorSystem()
	defer sysi.Shutdown()

	pid := actor.NewPID("127.0.0.1:3000", "bootstrap")
	ri := router.NewTCPRouter(sysi.Root, pid, "127.0.0.1", *port)
	defer ri.Shutdown()

	ri.Register(&proto.StoreRef{ID: fmt.Sprintf("store-%d", *port), PID: actor.NewPID("127.0.0.1:3000", "store")})

	time.Sleep(5 * time.Second)
}
