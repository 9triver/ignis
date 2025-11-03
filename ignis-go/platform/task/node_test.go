package task

import (
	"testing"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/transport/stub"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

func TestNodeWithCond(t *testing.T) {
	sys := actor.NewActorSystem()
	storeRef := store.Spawn(sys.Root, stub.NewActorStub(sys), "store")
	node := NodeFromFunction("123", functions.NewGo("test", func(struct {
		A int
	}) (int, error) {
		return 42, nil
	}, object.LangGo))

	rt := node.Runtime("test-01", storeRef.PID, "test")

	sys.Root.Send(storeRef.PID, &store.SaveObject{
		Value: object.NewLocal(100, object.LangGo),
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			rt.Invoke(nil, "A", ref)
		},
	})

	err := rt.Start(nil)
	if err != nil {
		t.Fatal(err)
	}
}
