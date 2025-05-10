package task

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type ActorInfo = proto.ActorInfo

type ActorGroup struct {
	name string
	pq   utils.PQueue[*ActorInfo]
	cond *sync.Cond
}

func (g *ActorGroup) Select() *ActorInfo {
	g.cond.L.Lock()
	defer g.cond.L.Unlock()

	for g.pq.Len() == 0 {
		g.cond.Wait()
	}

	return g.pq.Pop()
}

func (g *ActorGroup) Push(info *ActorInfo) {
	g.cond.L.Lock()
	g.pq.Push(info)
	g.cond.L.Unlock()

	g.cond.Signal()
}

func GroupWithLessFunc(name string, lessFunc utils.LessFunc[*ActorInfo], candidates ...*ActorInfo) *ActorGroup {
	return &ActorGroup{
		name: name,
		pq:   utils.MakePriorityQueue(lessFunc, candidates...),
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func NewGroup(name string, candidates ...*ActorInfo) *ActorGroup {
	return GroupWithLessFunc(name, func(i, j *ActorInfo) bool {
		return i.LinkLatency*2+i.CalcLatency < j.LinkLatency*2+j.CalcLatency
	}, candidates...)
}

type GroupedTaskHandler struct {
	baseHandler
	group    *ActorGroup
	selected *ActorInfo
}

func (h *GroupedTaskHandler) Start(ctx actor.Context, replyTo *proto.ActorRef) error {
	if h.selected == nil {
		return errors.New("no candidate actor selected")
	}

	ctx.Send(h.store, &proto.InvokeStart{
		Info:      h.selected,
		SessionID: h.sessionId,
		ReplyTo:   replyTo,
	})
	return nil
}

func (h *GroupedTaskHandler) Invoke(ctx actor.Context, param string, value *proto.Flow) (bool, error) {
	if h.selected == nil {
		h.selected = h.group.Select()
	}

	h.deps.Remove(param)
	ctx.Send(h.store, &proto.Invoke{
		Target:    h.selected.Ref,
		SessionID: h.sessionId,
		Param:     param,
		Value:     value,
	})
	return h.ready(), nil
}

func HandlerFromActorGroup(sessionId string, store *actor.PID, params []string, group *ActorGroup) *GroupedTaskHandler {
	return &GroupedTaskHandler{
		baseHandler: makeBaseHandler(sessionId, store, params),
		group:       group,
		selected:    nil,
	}
}

func ProducerFromActorGroup(params []string, group *ActorGroup) HandlerProducer {
	return func(sessionId string, store *actor.PID) Handler {
		return HandlerFromActorGroup(sessionId, store, params, group)
	}
}
