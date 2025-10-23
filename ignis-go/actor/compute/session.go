package compute

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Session struct {
	id       string
	store    *actor.PID
	executor *Executor
	params   map[string]objects.Interface
	start    *SessionStart
	deps     utils.Set[string]
	link     time.Duration
}

type SessionInvoke struct {
	Param string
	Link  time.Duration
	Value objects.Interface
}

type SessionStart struct {
	Info    *proto.ActorInfo
	ReplyTo string
}

func (s *Session) onInvoke(ctx actor.Context, invoke *SessionInvoke) {
	ctx.Logger().Info("session: receive invoke", "session", s.id, "param", invoke.Param)
	s.params[invoke.Param] = invoke.Value
	s.deps.Remove(invoke.Param)
	s.link += invoke.Link
	if s.deps.Empty() && s.start != nil {
		s.doInvoke(ctx)
	}
}

func (s *Session) onStart(ctx actor.Context, start *SessionStart) {
	ctx.Logger().Info("session: start execution", "session", s.id, "replyTo", start.ReplyTo)
	s.start = start
	if s.deps.Empty() {
		s.doInvoke(ctx)
	}
}

func (s *Session) onError(ctx actor.Context, err error) {
	ctx.Logger().Error("session: execution failed", "session", s.id, "err", err)

	ctx.Send(s.store, &proto.InvokeResponse{
		Target:    s.start.ReplyTo,
		SessionID: s.id,
		Info:      s.start.Info,
		Error:     err.Error(),
	})
}

func (s *Session) onComplete(ctx actor.Context, obj objects.Interface, duration time.Duration) {
	ctx.Logger().Info("session: execution complete", "session", s.id, "duration", duration)

	info := s.start.Info
	if info != nil {
		info.CalcLatency = (info.CalcLatency + int64(duration)) / 2
		info.LinkLatency = (info.LinkLatency + int64(s.link)) / 2
	}

	save := &store.SaveObject{
		Value: obj,
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			ctx.Send(s.store, &proto.InvokeResponse{
				Target:    s.start.ReplyTo,
				SessionID: s.id,
				Info:      info,
				Result:    ref,
			})
		},
	}
	ctx.Send(s.store, save)
}

func (s *Session) doInvoke(ctx actor.Context) {
	ctx.Logger().Info("session: invoke execution", "session", s.id, "params", s.params)
	exec := &ExecInput{
		Context:   ctx,
		SessionID: s.id,
		Params:    s.params,
		Timed:     s.start.Info != nil,
		OnDone: func(obj objects.Interface, err error, duration time.Duration) {
			if err != nil {
				s.onError(ctx, err)
				return
			}
			s.onComplete(ctx, obj, duration)
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
			params:   make(map[string]objects.Interface),
			deps:     utils.MakeSetFromSlice(executor.Deps()),
		}
	})
}
