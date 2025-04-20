package compute

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Session struct {
	id           string
	store        *actor.PID
	executor     *Executor
	deps         utils.Set[string]
	params       map[string]proto.Object
	successors   []*proto.Successor
	successorSet bool
}

type SessionInvoke struct {
	Param string
	Value proto.Object
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) doInvoke(ctx actor.Context) {
	ctx.Logger().Info("invoke started", "session", s.id)
	exec := &ExecInput{
		Context:   ctx,
		SessionID: s.id,
		Params:    s.params,
		OnDone: func(obj proto.Object, err error, duration time.Duration) {
			if err != nil {
				s.sendError(ctx, err)
				return
			}

			save := &store.SaveObject{
				Value: obj,
				Callback: func(ctx actor.Context, ref *proto.Flow) {
					s.sendResult(ctx, ref)
				},
			}
			ctx.Send(s.store, save)
		},
	}
	s.executor.Requests() <- exec
}

func (s *Session) onSuccessors(ctx actor.Context, successors *proto.AppendSuccessors) {
	if s.successorSet {
		return
	}
	ctx.Logger().Info("session updates successors", "session", s.id)
	s.successors = successors.Successors
	s.successorSet = true
	if s.ready() {
		s.doInvoke(ctx)
	}
}

func (s *Session) onInvoke(ctx actor.Context, invoke *SessionInvoke) {
	if invoke.Value == nil {
		return
	}

	ctx.Logger().Info("session invokes", "session", s.id, "param", invoke.Param)
	s.enqueue(invoke.Param, invoke.Value)
	if s.ready() {
		s.doInvoke(ctx)
	}
}

func (s *Session) enqueue(param string, obj proto.Object) {
	if !s.deps.Contains(param) {
		return
	}
	s.params[param] = obj
	s.deps.Remove(param)
}

func (s *Session) ready() bool {
	return s.successorSet && s.deps.Empty()
}

func (s *Session) sendError(ctx actor.Context, err error) {
	for _, succ := range s.successors {
		ctx.Send(succ.PID, &proto.Invoke{
			SessionID: s.id,
			Param:     succ.Param,
			Error:     err.Error(),
		})
	}
}

func (s *Session) sendResult(ctx actor.Context, obj *proto.Flow) {
	for _, succ := range s.successors {
		ctx.Send(succ.PID, &proto.Invoke{
			SessionID: s.id,
			Param:     succ.Param,
			Value:     obj,
		})
	}
}

func (s *Session) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.AppendSuccessors:
		s.onSuccessors(ctx, msg)
	case *SessionInvoke:
		s.onInvoke(ctx, msg)
	case *actor.Stop:
		s.executor.Close()
	default:
		ctx.Logger().Error("unknown message", "message", msg)
	}
}

func NewSession(
	id string,
	store *actor.PID,
	executor *Executor,
	deps utils.Set[string],
) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Session{
			id:         id,
			store:      store,
			executor:   executor,
			deps:       deps.Copy(),
			params:     make(map[string]proto.Object),
			successors: nil,
		}
	})
}
