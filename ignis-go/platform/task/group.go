package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
)

type CandidateStrategy interface {
	Select(candidates *actor.PIDSet) *actor.PID
}

type GroupedTaskHandler struct {
	baseHandler
	strategy   CandidateStrategy
	candidates *actor.PIDSet
	selected   *actor.PID
}

func (h *GroupedTaskHandler) Start(ctx actor.Context, replyTo *proto.ActorRef) error {
	if h.selected == nil {
		return errors.New("candidate not selected")
	}
	return nil
}

func (h *GroupedTaskHandler) Invoke(ctx actor.Context, param string, obj *proto.Flow) (ready bool, err error) {
	if h.selected == nil {
		h.selected = h.strategy.Select(h.candidates)
	}

	ctx.Send(h.selected, &proto.Invoke{
		SessionID: h.sessionId,
		Param:     param,
		Value:     obj,
	})
	h.deps.Remove(param)
	return h.ready(), nil
}
