package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
)

type Node struct {
	id              string
	inputs          []string        // dependencies of the node
	handlerProducer HandlerProducer // handler producer of specified task
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
	return NewNode(id, f.Params(), ProducerFromFunction(f))
}

func NodeFromPID(id string, params []string, pid *actor.PID) *Node {
	return NewNode(id, params, ProducerFromPID(params, pid))
}

func NodeFromActorGroup(id string, params []string, group *ActorGroup) *Node {
	return NewNode(id, params, ProducerFromActorGroup(params, group))
}

// Runtime serves as a bridge between graph definition and graph execution actors.
// A node runtime holds an actor context and is accessible to node definition (referred by `P`)
type Runtime struct {
	sessionId string
	replyTo   *proto.ActorRef
	node      *Node
	handler   Handler
}

func (rt *Runtime) closeWith(ctx actor.Context, err error) {
	ctx.Logger().Info("node: actor terminating",
		"name", rt.node.id,
		"session", rt.sessionId,
		"error", err,
	)
	if err != nil {
		println(errors.Stacktrace(err))
	}
	ctx.Stop(ctx.Self())
}

func (rt *Runtime) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	ctx.Logger().Info("task: receive invoke",
		"id", rt.node.id,
		"param", invoke.Param,
		"session", rt.sessionId,
	)
	ready, err := rt.handler.Invoke(ctx, invoke)
	if err != nil {
		rt.closeWith(ctx, err)
		return
	}

	if ready {
		err := rt.handler.Start(ctx, rt.replyTo)
		rt.closeWith(ctx, err)
	}
}

func (rt *Runtime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.Invoke:
		rt.onInvoke(ctx, msg)
	}
}
