package compute

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Session struct {
	id       string
	store    *actor.PID
	executor *Executor
	params   map[string]messages.Object
	replyTo  *proto.ActorRef
	deps     utils.Set[string]
}

type SessionInvoke struct {
	Param string
	Value messages.Object
}

type SessionStart struct {
	ReplyTo *proto.ActorRef
}

func (s *Session) onInvoke(ctx actor.Context, invoke *SessionInvoke) {
	ctx.Logger().Info("session: receive invoke", "session", s.id, "param", invoke.Param)
	s.params[invoke.Param] = invoke.Value
	s.deps.Remove(invoke.Param)

	if s.deps.Empty() && s.replyTo != nil {
		s.doInvoke(ctx)
	}
}

func (s *Session) onStart(ctx actor.Context, start *SessionStart) {
	ctx.Logger().Info("session: start execution", "session", s.id, "replyTo", start.ReplyTo.ID)
	s.replyTo = start.ReplyTo
	if s.deps.Empty() && s.replyTo != nil {
		s.doInvoke(ctx)
	}
}

func (s *Session) doInvoke(ctx actor.Context) {
	exec := &ExecInput{
		Context:   ctx,
		SessionID: s.id,
		Params:    s.params,
		OnDone: func(obj messages.Object, err error, duration time.Duration) {
			if err != nil {
				ctx.Send(s.store, &proto.InvokeRemote{
					Target: s.replyTo,
					Invoke: &proto.Invoke{
						SessionID: s.id,
						Error:     err.Error(),
					},
				})
				return
			}

			save := &messages.SaveObject{
				Value: obj,
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					ctx.Send(s.store, &proto.InvokeRemote{
						Target: s.replyTo,
						Invoke: &proto.Invoke{
							SessionID: s.id,
							Value:     ref,
						},
					})
				},
			}
			ctx.Send(s.store, save)
		},
	}
	s.executor.Requests() <- exec
}

func (s *Session) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *SessionInvoke:
		s.onInvoke(ctx, msg)
	case *SessionStart:
		s.onStart(ctx, msg)
	case *actor.Stop:
		s.executor.Close()
	}
}

func NewSession(
	id string,
	store *actor.PID,
	executor *Executor,
) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Session{
			id:       id,
			store:    store,
			executor: executor,
			params:   make(map[string]messages.Object),
			deps:     utils.MakeSetFromSlice(executor.Deps()),
		}
	})
}
