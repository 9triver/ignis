package compute

import (
	"actors/platform/handlers"
	"actors/platform/store"
	"actors/platform/system"
	"actors/platform/utils"
	"actors/proto"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type State int

const (
	StateCreated State = iota
	StateReady
	StateRunning
	StateError
	StateFinished
)

type LifeCycle int

const (
	LifeCycleOnce LifeCycle = iota
	LifeCycleLoop
)

type Actor struct {
	name           string
	self           *actor.PID
	execActor      *actor.PID
	state          State
	lifecycle      LifeCycle
	store          *store.Store
	processLatency int64
	handler        handlers.Function
}

func (a *Actor) Name() string {
	return a.name
}

func (a *Actor) Deps() utils.Set[string] {
	return a.handler.ParamSet()
}

func (a *Actor) State() State {
	return a.state
}

func (a *Actor) LifeCycle() LifeCycle {
	return a.lifecycle
}

func (a *Actor) handleObjectRequest(ctx actor.Context, msg *proto.ObjectRequest) {
	obj, ok := a.store.Get(msg.ID)
	if !ok {
		ctx.Respond(&proto.Error{
			Message: "object not found",
			Sender:  msg.Sender,
		})
		return
	}

	if utils.IsSameSystem(ctx.Self(), msg.Sender) {
		ctx.Respond(obj)
		return
	}

	encoded, err := obj.Encode()
	if err != nil {
		ctx.Respond(&proto.Error{
			Message: "error encoding object: " + err.Error(),
			Sender:  msg.Sender,
		})
		return
	}

	ctx.Respond(encoded)
}

func (a *Actor) doExecute(ctx actor.Context, execute *proto.Execute) (*store.Object, error) {
	params := make(map[string]*store.Object)
	futures := make(map[string]*actor.Future)

	for param := range a.Deps() {
		flow, ok := execute.Flows[param]
		if !ok {
			return nil, fmt.Errorf("handler %s: missing flow %s", a.handler.Name(), param)
		}
		futures[param] = system.RequestFlowFuture(ctx, flow)
	}

	for param, future := range futures {
		flow := execute.Flows[param]
		data, err := system.ExtractFlowFuture(flow, future)
		if err != nil {
			return nil, fmt.Errorf("handler %s: error fetching flow %s: %w", a.handler.Name(), param, err)
		}
		params[param] = data
	}
	ctx.Logger().Info(fmt.Sprintf("Calling handler \033[44;41m%s\033[0m", a.handler.Name()))
	obj, err := a.handler.Call(params, a.store)
	if err != nil {
		return nil, fmt.Errorf("handler %s: %w", a.handler.Name(), err)
	}

	return obj, nil
}

func (a *Actor) handleExecute(ctx actor.Context, execute *proto.Execute) {
	a.state = StateRunning
	tic := time.Now()
	obj, err := a.doExecute(ctx, execute)
	if err != nil {
		ctx.Send(execute.ReplyTo, &proto.Error{
			Message: err.Error(),
			Sender:  a.self,
		})
		return
	}

	if a.processLatency > 0 {
		processCost := time.Since(tic).Milliseconds()
		a.updateLatency(ctx, processCost)
	}

	ctx.Send(execute.ReplyTo, &proto.Flow{
		Actor:    a.self,
		ObjectID: obj.ID,
	})

	if a.lifecycle == LifeCycleOnce {
		a.state = StateFinished
	} else {
		a.state = StateReady
	}
}

func (a *Actor) updateLatency(ctx actor.Context, latency int64) {
	ctx.Send(system.ActorRef().Head(), &proto.ActorReady{
		ActorName:      a.Name(),
		GroupName:      a.handler.Name(),
		Sender:         a.self,
		ProcessLatency: latency,
	})
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.state = StateReady
		a.self = ctx.Self()
		a.updateLatency(ctx, a.processLatency)
	case *proto.ObjectRequest:
		a.handleObjectRequest(ctx, msg)
	case *proto.Execute:
		// proto.Execute may stuck proto.ObjectRequest, so we move it to another actor
		ctx.Forward(a.execActor)
	}
}

func (a *Actor) Props() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return a
	}, actor.WithOnInit(func(ctx actor.Context) {
		props := actor.PropsFromFunc(func(ctx actor.Context) {
			switch msg := ctx.Message().(type) {
			case *proto.Execute:
				a.handleExecute(ctx, msg)
			}
		})
		a.execActor = ctx.Spawn(props)
	}))
}

func NewActor(handler handlers.Function, initialProcessLatency time.Duration) *Actor {
	return NewNamedActor("<anonymous_actor>", handler, initialProcessLatency)
}

func NewNamedActor(name string, handler handlers.Function, initialProcessLatency time.Duration) *Actor {
	return &Actor{
		name:           name,
		state:          StateCreated,
		lifecycle:      LifeCycleLoop,
		store:          store.New(),
		handler:        handler,
		processLatency: initialProcessLatency.Milliseconds(),
	}
}
