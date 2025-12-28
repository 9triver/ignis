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
	t.Logf("[go] connect to venv %s", env.Name)

	params := make(map[string]object.Interface)
	params["a"] = object.NewLocal(20, object.LangJson)
	params["b"] = object.NewLocal(20, object.LangJson)

	t.Logf("[go] execute object-object function")
	fut := env.Execute("__add", "call", params)
	ret, err := fut.Result()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("[go] receive from python: %v", ret)
}

func TestIPCStream2Object(t *testing.T) {
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
	t.Logf("[go] connect to venv %s", env.Name)

	testStream2Obj := func(t *testing.T, env *python.VirtualEnv) {
		ints := make(chan int, 10)
		go func() {
			defer close(ints)
			for i := range 10 {
				ints <- i
			}
		}()
		params := make(map[string]object.Interface)
		params["ints"] = object.NewStream(ints, object.LangJson)
		t.Logf("[go] execute stream-object function")
		fut := env.Execute("__sum", "call", params)
		ret, _ := fut.Result()
		t.Logf("[go] receive from python: %s (isStream = %t)", ret.GetID(), ret.IsStream())
		v, _ := ret.Value()
		t.Logf("[go] receive object: %v", v)
	}

	testStream2Obj(t, env)
}

func TestIPCObject2Stream(t *testing.T) {
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
	t.Logf("[go] connect to venv %s", env.Name)

	testObj2Stream := func(t *testing.T, env *python.VirtualEnv) {
		params := make(map[string]object.Interface)
		params["n"] = object.NewLocal(10, object.LangJson)
		t.Logf("[go] execute object-stream function")
		fut := env.Execute("__gen", "call", params)
		ret, _ := fut.Result()

		t.Logf("[go] receive from python: %s (isStream = %t)", ret.GetID(), ret.IsStream())
		s := ret.(*object.Stream)
		for obj := range s.ToChan() {
			t.Logf("[go] receive chunk from python: %v", obj)
		}
	}

	testObj2Stream(t, env)
}

func TestIPCStream2Stream(t *testing.T) {
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
	t.Logf("[go] connect to venv %s", env.Name)

	testStream2Stream := func(t *testing.T, env *python.VirtualEnv) {
		ints := make(chan int)
		go func() {
			defer close(ints)
			for i := range 10 {
				ints <- i
			}
		}()
		params := make(map[string]object.Interface)
		params["ints"] = object.NewStream(ints, object.LangJson)
		t.Logf("[go] execute stream-stream function")

		fut := env.Execute("__map", "call", params)
		ret, _ := fut.Result()
		t.Logf("[go] receive from python: %s (isStream = %t)", ret.GetID(), ret.IsStream())
		s := ret.(*object.Stream)
		for obj := range s.ToChan() {
			t.Logf("[go] receive chunk from python: %v", obj)
		}
	}

	testStream2Stream(t, env)
}
