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

	env, err := manager.GetVenv("test2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(env)

	testCall(t, env)
	testStream(t, env)
	testGenerate(t, env)

	<-ctx.Done()
}

func testCall(t *testing.T, env *python.VirtualEnv) {
	params := make(map[string]proto.Object)
	params["a"] = proto.NewLocalObject(10, proto.LangJson)
	params["b"] = proto.NewLocalObject(2, proto.LangJson)
	fut := env.Execute(nil, "__add", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testStream(t *testing.T, env *python.VirtualEnv) {
	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := 0; i < 10; i++ {
			ints <- i
		}
	}()
	params := make(map[string]proto.Object)
	params["ints"] = proto.NewLocalStream(ints, proto.LangJson)
	fut := env.Execute(nil, "__sum", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testGenerate(t *testing.T, env *python.VirtualEnv) {
	params := make(map[string]proto.Object)
	params["n"] = proto.NewLocalObject(10, proto.LangJson)
	fut := env.Execute(nil, "__generate", "call", params)
	ret, _ := fut.Result()

	s, _ := ret.ToStream()
	for v := range s.ToChan(nil) {
		t.Log(v)
	}
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
