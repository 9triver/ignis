package head

import (
	"actors/platform/actor/worker"
	"actors/platform/dag"
	"actors/platform/system"
	"actors/proto"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type Actor struct {
	*worker.Actor
	dagManager *actor.PID
	latencies  *system.ActorLatencies
	workers    *actor.PIDSet
	actors     map[string]*actor.PIDSet
}

func (node *Actor) ActorSet(name string) (*actor.PIDSet, bool) {
	set, ok := node.actors[name]
	return set, ok
}

func (node *Actor) Latencies() *system.ActorLatencies {
	return node.latencies
}

func (node *Actor) Workers() *actor.PIDSet {
	return node.workers
}

func (node *Actor) DAGManager() *actor.PID {
	return node.dagManager
}

func (node *Actor) onLinkResponse(ctx actor.Context, res *proto.LinkResponse) {
	currentTs := time.Now().UnixMilli()
	latency := currentTs - res.SchedulerTime
	ctx.Logger().Info("Link latency", "source", res.Source, "target", res.Target, "latency", latency)
	node.latencies.Link.Set(res.Source.Address, res.Target.Address, latency)
}

func (node *Actor) onWorkerRegister(ctx actor.Context, msg *proto.WorkerRegister) {
	ctx.Send(msg.Sender, &proto.LinkPong{
		SchedulerTime: time.Now().UnixMilli(),
		Scheduler:     ctx.Self(),
		Source:        ctx.Self(),
	})

	for _, worker := range node.workers.Values() {
		ctx.Send(worker, &proto.LinkPing{
			SchedulerTime: time.Now().UnixMilli(),
			Scheduler:     ctx.Self(),
			Target:        msg.Sender,
		})
	}

	node.workers.Add(msg.Sender)
	ctx.Logger().Info("Worker registered", "worker", msg.Sender)
}

func (node *Actor) onActorReady(msg *proto.ActorReady) {
	node.latencies.Process.Set(msg.Sender, msg.ProcessLatency)
	if s, ok := node.actors[msg.GroupName]; !ok {
		node.actors[msg.GroupName] = actor.NewPIDSet(msg.Sender)
	} else {
		s.Add(msg.Sender)
	}
}

func (node *Actor) onActorStopped(msg *proto.ActorStopped) {
	node.latencies.Process.Delete(msg.Sender)
	if s, ok := node.actors[msg.ActorName]; ok {
		s.Remove(msg.Sender)
		if s.Len() == 0 {
			delete(node.actors, msg.ActorName)
		}
	}
}

func (node *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.LinkPing:
		node.OnLinkPing(ctx, msg)
	case *proto.LinkPong:
		node.OnLinkPong(ctx, msg)
	case *proto.ObjectRequest:
		node.OnObjectRequest(ctx, msg)
	case *proto.WorkerRegister:
		node.onWorkerRegister(ctx, msg)
	case *proto.LinkResponse:
		node.onLinkResponse(ctx, msg)
	case *proto.ActorReady:
		node.onActorReady(msg)
	case *proto.ActorStopped:
		node.onActorStopped(msg)
	default:
		ctx.Logger().Info("Unknown message", "msg", msg)
	}
}

func ActorRef() *Actor {
	return system.ActorRef().(*Actor)
}

func Start(addr string, port int) error {
	sys := system.ActorSystem()
	logger := sys.Logger()

	// config := remote.Configure(
	// 	addr, port,
	// 	remote.WithServerOptions(grpc.MaxRecvMsgSize(system.MaximumMessageSize)),
	// )
	// remoter := remote.NewRemote(sys, config)
	// remoter.Start()
	logger.Info("Platform starting", "port", port)

	head := &Actor{
		Actor:      worker.NewActor(nil),
		dagManager: sys.Root.Spawn(dag.NewManager()),
		latencies:  system.NewLatencies(),
		workers:    actor.NewPIDSet(),
		actors:     make(map[string]*actor.PIDSet),
	}

	pid, err := system.Start(head, addr, port)
	logger.Info("Started head actor", "pid", pid)
	if err != nil {
		return fmt.Errorf("failed to start head actor: %w", err)
	}
	head.SetPID(pid)

	return nil
}
