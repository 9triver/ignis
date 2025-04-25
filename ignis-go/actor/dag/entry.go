package dag

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
)

type EntryNodeRuntime struct {
	baseNodeRuntime[*EntryNode]
}

func (rt *EntryNodeRuntime) onInvoke(ctx actor.Context) {
	ctx.Logger().Info("entry: receive invoke",
		"id", rt.node.id,
		"session", rt.sessionId,
	)
	for _, edge := range rt.successors {
		ctx.Send(edge.PID, &proto.Invoke{
			Param: edge.Param,
			Value: rt.node.Value(),
		})
	}
}

func (rt *EntryNodeRuntime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *messages.Successor:
		rt.onAddEdge(ctx, msg)
	case *proto.InvokeEmpty, *proto.Invoke:
		rt.onInvoke(ctx)
	}
}

type EntryNode struct {
	baseGraphNode
	value *proto.Flow
}

func (node *EntryNode) Value() *proto.Flow {
	return node.value
}

func (node *EntryNode) Type() NodeType {
	return EntryNodeType
}

func (node *EntryNode) newRuntime(sessionId string, store *actor.PID) *EntryNodeRuntime {
	return &EntryNodeRuntime{
		baseNodeRuntime: makeBaseNodeRuntime(node, sessionId, store),
	}
}

func (node *EntryNode) Props(sessionId string, store *actor.PID) *actor.Props {
	rt := node.newRuntime(sessionId, store)
	return actor.PropsFromProducer(func() actor.Actor {
		return rt
	})
}

func NewEntryNode(id string, value *proto.Flow) *EntryNode {
	return &EntryNode{
		baseGraphNode: baseGraphNode{
			id: id,
		},
		value: value,
	}
}

type ExitNodeRuntime struct {
	baseNodeRuntime[*ExitNode]
	results map[string]*proto.Flow
}

func (rt *ExitNodeRuntime) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	ctx.Logger().Info("exit: receive result",
		"exit", rt.node.id,
		"param", invoke.Param,
		"object", invoke.Value.ObjectID,
		"session", rt.sessionId,
	)
	rt.results[invoke.Param] = invoke.Value
}

func (rt *ExitNodeRuntime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *messages.Successor:
		rt.onAddEdge(ctx, msg)
	case *proto.Invoke:
		rt.onInvoke(ctx, msg)
	}
}

type ExitNode struct {
	baseGraphNode
}

func (node *ExitNode) newRuntime(sessionId string, storePID *actor.PID) *ExitNodeRuntime {
	return &ExitNodeRuntime{
		baseNodeRuntime: makeBaseNodeRuntime(node, sessionId, storePID),
		results:         make(map[string]*proto.Flow),
	}
}

func (node *ExitNode) Type() NodeType {
	return ExitNodeType
}

func (node *ExitNode) Props(sessionId string, store *actor.PID) *actor.Props {
	rt := node.newRuntime(sessionId, store)
	return actor.PropsFromProducer(func() actor.Actor {
		return rt
	})
}

func NewExitNode(id string) *ExitNode {
	return &ExitNode{
		baseGraphNode: baseGraphNode{
			id: id,
		},
	}
}
