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

func (node *Node) Runtime(sessionId string, store *actor.PID, replyTo *proto.ActorRef) *Runtime {
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
	replyTo *proto.ActorRef
	handler Handler
}

func (rt *Runtime) Invoke(ctx actor.Context, invoke *proto.Invoke) error {
	ready, err := rt.handler.Invoke(ctx, invoke)
	if err != nil {
		return err
	}

	if ready {
		return rt.handler.Start(ctx, rt.replyTo)
	}
	return nil
}
