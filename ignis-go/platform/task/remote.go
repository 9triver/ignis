package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
)

type RemoteTaskHandler struct {
	baseHandler
	pid *actor.PID
}

func (h *RemoteTaskHandler) Start(ctx actor.Context, replyTo *proto.ActorRef) error {
	ctx.Send(h.pid, &proto.InvokeStart{
		SessionID: h.sessionId,
		ReplyTo:   replyTo,
	})
	return nil
}

func (h *RemoteTaskHandler) Invoke(ctx actor.Context, param string, obj *proto.Flow) (bool, error) {
	ctx.Send(h.pid, &proto.Invoke{
		SessionID: h.sessionId,
		Param:     param,
		Value:     obj,
	})
	h.deps.Remove(param)
	return h.ready(), nil
}

func HandlerFromPID(sessionId string, store *actor.PID, params []string, pid *actor.PID) *RemoteTaskHandler {
	return &RemoteTaskHandler{
		baseHandler: makeBaseHandler(sessionId, store, params),
		pid:         pid,
	}
}
