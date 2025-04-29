package actor_test

import (
	"context"

	"path"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
)

func rpcClient(storePID, computePID *actor.PID) {
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

	err = stream.Send(controller.NewAppendData("session-0", &proto.EncodedObject{
		ID:       "obj-1",
		Data:     []byte("123456"),
		Language: proto.LangJson,
	}))
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendActor("func", []string{"A", "B"}, computePID))
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendArgFromRef("session-0", "instance-0", "func", "A", &proto.Flow{
		ObjectID: "obj-1",
		Source: &proto.StoreRef{
			ID:  "store",
			PID: storePID,
		},
	}))
	if err != nil {
		panic(err)
	}

	encoded, _ := messages.NewLocalObject(10, proto.LangJson).GetEncoded()
	err = stream.Send(controller.NewAppendArgFromEncoded("session-0", "instance-0", "func", "B", encoded))
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			result, err := stream.Recv()
			if err != nil {
				return
			}
			println(result.String())
		}
	}()

	//stream.CloseSend()
	<-stream.Context().Done()
}

func TestRemoteTask(t *testing.T) {
	sys := actor.NewActorSystem()
	cm := rpc.NewManager("127.0.0.1:8082")
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	storeRef := store.Spawn(sys.Root, remote.NewActorStub(sys), "store")
	type Input struct {
		A int
		B int
	}

	taskFunc := functions.NewGo("graph-task", func(args Input) (ret int, err error) {
		return args.A + args.B, nil
	}, proto.LangJson)
	computePID := sys.Root.Spawn(compute.NewActor("graph-task", taskFunc, storeRef.PID))

	go func() {
		_ = cm.Run(ctx)
	}()
	<-time.After(1 * time.Second)
	go rpcClient(storeRef.PID, computePID)

	venvs, err := python.NewManager(context.TODO(), em)
	if err != nil {
		panic(err)
	}

	wait := make(chan struct{})
	control.SpawnTaskController(sys.Root, storeRef, venvs, cm, func() { close(wait) })
	<-wait
}
