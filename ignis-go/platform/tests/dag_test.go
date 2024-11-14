package platform_test

import (
	"actors/platform/actor/compute"
	"actors/platform/actor/head"
	"actors/platform/dag"
	"actors/platform/handlers"
	"actors/platform/handlers/ipc"
	"actors/platform/store"
	"actors/platform/system"
	protoDAG "actors/proto/dag"
	"fmt"
	"os"
	"testing"
	"time"

	"runtime"

	"github.com/asynkron/protoactor-go/actor"
)

func memoryUsage(operation func()) uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before := m.Alloc

	operation()

	runtime.ReadMemStats(&m)
	usage := m.Alloc - before
	return usage
}

func timeCost(operation func()) int64 {
	tic := time.Now()

	operation()

	interval := time.Since(tic).Milliseconds()
	return interval
}

func TestDAG(t *testing.T) {
	if err := head.Start("127.0.0.1", 2312); err != nil {
		t.Fatal(err)
	}

	sys := system.ActorSystem()

	props := actor.PropsFromFunc(func(c actor.Context) {}, actor.WithOnInit(func(ctx actor.Context) {
		dag := dag.New("test-dag", ctx)

		dag.AddNodeFromFunction("node1", handlers.NewGo("f1", func(req *struct{ A int }) (int, error) {
			t.Logf("node1 req.A=%d", req.A)
			return req.A * 3, nil
		}, store.LangJson), 0)

		dag.AddDataNode("driver", 1, store.LangJson)
		dag.AddEdge("driver", "node1", "A")

		ch, _ := dag.AddOutputEdge("node1")

		dag.Serve()
		defer dag.Destroy()

		<-time.After(1 * time.Second)
		dag.Execute("exec1")

		res := <-ch
		t.Log(res.Proto)
	}))

	sys.Root.Spawn(props)
}

func startupNative(t *testing.T, actors int) {
	if err := head.Start("127.0.0.1", 2312); err != nil {
		t.Fatal(err)
	}
	sys := system.ActorSystem()

	f1 := handlers.DeclareTyped[struct{ A int }, int]("f1")
	impl1 := handlers.ImplGo(f1, func(req *struct{ A int }) (int, error) {
		<-time.After(500 * time.Millisecond)
		return req.A + 3, nil
	}, store.LangJson)

	for i := range actors {
		_ = sys.Root.Spawn(compute.NewNamedActor(fmt.Sprintf("node1-%d", i), impl1, system.LatencyNotSpecified).Props())
	}
}

func startupDynamic(t *testing.T, actors int) {
	if err := head.Start("127.0.0.1", 2312); err != nil {
		t.Fatal(err)
	}
	sys := system.ActorSystem()

	f1 := handlers.DeclareTyped[struct{ A int }, int]("f1")
	m := ipc.GetVenvManager()
	defer m.Close()

	v, err := m.GetVenv("test")
	if err != nil {
		t.Fatal(err)
	}

	m.Run()
	for i := range actors {
		pkl, err := os.ReadFile("func.pkl")
		if err != nil {
			t.Fatal(err)
		}
		impl1, err := handlers.ImplPyFunc(f1, "test", []string{}, pkl)
		if err != nil {
			t.Fatal(err)
		}
		addHandler := ipc.MakeAddHandler(fmt.Sprintf("add-%d", i), pkl, store.LangJson, []string{"add"})
		v.Tell(ipc.MakeRouterMessage(addHandler))
		_ = sys.Root.Spawn(compute.NewNamedActor(fmt.Sprintf("node1-%d", i), impl1, system.LatencyNotSpecified).Props())
	}
}

func TestStartupNative(t *testing.T) {
	mem := memoryUsage(func() {
		time := timeCost(func() {
			startupNative(t, 1000)
		})
		t.Logf("Time cost (Native): %d ms", time)
	})
	t.Logf("Memory cost (Native): %f KB", float64(mem)/1024)
}

func TestStartupDynamic(t *testing.T) {
	mem := memoryUsage(func() {
		time := timeCost(func() {
			startupDynamic(t, 100)
		})
		t.Logf("Time cost (Dynamic): %d ms", time)
	})
	t.Logf("Memory cost (Dynamic): %f KB", float64(mem)/1024)
}

func TestGroup(t *testing.T) {
	if err := head.Start("127.0.0.1", 2312); err != nil {
		t.Fatal(err)
	}

	sys := system.ActorSystem()
	s := head.ActorRef().Store()
	f1 := handlers.DeclareTyped[struct{ A int }, int]("f1")
	f2 := handlers.DeclareTyped[struct{ A, B int }, int]("f2")
	logger := sys.Logger()

	impl1 := handlers.ImplGo(f1, func(req *struct{ A int }) (int, error) {
		logger.Info("node1", "a", req.A, "lat", "1s")
		<-time.After(500 * time.Millisecond)
		return req.A + 3, nil
	}, store.LangJson)

	impl2 := handlers.ImplGo(f2, func(req *struct{ A, B int }) (int, error) {
		logger.Info("node2", "a", req.A, "b", req.B)
		return req.A + req.B, nil
	}, store.LangJson)

	for i := range 10 {
		_ = sys.Root.Spawn(compute.NewNamedActor(fmt.Sprintf("node1-%d", i), impl1, system.LatencyNotSpecified).Props())
	}
	node2 := sys.Root.Spawn(compute.NewActor(impl2, system.LatencyNotSpecified).Props())

	tasks := 1000
	done := make(chan struct{}, tasks)
	consumer := actor.PropsFromFunc(func(c actor.Context) {
		done <- struct{}{}
		switch msg := c.Message().(type) {
		case *protoDAG.ExecutionResult:
			println(msg.ExecutionID, msg.Results, msg.Error)
		}
	})
	replyTo := sys.Root.Spawn(consumer)
	<-time.After(1 * time.Second)

	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.Create{}, replyTo))
	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendNode{
		Name:   "node1",
		Params: f1.Params(),
		Option: &protoDAG.AppendNode_Group{Group: "f1"},
	}, replyTo))
	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendNode{
		Name:   "node2",
		Params: f2.Params(),
		Option: &protoDAG.AppendNode_Actor{Actor: node2},
	}, replyTo))
	obj, _ := s.Add(1, store.LangJson).Encode()
	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendNode{
		Name:   "driver",
		Option: &protoDAG.AppendNode_Value{Value: obj},
	}, replyTo))

	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
		From:  "node1",
		To:    "node2",
		Param: "A",
	}, replyTo))
	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
		From:  "driver",
		To:    "node1",
		Param: "A",
	}, replyTo))
	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendEdge{
		From:  "driver",
		To:    "node2",
		Param: "B",
	}, replyTo))

	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.AppendOutput{
		Node: "node2",
	}, replyTo))

	sys.Root.Send(head.ActorRef().DAGManager(), dag.MakeCommand("test-dag", &protoDAG.Serve{}, replyTo))

	manager := head.ActorRef().DAGManager()
	execute := dag.MakeCommand("test-dag", &protoDAG.Execute{}, replyTo)

	tic := time.Now()
	for range tasks {
		sys.Root.Send(manager, execute)
	}
	for range tasks {
		<-done
	}
	cost := time.Since(tic).Milliseconds()
	t.Log("time", cost)
}
