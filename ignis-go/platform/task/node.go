package task

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/proto"
)

type Node struct {
	id       string
	inputs   []string        // dependencies of the node
	producer HandlerProducer // handler producer of specified task
}

func (node *Node) Runtime(sessionId string, store *actor.PID, replyTo string) *Runtime {
	return &Runtime{
		replyTo: replyTo,
		handler: node.producer(sessionId, store),
		cond:    sync.NewCond(&sync.Mutex{}),
	}
}

func NewNode(id string, inputs []string, handler HandlerProducer) *Node {
	return &Node{
		id:       id,
		inputs:   inputs,
		producer: handler,
	}
}

func NodeFromFunction(id string, f functions.Function) *Node {
	return NewNode(id, f.Params(), ProducerFromFunction(f))
}

func NodeFromPID(id string, params []string, pid *actor.PID) *Node {
	return NewNode(id, params, ProducerFromPID(params, pid))
}

func NodeFromActorGroup(id string, params []string, group *ActorGroup) *Node {
	return NewNode(id, params, ProducerFromActorGroup(params, group))
}

type Runtime struct {
	replyTo string
	handler Handler
	cond    *sync.Cond
}

func (rt *Runtime) Start(ctx actor.Context) error {
	rt.cond.L.Lock()
	for !rt.handler.Ready() {
		rt.cond.Wait()
	}
	rt.cond.L.Unlock()

	return rt.handler.Start(ctx, rt.replyTo)
}

func (rt *Runtime) Invoke(ctx actor.Context, param string, value *proto.Flow) (err error) {
	ready, err := rt.handler.Invoke(ctx, param, value)
	if err != nil {
		return err
	}

	if ready {
		rt.cond.L.Lock()
		rt.cond.Signal()
		rt.cond.L.Unlock()
	}

	return
}
