package handlers

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type RemoteTaskHandler struct {
	baseTaskHandler
	pid *actor.PID
}

func (h *RemoteTaskHandler) InvokeAll(ctx actor.Context, successors []*proto.Successor) error {
	ctx.Send(h.pid, &proto.CreateSession{
		SessionID:  h.sessionId,
		Successors: successors,
	})
	return nil
}

func (h *RemoteTaskHandler) InvokeOne(
	ctx actor.Context,
	_ []*proto.Successor, // successors are sent after handler is ready
	param string,
	obj *proto.Flow,
) (done bool, err error) {
	ctx.Send(h.pid, &proto.Invoke{
		SessionID: h.sessionId,
		Param:     param,
		Value:     obj,
	})
	h.inDegrees.Remove(param)
	return h.ready(), nil
}

func (h *RemoteTaskHandler) InvokeEmpty(ctx actor.Context, successors []*proto.Successor) (ready bool, err error) {
	return h.ready(), nil
}

func FromPID(sessionId string, store *actor.PID, params utils.Set[string], pid *actor.PID) *RemoteTaskHandler {
	return &RemoteTaskHandler{
		baseTaskHandler: makeBaseTaskHandler(sessionId, store, params),
		pid:             pid,
	}
}
