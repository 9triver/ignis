package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/platform"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
	"github.com/asynkron/protoactor-go/actor"
)

func TestController(t *testing.T) {
	cm := rpc.NewManager("127.0.0.1:8082")
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	sys := actor.NewActorSystem()
	props := store.New(store.NewActorStub(sys), "store")
	storePID := sys.Root.Spawn(props)

	ctx, cancel := context.WithTimeout(context.TODO(), 1000*time.Second)
	defer cancel()

	go func() {
		if err := cm.Run(ctx); err != nil {
		}
	}()
	go func() {
		if err := em.Run(ctx); err != nil {
		}
	}()
	<-time.After(1 * time.Second)

	venvs, err := python.NewManager(ctx, em)
	if err != nil {
		panic(err)
	}

	env, err := venvs.GetVenv("test2")
	t.Log(env, err)

	c, props := platform.NewTaskController(storePID, venvs, cm)
	pid := sys.Root.Spawn(props)

	for msg := range c.RecvChan() {
		sys.Root.Send(pid, msg)
	}
}
