package main

import (
	"fmt"
	"time"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

type DemoInput struct {
	A int
	B int
}

type DemoOutput = int

func Add(inputs DemoInput) (o DemoOutput, err error) {
	fmt.Printf("Calling function `Add` with Input { A = %d, B = %d }\n", inputs.A, inputs.B)
	o = inputs.A + inputs.B
	fmt.Printf("Calling function `Add` with Output = %d\n", o)
	return
}

func main() {
	sys := actor.NewActorSystem()
	ctx := sys.Root
	storeRef := store.Spawn(sys.Root, nil, "store")

	node := task.NewNode(
		"demo",
		[]string{"A", "B"},
		task.ProducerFromFunction(functions.NewGo("add", Add, object.LangGo)),
	)

	wait := make(chan struct{})
	collector := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *proto.InvokeResponse:
			c.Logger().Info("receive response", "result", msg.Result)
			obj, _ := store.GetObject(c, storeRef.PID, msg.Result).Result()
			fmt.Println(obj.Value())
			close(wait)
		}
	}))
	defer ctx.Stop(collector)
	router.Register("collector", collector)

	rt := node.Runtime("session-0", storeRef.PID, "collector")
	pid := ctx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch c.Message().(type) {
		case *actor.Started:
			c.Send(storeRef.PID, &store.SaveObject{
				Value: object.NewLocal(10, object.LangGo),
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					rt.Invoke(c, "A", ref)
				},
			})

			c.Send(storeRef.PID, &store.SaveObject{
				Value: object.NewLocal(20, object.LangGo),
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					rt.Invoke(c, "B", ref)
				},
			})

			rt.Start(c)
		}
	}))
	defer ctx.Stop(pid)

	<-wait
	time.Sleep(1 * time.Hour)
}
