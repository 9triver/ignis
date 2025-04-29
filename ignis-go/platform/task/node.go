package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/proto"
)

type Node struct {
	id              string
	inputs          []string        // dependencies of the node
	handlerProducer HandlerProducer // handler producer of specified task
}

func (node *Node) ID() string {
	return node.id
}

func (node *Node) Inputs() []string {
	return node.inputs
}

func (node *Node) Props(sessionId string, store *actor.PID, replyTo *proto.ActorRef) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Runtime{
			sessionId: sessionId,
			replyTo:   replyTo,
			node:      node,
			handler:   node.handlerProducer(sessionId, store),
		}
	})
}

func NewNode(id string, inputs []string, handler HandlerProducer) *Node {
	return &Node{
		id:              id,
		inputs:          inputs,
		handlerProducer: handler,
	}
}

func NodeFromFunction(id string, f functions.Function) *Node {
	return NewNode(id, f.Params(), func(sessionId string, store *actor.PID) Handler {
		return HandlerFromFunction(sessionId, store, f)
	})
}

func NodeFromPID(id string, params []string, pid *actor.PID) *Node {
	return NewNode(id, params, func(sessionId string, store *actor.PID) Handler {
		return HandlerFromPID(sessionId, store, params, pid)
	})
}
