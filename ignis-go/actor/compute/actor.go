package compute

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Actor struct {
	name     string
	store    *actor.PID
	executor *Executor
	sessions utils.Map[string, *actor.PID]
}

func (a *Actor) Name() string {
	return a.name
}

func (a *Actor) Deps() utils.Set[string] {
	return a.executor.Deps()
}

func (a *Actor) newSession(ctx actor.Context, sessionId string, successors []*proto.Successor) *actor.PID {
	ctx.Logger().Info("compute: create session", "actor", a.name, "session", sessionId)
	props := NewSession(sessionId, a.store, a.executor, a.Deps(), successors)
	session, _ := ctx.SpawnNamed(props, sessionId)
	a.sessions.Put(sessionId, session)
	return session
}

func (a *Actor) onCreateSession(ctx actor.Context, create *proto.CreateSession) {
	if _, ok := a.sessions[create.SessionID]; ok {
		ctx.Logger().Warn("compute: session exists, ignoring", "actor", a.name, "session", create.SessionID)
		return
	}
	a.sessions[create.SessionID] = a.newSession(ctx, create.SessionID, create.Successors)
}

func (a *Actor) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	session, ok := a.sessions.Get(invoke.SessionID)
	if !ok {
		ctx.Logger().Error("compute: session not found", "actor", a.name, "session", invoke.SessionID)
		return
	}

	flow := invoke.Value
	flow.Get(ctx).OnDone(func(obj proto.Object, err error) {
		if err != nil {
			ctx.Logger().Error("fetch failed",
				"a", a.name,
				"session", invoke.SessionID,
				"object-id", obj.GetID(),
			)
			ctx.Stop(session)
			return
		}
		ctx.Send(session, &SessionInvoke{Param: invoke.Param, Value: obj})
	})
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.CreateSession:
		a.onCreateSession(ctx, msg)
	case *proto.Invoke:
		a.onInvoke(ctx, msg)
	}
}

func NewActor(name string, handler functions.Function, store *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Actor{
			name:     name,
			store:    store,
			executor: NewExecutor(handler),
			sessions: make(utils.Map[string, *actor.PID]),
		}
	})
}
