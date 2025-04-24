package dag

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/dag/handlers"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type TaskHandler interface {
	InvokeAll(ctx actor.Context, successors []*messages.Successor) error
	InvokeOne(ctx actor.Context, successors []*messages.Successor, param string, obj *proto.Flow) (ready bool, err error)
	InvokeEmpty(ctx actor.Context, successors []*messages.Successor) (ready bool, err error)
}

type TaskHandlerProducer func(sessionId string, store *actor.PID) TaskHandler

type TaskNode struct {
	baseGraphNode
	inputs          utils.Set[string]   // dependencies of the node
	handlerProducer TaskHandlerProducer // handler producer of specified task
}

func (node *TaskNode) Type() NodeType {
	return TaskNodeType
}

func (node *TaskNode) Inputs() utils.Set[string] {
	return node.inputs
}

func (node *TaskNode) newRuntime(sessionId string, store *actor.PID) *TaskNodeRuntime {
	return &TaskNodeRuntime{
		baseNodeRuntime: makeBaseNodeRuntime(node, sessionId, store),
		handler:         node.handlerProducer(sessionId, store),
	}
}

func (node *TaskNode) Props(sessionId string, store *actor.PID) *actor.Props {
	rt := node.newRuntime(sessionId, store)
	return actor.PropsFromProducer(func() actor.Actor {
		return rt
	})
}

func NewTaskNode(id string, inputs utils.Set[string], handler TaskHandlerProducer) *TaskNode {
	return &TaskNode{
		baseGraphNode: baseGraphNode{
			id: id,
		},
		inputs:          inputs.Copy(),
		handlerProducer: handler,
	}
}

type TaskNodeRuntime struct {
	baseNodeRuntime[*TaskNode]
	handler TaskHandler // task handler, produced from task node since it may have states
}

func (rt *TaskNodeRuntime) onTaskError(ctx actor.Context, err error) {
	ctx.Logger().Error("invoke error",
		"node", rt.node.id,
		"message", err.Error(),
		"session", rt.sessionId,
	)
	println(errors.Stacktrace(err))
	rt.closeWith(ctx, err)
}

func (rt *TaskNodeRuntime) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	ctx.Logger().Info("task: receive invoke",
		"id", rt.node.id,
		"param", invoke.Param,
		"session", rt.sessionId,
	)

	ready, err := rt.handler.InvokeOne(ctx, rt.successors, invoke.Param, invoke.Value)
	if err != nil {
		rt.onTaskError(ctx, err)
		return
	}
	if ready {
		if err := rt.handler.InvokeAll(ctx, rt.successors); err != nil {
			rt.onTaskError(ctx, err)
		} else {
			rt.closeWith(ctx, nil)
		}
	}
}

func (rt *TaskNodeRuntime) onInvokeEmpty(ctx actor.Context) {
	ctx.Logger().Info("task: receive invoke empty",
		"id", rt.node.id,
		"session", rt.sessionId,
	)

	ready, err := rt.handler.InvokeEmpty(ctx, rt.successors)
	if err != nil {
		rt.onTaskError(ctx, err)
	}
	if ready {
		if err := rt.handler.InvokeAll(ctx, rt.successors); err != nil {
			rt.onTaskError(ctx, err)
		} else {
			rt.closeWith(ctx, nil)
		}
	}
}

func (rt *TaskNodeRuntime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
	case *messages.Successor:
		rt.onAddEdge(ctx, msg)
	case *proto.Invoke:
		rt.onInvoke(ctx, msg)
	case *proto.InvokeEmpty:
		rt.onInvokeEmpty(ctx)
	}
}

var (
	_ actor.Actor = (*TaskNodeRuntime)(nil)
	_ TaskHandler = (*handlers.LocalTaskHandler)(nil)
	_ TaskHandler = (*handlers.RemoteTaskHandler)(nil)
)

func TaskNodeFromFunction(id string, f functions.Function) *TaskNode {
	return NewTaskNode(id, f.Params(), func(sessionId string, store *actor.PID) TaskHandler {
		return handlers.FromFunction(sessionId, store, f)
	})
}

func TaskNodeFromPID(id string, params utils.Set[string], pid *actor.PID) *TaskNode {
	return NewTaskNode(id, params, func(sessionId string, store *actor.PID) TaskHandler {
		return handlers.FromPID(sessionId, store, params, pid)
	})
}
