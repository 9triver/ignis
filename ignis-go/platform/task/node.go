package task

import (
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
}

func (rt *Runtime) Start(ctx actor.Context) error {
	return rt.handler.Start(ctx, rt.replyTo)
}

func (rt *Runtime) Invoke(ctx actor.Context, param string, value *proto.Flow) (err error) {
	_, err = rt.handler.Invoke(ctx, param, value)
	return
}
