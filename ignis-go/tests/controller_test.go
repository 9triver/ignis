package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/platform/control"
	"github.com/asynkron/protoactor-go/actor"
)

func TestController(t *testing.T) {
	cm := rpc.NewManager("127.0.0.1:8082")
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	sys := actor.NewActorSystem()
	storeRef := store.Spawn(sys.Root, remote.NewActorStub(sys), "store")

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

	venvs, err := functions.NewVenvManager(ctx, em)
	if err != nil {
		panic(err)
	}

	env, err := venvs.GetVenv("test2")
	t.Log(env, err)

	wait := make(chan struct{})
	control.SpawnTaskController(sys.Root, storeRef, venvs, cm, func() { close(wait) })
	<-wait
}
