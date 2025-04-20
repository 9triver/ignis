package actor_test

import (
	"github.com/9triver/ignis/actor/store"
	"testing"
	"time"

	"github.com/9triver/ignis/proto"
	console "github.com/asynkron/goconsole"
	"github.com/asynkron/protoactor-go/actor"
)

func TestObject(t *testing.T) {
	v := map[string]any{
		"name":  "b",
		"value": 100.0,
		"id":    10,
	}
	obj := store.NewLocalObject(v, proto.LangGo)
	enc, err := obj.GetEncoded()
	if err != nil {
		t.Fatal("encode", err)
	}
	t.Logf("Encoded: %v -> %v\n", v, enc)

	dec, err := enc.GetValue(nil)
	if err != nil {
		t.Fatal("decode", err)
	}
	t.Logf("Decoded: %v -> %v\n", enc, dec)
}

func TestActor(t *testing.T) {
	system := actor.NewActorSystem()
	storeProps := store.New()
	props := actor.PropsFromFunc(func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case proto.Object:
			ctx.Logger().Info("receive", "object", msg)
		case *proto.Error:
			ctx.Logger().Info("receive", "error", msg)
		}
	}, actor.WithOnInit(func(ctx actor.Context) {
		pid := ctx.Spawn(storeProps)
		ctx.Send(pid, store.NewLocalObject(1000, proto.LangJson))
		enc, _ := store.NewLocalObject(1000, proto.LangJson).GetEncoded()
		ctx.Send(pid, enc)
		ctx.Logger().Info("store", "object", enc.ID)

		ctx.Send(pid, &proto.ObjectRequest{ID: "obj1", ReplyTo: ctx.Self()})
	}))

	system.Root.Spawn(props)
	<-time.After(time.Second * 5)
	_, _ = console.ReadLine()
}
