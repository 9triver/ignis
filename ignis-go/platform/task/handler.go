package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type Handler interface {
	SessionID() string
	Start(ctx actor.Context, replyTo *proto.ActorRef) error
	Invoke(ctx actor.Context, param string, value *proto.Flow) (ready bool, err error)
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

func (h *baseHandler) SessionID() string {
	return h.sessionId
}

var (
	_ Handler = (*LocalTaskHandler)(nil)
	_ Handler = (*ActorTaskHandler)(nil)
	_ Handler = (*GroupedTaskHandler)(nil)
)

type HandlerProducer func(sessionId string, store *actor.PID) Handler
