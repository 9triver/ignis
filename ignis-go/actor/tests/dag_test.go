package actor_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/dag"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Input struct {
	A int
	B int
}

type Output struct {
	Sum int
}

func demoFunc(input Input) (Output, error) {
	return Output{Sum: input.A + input.B}, nil
}

func TestDAGWithLocal(t *testing.T) {
	var nodes []dag.Node
	storeProps := store.New()
	sys := actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {
		return utils.Logger()
	}))

	storePID, _ := sys.Root.SpawnNamed(storeProps, "store")
	obj := store.NewLocalObject(100, proto.LangGo)
	sys.Root.Send(storePID, &store.SaveObject{
		Value: obj,
	})

	ref := &proto.Flow{
		ObjectID: obj.ID,
		Source:   storePID,
	}
	entry := dag.NewEntryNode("graph-entry", ref)

	taskFunc := functions.NewGo("graph-task", demoFunc, proto.LangGo)
	task := dag.TaskNodeFromFunction("graph-task", taskFunc)
	exit := dag.NewExitNode("graph-exit")

	nodes = append(nodes, entry, task, exit)
	graph := dag.New("test-graph", nodes...)
	graph.AddEdge("graph-entry", "graph-task", "A")
	graph.AddEdge("graph-entry", "graph-task", "B")
	graph.AddEdge("graph-task", "graph-exit", "result")
	props := graph.RootProps("session0", storePID)
	pid := sys.Root.Spawn(props)
	sys.Root.Send(pid, &proto.InvokeEmpty{})
	<-time.After(10 * time.Second)
}

func TestDAGWithRemote(t *testing.T) {
	var nodes []dag.Node
	storeProps := store.New()
	sys := actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {
		logger := utils.Logger()
		return logger.With("system", system.ID)
	}))
	storePID, _ := sys.Root.SpawnNamed(storeProps, "store")
	obj := store.NewLocalObject(100, proto.LangGo)
	sys.Root.Send(storePID, &store.SaveObject{
		Value: obj,
	})

	ref := &proto.Flow{
		ObjectID: obj.ID,
		Source:   storePID,
	}
	entry := dag.NewEntryNode("graph-entry", ref)

	taskFunc := functions.NewGo("graph-task", demoFunc, proto.LangGo)
	computePID := sys.Root.Spawn(compute.NewActor("graph-task", taskFunc, storePID))
	task := dag.TaskNodeFromPID("graph-task", taskFunc.Params(), computePID)
	exit := dag.NewExitNode("graph-exit")

	nodes = append(nodes, entry, task, exit)
	graph := dag.New("test-graph", nodes...)
	graph.AddEdge("graph-entry", "graph-task", "A")
	graph.AddEdge("graph-entry", "graph-task", "B")
	graph.AddEdge("graph-task", "graph-exit", "result")
	props := graph.Props("session0", storePID)
	pid := sys.Root.Spawn(props)
	sys.Root.Send(pid, &proto.InvokeEmpty{})
	<-time.After(10 * time.Second)
}
