package compute

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
)

type Actor struct {
	name     string
	store    *actor.PID
	executor *Executor
	sessions map[string]*actor.PID
}

func (a *Actor) newSession(ctx actor.Context, sessionId string) *actor.PID {
	ctx.Logger().Info("compute: create session", "actor", a.name, "session", sessionId)
	props := NewSession(sessionId, a.store, a.executor)
	session, _ := ctx.SpawnNamed(props, "session."+sessionId)
	a.sessions[sessionId] = session
	return session
}

func (a *Actor) onInvokeStart(ctx actor.Context, sr *proto.StartRemote) {
	start := sr.Start
	session, ok := a.sessions[start.SessionID]
	if !ok {
		session = a.newSession(ctx, start.SessionID)
		a.sessions[start.SessionID] = session
	}
	ctx.Send(session, &SessionStart{
		Info:    sr.Info,
		ReplyTo: start.ReplyTo,
	})
}

func (a *Actor) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	ctx.Logger().Info("compute: receive invoke",
		"actor", a.name,
		"session", invoke.SessionID,
		"param", invoke.Param,
		"value", invoke.Value.ObjectID,
	)
	session, ok := a.sessions[invoke.SessionID]
	if !ok {
		session = a.newSession(ctx, invoke.SessionID)
		a.sessions[invoke.SessionID] = session
	}

	store.GetObject(ctx, a.store, invoke.Value).OnDone(func(obj messages.Object, duration time.Duration, err error) {
		if err != nil {
			ctx.Logger().Error("compute: object fetch failed",
				"actor", a.name,
				"session", invoke.SessionID,
				"object-id", obj.GetID(),
			)
			ctx.Stop(session)
			return
		}
		ctx.Send(session, &SessionInvoke{Link: duration, Param: invoke.Param, Value: obj})
	})
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.StartRemote:
		a.onInvokeStart(ctx, msg)
	case *proto.InvokeRemote:
		a.onInvoke(ctx, msg.Invoke)
	default:
		ctx.Logger().Error("compute: unknown message", "message", msg)
	}
}

func NewActor(name string, handler functions.Function, store *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Actor{
			name:     name,
			store:    store,
			executor: NewExecutor(handler),
			sessions: make(map[string]*actor.PID),
		}
	})
}
