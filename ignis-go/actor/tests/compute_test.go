package actor_test

import (
	"context"
	"github.com/9triver/ignis/utils"
	"log/slog"
	"path"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/platform"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/configs"
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
		Source:   storePID,
	}))
	if err != nil {
		panic(err)
	}

	encoded, _ := proto.NewLocalObject(10, proto.LangJson).GetEncoded()
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

	storeProps := store.New()
	storePID := sys.Root.Spawn(storeProps)

	taskFunc := functions.NewGo("graph-task", demoFunc, proto.LangJson)
	computePID := sys.Root.Spawn(compute.NewActor("graph-task", taskFunc, storePID))

	go func() {
		_ = cm.Run(ctx)
	}()
	<-time.After(1 * time.Second)
	go rpcClient(storePID, computePID)

	venvs, err := python.NewManager(context.TODO(), em)
	if err != nil {
		panic(err)
	}
	c, props := platform.NewTaskController(storePID, venvs, cm)
	pid := sys.Root.Spawn(props)

	go func() {
		for msg := range c.RecvChan() {
			sys.Root.Send(pid, msg)
		}
	}()

	<-time.After(10 * time.Second)
}

func TestRemoteStream(t *testing.T) {
	storeProps := store.New()
	sys := actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {
		logger := utils.Logger()
		return logger.With("system", system.ID)
	}))
	storePID, _ := sys.Root.SpawnNamed(storeProps, "store")

	type N struct {
		Num int
	}

	type I struct {
		Ints <-chan int
	}

	type O struct {
		Sum int
	}

	generateInts := func(input N) (<-chan int, error) {
		ch := make(chan int)
		go func() {
			defer close(ch)
			for i := range input.Num {
				ch <- i
			}
		}()

		return ch, nil
	}

	getSum := func(input I) (O, error) {
		sum := 0
		for i := range input.Ints {
			println(i)
			sum += i
		}
		return O{Sum: sum}, nil
	}

	f1 := functions.NewGo("genInts", generateInts, proto.LangJson)
	f2 := functions.NewGo("getSum", getSum, proto.LangJson)

	props1 := compute.NewActor("f1", f1, storePID)
	props2 := compute.NewActor("f2", f2, storePID)
	pid1 := sys.Root.Spawn(props1)
	pid2 := sys.Root.Spawn(props2)

	sys.Root.Send(pid1, &proto.CreateSession{
		SessionID: "test",
		Successors: []*proto.Successor{
			{
				ID:    "f2",
				Param: "Ints",
				PID:   pid2,
			},
		},
	})

	sys.Root.Send(pid2, &proto.CreateSession{
		SessionID:  "test",
		Successors: []*proto.Successor{},
	})

	sys.Root.Send(storePID, &store.SaveObject{
		Value: proto.NewLocalObject(10, proto.LangJson),
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			ctx.Send(pid1, &proto.Invoke{
				SessionID: "test",
				Param:     "Num",
				Value:     ref,
			})
		},
	})

	<-time.After(100 * time.Second)
}
