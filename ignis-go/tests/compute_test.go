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
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/remote/stub"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils"
)

func rpcClient(computeRef *proto.ActorRef) {
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

	err = stream.Send(controller.NewAppendData("session-0", &objects.Remote{
		ID:       "obj-1",
		Data:     []byte("123456"),
		Language: objects.LangJson,
	}))
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendActor("func", []string{"A", "B"}, computeRef))
	if err != nil {
		panic(err)
	}

	err = stream.Send(controller.NewAppendArgFromRef("session-0", "instance-0", "func", "A", &proto.Flow{
		ID:     "obj-1",
		Source: computeRef.Store,
	}))
	if err != nil {
		panic(err)
	}

	encoded, _ := objects.NewLocal(10, objects.LangJson).Encode()
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
	sys := actor.NewActorSystem(utils.WithLogger())
	cm := rpc.NewManager("127.0.0.1:8082")
	em := ipc.NewManager("ipc://" + path.Join(configs.StoragePath, "test-ipc"))
	ctx, cancel := context.WithTimeout(context.TODO(), 120*time.Second)
	defer cancel()

	storeRef := store.Spawn(sys.Root, stub.NewActorStub, "store")
	type Input struct {
		A int
		B int
	}

	taskFunc := functions.NewGo("graph-task", func(args Input) (ret int, err error) {
		return args.A + args.B, nil
	}, objects.LangJson)
	computePID := sys.Root.Spawn(compute.NewActor("graph-task", taskFunc, storeRef.PID))

	go func() {
		_ = cm.Run(ctx)
	}()
	<-time.After(1 * time.Second)
	ref := &proto.ActorRef{
		ID:    "graph-task",
		PID:   computePID,
		Store: storeRef,
	}
	go rpcClient(ref)

	venvs, err := functions.NewVenvManager(context.TODO(), em)
	if err != nil {
		panic(err)
	}

	wait := make(chan struct{})
	control.SpawnTaskController(sys.Root, storeRef, venvs, cm, func() { close(wait) })
	<-wait
}
