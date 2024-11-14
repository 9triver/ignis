package system

import (
	"actors/platform/store"
	"actors/proto"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"google.golang.org/grpc"
)

type WorkerActor interface {
	actor.Actor

	Head() *actor.PID
	Self() *actor.PID
	Name() string
	Store() *store.Store

	SetPID(pid *actor.PID)
	DeployManager() *actor.PID
}

type HeadActor interface {
	WorkerActor

	ActorSet(name string) (*actor.PIDSet, bool)
	Latencies() *ActorLatencies
	Workers() *actor.PIDSet
	DAGManager() *actor.PID
}

var (
	system      *actor.ActorSystem
	remoter     *remote.Remote
	workerActor WorkerActor
	headActor   HeadActor
)

const (
	kDefaultSystemTimeout      = 30 * time.Second
	kDefaultExecutionTimeout   = 300 * time.Second
	kDefaultSystemName         = "platform_head"
	kDefaultChannelBufferSize  = 50
	kDefaultMaximumMessageSize = 4 * 1024 * 1024

	LatencyNotSpecified time.Duration = 1 * time.Millisecond // unspecified latency, default to 1ms
	LatencyZero         time.Duration = 0                    // never updates latency
)

var (
	once sync.Once

	RequestTimeout     = kDefaultSystemTimeout    // system requests
	FlowTimeout        = kDefaultSystemTimeout    // flow object requests
	ExecutionTimeout   = kDefaultExecutionTimeout // actor execution
	SystemName         = kDefaultSystemName
	ChannelBufferSize  = kDefaultChannelBufferSize
	MaximumMessageSize = kDefaultMaximumMessageSize
)

func Logger() *slog.Logger {
	return ActorSystem().Logger()
}

func ActorSystem() *actor.ActorSystem {
	once.Do(func() {
		system = actor.NewActorSystem()
	})

	return system
}

func Spawn(props *actor.Props) *actor.PID {
	return ActorSystem().Root.Spawn(props)
}

func SpawnNamed(props *actor.Props, name string) (*actor.PID, error) {
	return ActorSystem().Root.SpawnNamed(props, name)
}

func Start(worker WorkerActor, localAddr string, port int) (*actor.PID, error) {
	if workerActor != nil {
		return nil, errors.New("worker already started")
	}

	config := remote.Configure(
		localAddr, port,
		remote.WithServerOptions(grpc.MaxRecvMsgSize(MaximumMessageSize)),
	)
	remoter = remote.NewRemote(ActorSystem(), config)
	remoter.Start()

	workerActor = worker
	isHead := false
	if head, ok := worker.(HeadActor); ok {
		headActor = head
		isHead = true
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return workerActor
	})

	if isHead {
		return ActorSystem().Root.SpawnNamed(props, SystemName)
	}
	return ActorSystem().Root.Spawn(props), nil
}

func Shutdown() {
	system.Shutdown()
	remoter.Shutdown(true)
	workerActor = nil
	headActor = nil
}

func ActorRef() WorkerActor {
	return workerActor
}

func HeadRef() HeadActor {
	return headActor
}

func RequestFlowFuture(ctx actor.SenderContext, flow *proto.Flow) *actor.Future {
	if flow == nil {
		panic("flow is nil")
	}
	return ctx.RequestFuture(flow.Actor, &proto.ObjectRequest{
		Sender: ctx.Self(),
		ID:     flow.ObjectID,
	}, FlowTimeout)
}

func ExtractFlowFuture(flow *proto.Flow, fut *actor.Future) (*store.Object, error) {
	resp, err := fut.Result()
	if err != nil {
		return nil, err
	}

	switch objResp := resp.(type) {
	case *proto.Error:
		return nil, errors.New(objResp.Message)
	case *proto.EncodedObject:
		if objResp.ID != flow.ObjectID {
			return nil, errors.New("unexpected objectID")
		}
		obj := &store.Object{}
		if err := obj.Decode(objResp); err != nil {
			return nil, err
		}
		return obj, nil
	case *store.Object:
		return objResp, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func FlowData(ctx actor.SenderContext, flow *proto.Flow) (*store.Object, error) {
	future := RequestFlowFuture(ctx, flow)
	return ExtractFlowFuture(flow, future)
}
