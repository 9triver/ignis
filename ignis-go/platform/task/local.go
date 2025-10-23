package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/objects"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type LocalTaskHandler struct {
	baseHandler
	localFunc functions.Function
	params    map[string]*proto.Flow
}

func (h *LocalTaskHandler) Start(ctx actor.Context, replyTo string) error {
	futures := make(map[string]utils.Future[objects.Interface])
	for param, flow := range h.params {
		futures[param] = store.GetObject(ctx, h.store, flow)
	}

	invoke := make(map[string]objects.Interface)
	for param, fut := range futures {
		obj, err := fut.Result()
		if err != nil {
			return err
		}
		invoke[param] = obj
	}

	obj, err := h.localFunc.Call(invoke)
	if err != nil {
		return err
	}

	ctx.Send(h.store, &store.SaveObject{
		Value: obj,
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			ctx.Send(h.store, &proto.InvokeResponse{
				Target:    replyTo,
				SessionID: h.sessionId,
				Result:    ref,
			})
		},
	})

	return nil
}

func (h *LocalTaskHandler) Invoke(_ actor.Context, param string, value *proto.Flow) (bool, error) {
	if !h.deps.Contains(param) {
		return false, errors.Format("received unexpected param %s", param)
	}

	h.params[param] = value
	h.deps.Remove(param)

	return h.ready(), nil
}

func HandlerFromFunction(sessionId string, store *actor.PID, f functions.Function) *LocalTaskHandler {
	return &LocalTaskHandler{
		baseHandler: makeBaseHandler(sessionId, store, f.Params()),
		localFunc:   f,
		params:      make(map[string]*proto.Flow),
	}
}

func ProducerFromFunction(f functions.Function) HandlerProducer {
	return func(sessionId string, store *actor.PID) Handler {
		return HandlerFromFunction(sessionId, store, f)
	}
}

type ActorTaskHandler struct {
	baseHandler
	pid *actor.PID
}

func (h *ActorTaskHandler) Start(ctx actor.Context, replyTo string) error {
	ctx.Send(h.pid, &proto.InvokeStart{
		SessionID: h.sessionId,
		ReplyTo:   replyTo,
	})
	return nil
}

func (h *ActorTaskHandler) Invoke(ctx actor.Context, param string, value *proto.Flow) (bool, error) {
	ctx.Send(h.pid, &proto.Invoke{
		SessionID: h.sessionId,
		Param:     param,
		Value:     value,
	})
	h.deps.Remove(param)
	return h.ready(), nil
}

func HandlerFromPID(sessionId string, store *actor.PID, params []string, pid *actor.PID) *ActorTaskHandler {
	return &ActorTaskHandler{
		baseHandler: makeBaseHandler(sessionId, store, params),
		pid:         pid,
	}
}

func ProducerFromPID(params []string, pid *actor.PID) HandlerProducer {
	return func(sessionId string, store *actor.PID) Handler {
		return HandlerFromPID(sessionId, store, params, pid)
	}
}
