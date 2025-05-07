package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type LocalTaskHandler struct {
	baseHandler
	localFunc functions.Function
	params    map[string]*proto.Flow
}

func (h *LocalTaskHandler) Start(ctx actor.Context, replyTo *proto.ActorRef) error {
	futures := make(map[string]utils.Future[messages.Object])
	for param, flow := range h.params {
		futures[param] = store.GetObject(ctx, h.store, flow)
	}

	invoke := make(map[string]messages.Object)
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

	ctx.Send(h.store, &messages.SaveObject{
		Value: obj,
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			ctx.Send(h.store, &proto.InvokeRemote{
				Target: replyTo,
				Invoke: &proto.Invoke{
					SessionID: h.sessionId,
					Value:     ref,
				},
			})
		},
	})

	return nil
}

func (h *LocalTaskHandler) Invoke(_ actor.Context, invoke *proto.Invoke) (done bool, err error) {
	param, obj := invoke.Param, invoke.Value
	if !h.deps.Contains(param) {
		return false, errors.Format("received unexpected param %s", param)
	}

	h.params[param] = obj
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

func (h *ActorTaskHandler) Start(ctx actor.Context, replyTo *proto.ActorRef) error {
	ctx.Send(h.pid, &proto.InvokeStart{
		SessionID: h.sessionId,
		ReplyTo:   replyTo,
	})
	return nil
}

func (h *ActorTaskHandler) Invoke(ctx actor.Context, invoke *proto.Invoke) (bool, error) {
	invoke.SessionID = h.sessionId
	ctx.Send(h.pid, invoke)
	h.deps.Remove(invoke.Param)
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
