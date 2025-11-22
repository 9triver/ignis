package store

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/transport/ipc"
)

func TestIPC(t *testing.T) {
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	go em.Run(ctx)

	manager, err := python.NewVenvManager(ctx, em)
	if err != nil {
		t.Fatal(err)
	}

	defer manager.Close()

	env, err := manager.GetVenv("test")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(env)

	params := make(map[string]object.Interface)
	params["a"] = object.NewLocal(10, object.LangJson)
	params["b"] = object.NewLocal(2, object.LangJson)
	fut := env.Execute("__add", "call", params)
	ret, err := fut.Result()
	t.Log(ret, err)
}

func TestIPCStream(t *testing.T) {
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	go em.Run(ctx)

	manager, err := python.NewVenvManager(ctx, em)
	if err != nil {
		t.Fatal(err)
	}

	defer manager.Close()

	env, err := manager.GetVenv("test")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(env)

	testStream2Obj := func(t *testing.T, env *python.VirtualEnv) {
		ints := make(chan int)
		go func() {
			defer close(ints)
			for i := 0; i < 10; i++ {
				ints <- i
			}
		}()
		params := make(map[string]object.Interface)
		params["ints"] = object.NewStream(ints, object.LangJson)
		fut := env.Execute("__sum", "call", params)
		ret, err := fut.Result()
		t.Log(ret, err)
	}

	t.Logf("testing stream to object...")
	testStream2Obj(t, env)
	time.Sleep(1 * time.Second)

	testObj2Stream := func(t *testing.T, env *python.VirtualEnv) {
		params := make(map[string]object.Interface)
		params["n"] = object.NewLocal(10, object.LangJson)
		fut := env.Execute("__gen", "call", params)
		ret, _ := fut.Result()

		s := ret.(*object.Stream)
		for obj := range s.ToChan() {
			t.Log(obj)
		}
	}

	t.Logf("testing object to stream...")
	testObj2Stream(t, env)
	time.Sleep(1 * time.Second)

	testStream2Stream := func(t *testing.T, env *python.VirtualEnv) {
		ints := make(chan int)
		go func() {
			defer close(ints)
			for i := 0; i < 10; i++ {
				ints <- i
			}
		}()
		params := make(map[string]object.Interface)
		params["ints"] = object.NewStream(ints, object.LangJson)
		fut := env.Execute("__map", "call", params)
		ret, _ := fut.Result()

		s := ret.(*object.Stream)
		for obj := range s.ToChan() {
			t.Log(obj)
		}
	}

	t.Logf("testing stream to stream...")
	testStream2Stream(t, env)
	time.Sleep(1 * time.Second)
}
