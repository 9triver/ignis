package handlers

import (
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
	"github.com/asynkron/protoactor-go/actor"
)

type CandidateStrategy interface {
	Select(candidates *actor.PIDSet) *actor.PID
}

type GroupedTaskHandler struct {
	baseTaskHandler
	strategy   CandidateStrategy
	candidates *actor.PIDSet
	selected   *actor.PID
}

func (h *GroupedTaskHandler) InvokeAll(ctx actor.Context, successors []*messages.Successor) error {
	if h.selected == nil {
		return errors.New("candidate not selected")
	}

	ctx.Send(h.selected, &messages.CreateSession{
		SessionID:  h.sessionId,
		Successors: successors,
	})
	return nil
}

func (h *GroupedTaskHandler) InvokeOne(ctx actor.Context, _ []*messages.Successor, param string, obj *proto.Flow) (ready bool, err error) {
	if h.selected == nil {
		h.selected = h.strategy.Select(h.candidates)
	}

	ctx.Send(h.selected, &proto.Invoke{
		SessionID: h.sessionId,
		Param:     param,
		Value:     obj,
	})
	h.inDegrees.Remove(param)
	return h.ready(), nil
}

func (h *GroupedTaskHandler) InvokeEmpty(ctx actor.Context, successors []*messages.Successor) (ready bool, err error) {
	if h.selected == nil {
		h.selected = h.strategy.Select(h.candidates)
	}

	return h.ready(), nil
}
