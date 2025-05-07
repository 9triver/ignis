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
	cond sync.Cond
	ch   chan *ActorInfo
}

func (g *ActorGroup) SelectChan() <-chan *ActorInfo {
	return g.ch
}

func (g *ActorGroup) Run() {
	defer close(g.ch)
	for {
		g.cond.L.Lock()
		for g.pq.Len() == 0 {
			g.cond.Wait()
		}
		g.cond.L.Unlock()

		selected := g.pq.Pop()
		g.ch <- selected
	}
}

func (g *ActorGroup) Push(info *ActorInfo) {
	g.cond.L.Lock()
	g.pq.Push(info)
	g.cond.L.Unlock()

	g.cond.Broadcast()
}

func (g *ActorGroup) TaskDone(info *ActorInfo) {
	g.Push(info)
}

func NewGroup(name string, candidates ...*ActorInfo) *ActorGroup {
	pq := utils.MakePriorityQueue(func(i, j *ActorInfo) bool {
		return i.LinkLatency*2+i.CalcLatency < j.LinkLatency*2+j.CalcLatency
	})
	for _, info := range candidates {
		pq.Push(info)
	}

	return &ActorGroup{
		name: name,
		cond: *sync.NewCond(&sync.Mutex{}),
		ch:   make(chan *ActorInfo),
		pq:   pq,
	}
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

	ctx.Send(h.store, &proto.StartRemote{
		Info: h.selected,
		Start: &proto.InvokeStart{
			SessionID: h.sessionId,
			ReplyTo:   replyTo,
		},
	})
	return nil
}

func (h *GroupedTaskHandler) Invoke(ctx actor.Context, invoke *proto.Invoke) (ready bool, err error) {
	if h.selected == nil {
		h.selected = <-h.group.SelectChan()
	}

	h.deps.Remove(invoke.Param)
	ctx.Send(h.store, &proto.InvokeRemote{
		Target: h.selected.Ref,
		Invoke: invoke,
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
