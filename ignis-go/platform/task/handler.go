package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Handler interface {
	Start(ctx actor.Context, replyTo *proto.ActorRef) error
	Invoke(ctx actor.Context, param string, obj *proto.Flow) (ready bool, err error)
}

type baseHandler struct {
	sessionId string
	store     *actor.PID
	deps      utils.Set[string]
}

func makeBaseHandler(sessionId string, store *actor.PID, inDegrees []string) baseHandler {
	return baseHandler{
		sessionId: sessionId,
		store:     store,
		deps:      utils.MakeSetFromSlice(inDegrees),
	}
}

func (h *baseHandler) ready() bool {
	return h.deps.Empty()
}

var (
	_ Handler = (*LocalTaskHandler)(nil)
	_ Handler = (*RemoteTaskHandler)(nil)
	_ Handler = (*GroupedTaskHandler)(nil)
)

type HandlerProducer func(sessionId string, store *actor.PID) Handler
