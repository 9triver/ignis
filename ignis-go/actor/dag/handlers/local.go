package handlers

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type baseTaskHandler struct {
	sessionId string
	store     *actor.PID
	inDegrees utils.Set[string]
}

func makeBaseTaskHandler(sessionId string, store *actor.PID, inDegrees utils.Set[string]) baseTaskHandler {
	return baseTaskHandler{
		sessionId: sessionId,
		store:     store,
		inDegrees: inDegrees.Copy(),
	}
}

func (h *baseTaskHandler) ready() bool {
	return h.inDegrees.Empty()
}

type LocalTaskHandler struct {
	baseTaskHandler
	localFunc functions.Function
	params    utils.Map[string, *proto.Flow]
}

func (h *LocalTaskHandler) InvokeAll(ctx actor.Context, successors []*proto.Successor) error {
	futures := make(map[string]utils.Future[proto.Object])
	for param, flow := range h.params {
		futures[param] = flow.Get(ctx)
	}

	invoke := make(map[string]proto.Object)
	for param, fut := range futures {
		obj, err := fut.Result()
		if err != nil {
			return err
		}
		invoke[param] = obj
	}

	obj, err := h.localFunc.Call(ctx, h.sessionId, invoke)
	if err != nil {
		return err
	}

	ctx.Send(h.store, &store.SaveObject{
		Value: obj,
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			for _, succ := range successors {
				ctx.Send(succ.PID, &proto.Invoke{
					SessionID: h.sessionId,
					Param:     succ.Param,
					Value:     ref,
				})
			}
		},
	})

	return nil
}

func (h *LocalTaskHandler) InvokeOne(
	_ actor.Context,
	_ []*proto.Successor,
	param string,
	obj *proto.Flow,
) (done bool, err error) {
	if !h.inDegrees.Contains(param) {
		return false, errors.Format("received unexpected param %s", param)
	}

	h.params.Put(param, obj)
	h.inDegrees.Remove(param)

	return h.ready(), nil
}

func (h *LocalTaskHandler) InvokeEmpty(ctx actor.Context, successors []*proto.Successor) (ready bool, err error) {
	return h.ready(), nil
}

func FromFunction(sessionId string, store *actor.PID, f functions.Function) *LocalTaskHandler {
	return &LocalTaskHandler{
		baseTaskHandler: makeBaseTaskHandler(sessionId, store, f.Params()),
		localFunc:       f,
		params:          utils.MakeMap[string, *proto.Flow](),
	}
}
