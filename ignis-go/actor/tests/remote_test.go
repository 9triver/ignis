package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
)

func TestVenvExecutor(t *testing.T) {
	cm := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	go func() {
		if err := cm.Run(ctx); err != nil {
		}
	}()

	manager, err := python.NewManager(ctx, cm)
	if err != nil {
		t.Fatal(err)
	}

	defer func(manager *python.VenvManager) { _ = manager.Close() }(manager)

	env, err := manager.GetVenv("test")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(env)

	fut := env.Execute("test", "call", nil)
	ret, err := fut.Result()
	t.Log(ret, err)
	<-ctx.Done()
}

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
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	go func() {
		if err := cm.Run(ctx); err != nil {
		}
	}()
	<-time.After(1 * time.Second)
	go rpcClient()
	c := cm.NextController()

	for msg := range c.RecvChan() {
		println(msg.String())
	}
}
