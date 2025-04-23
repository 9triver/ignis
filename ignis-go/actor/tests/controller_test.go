package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/platform"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/asynkron/protoactor-go/actor"
)

func rpcClient() {
	conn, err := grpc.NewClient("127.0.0.1:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := controller.NewServiceClient(conn)
	stream, err := client.Session(context.TODO())
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewReady())
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendData("session-0", &proto.EncodedObject{
		ID:       "obj-1",
		Data:     []byte("123456"),
		Language: proto.Language_LANG_PYTHON,
	}))
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendArgFromRef("session-0", "instance-0", "func", "a", &proto.Flow{
		ObjectID: "obj-1",
		Source:   nil,
	}))
	if err != nil {
		panic(err)
	}

	stream.CloseSend()
	<-stream.Context().Done()
}

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
