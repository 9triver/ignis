package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
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

	// I -> O
	testDirect(t, env)
	<-time.After(1 * time.Second)

	// [I] -> O
	testJoin(t, env)
	<-time.After(1 * time.Second)

	// I -> [O]
	testStream(t, env)
	<-time.After(1 * time.Second)

	// [I] -> [O]
	testMap(t, env)
	<-ctx.Done()
}

func testDirect(t *testing.T, env *python.VirtualEnv) {
	params := make(map[string]messages.Object)
	params["a"] = messages.NewLocalObject(10, proto.LangJson)
	params["b"] = messages.NewLocalObject(2, proto.LangJson)
	fut := env.Execute("__add", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testJoin(t *testing.T, env *python.VirtualEnv) {
	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := 0; i < 10; i++ {
			ints <- i
		}
	}()
	params := make(map[string]messages.Object)
	params["ints"] = messages.NewLocalStream(ints, proto.LangJson)
	fut := env.Execute("__sum", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testStream(t *testing.T, env *python.VirtualEnv) {
	params := make(map[string]messages.Object)
	params["n"] = messages.NewLocalObject(10, proto.LangJson)
	fut := env.Execute("__gen", "call", params)
	ret, _ := fut.Result()

	s := ret.(*messages.LocalStream)
	for obj := range s.ToChan() {
		t.Log(obj)
	}
}

func testMap(t *testing.T, env *python.VirtualEnv) {
	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := 0; i < 10; i++ {
			ints <- i
		}
	}()
	params := make(map[string]messages.Object)
	params["ints"] = messages.NewLocalStream(ints, proto.LangJson)
	fut := env.Execute("__map", "call", params)
	ret, _ := fut.Result()

	s := ret.(*messages.LocalStream)
	for obj := range s.ToChan() {
		t.Log(obj)
	}
}
