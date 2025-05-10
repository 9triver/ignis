package actor

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/objects"
)

func TestVenvExecutor(t *testing.T) {
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	go func() {
		if err := em.Run(ctx); err != nil {
		}
	}()

	manager, err := functions.NewVenvManager(ctx, em)
	if err != nil {
		t.Fatal(err)
	}

	defer func(manager *functions.VenvManager) { _ = manager.Close() }(manager)

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

func testDirect(t *testing.T, env *functions.VirtualEnv) {
	params := make(map[string]objects.Interface)
	params["a"] = objects.NewLocal(10, objects.LangJson)
	params["b"] = objects.NewLocal(2, objects.LangJson)
	fut := env.Execute("__add", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testJoin(t *testing.T, env *functions.VirtualEnv) {
	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := 0; i < 10; i++ {
			ints <- i
		}
	}()
	params := make(map[string]objects.Interface)
	params["ints"] = objects.NewStream(ints, objects.LangJson)
	fut := env.Execute("__sum", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func testStream(t *testing.T, env *functions.VirtualEnv) {
	params := make(map[string]objects.Interface)
	params["n"] = objects.NewLocal(10, objects.LangJson)
	fut := env.Execute("__gen", "call", params)
	ret, _ := fut.Result()

	s := ret.(*objects.Stream)
	for obj := range s.ToChan() {
		t.Log(obj)
	}
}

func testMap(t *testing.T, env *functions.VirtualEnv) {
	ints := make(chan int)
	go func() {
		defer close(ints)
		for i := 0; i < 10; i++ {
			ints <- i
		}
	}()
	params := make(map[string]objects.Interface)
	params["ints"] = objects.NewStream(ints, objects.LangJson)
	fut := env.Execute("__map", "call", params)
	ret, _ := fut.Result()

	s := ret.(*objects.Stream)
	for obj := range s.ToChan() {
		t.Log(obj)
	}
}
