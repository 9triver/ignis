package dag

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type NodeType int

const (
	EntryNodeType NodeType = iota // EntryNode of the DAG
	TaskNodeType                  // TaskNode with inputs and outputs, carrying task
	ExitNodeType                  // ExitNode that aggregates outputs of the DAG
	GraphNodeType                 // Graph wrapper for a sub DAG
)

type Node interface {
	ID() string
	Type() NodeType
	Props(sessionId string, store *actor.PID) *actor.Props
}

type Entry interface {
	Node
	Value() *proto.Flow
}

type Exit interface {
	Node
}

type Task interface {
	Node
	Inputs() utils.Set[string]
}

type baseGraphNode struct {
	id string
}

func (node *baseGraphNode) ID() string {
	return node.id
}

type NodeRuntime interface {
	actor.Actor
	Type() NodeType
}

// NodeRuntime serves as a bridge between graph definition and graph execution actors.
// A node runtime holds an actor context and is accessible to node definition (referred by `P`)
type baseNodeRuntime[P Node] struct {
	sessionId  string
	node       P
	store      *actor.PID
	successors []*proto.Successor
}

func (rt *baseNodeRuntime[P]) closeWith(ctx actor.Context, err error) {
	ctx.Logger().Info("actor terminating",
		"name", rt.node.ID(),
		"session", rt.sessionId,
		"error", err,
	)
	ctx.Send(ctx.Parent(), &ChildTerminated{
		PID:   ctx.Self(),
		Error: err,
	})
	ctx.Stop(ctx.Self())
}

func (rt *baseNodeRuntime[P]) onAddEdge(ctx actor.Context, edge *proto.Successor) {
	ctx.Logger().Info("add edge",
		"from", rt.node.ID(),
		"to", edge.ID,
		"param", edge.Param,
		"session", rt.sessionId,
	)
	rt.successors = append(rt.successors, edge)
}

func (rt *baseNodeRuntime[P]) SessionID() string {
	return rt.sessionId
}

func (rt *baseNodeRuntime[P]) Node() P {
	return rt.node
}

func (rt *baseNodeRuntime[P]) Type() NodeType {
	return rt.node.Type()
}

func makeBaseNodeRuntime[P Node](node P, sessionId string, store *actor.PID) baseNodeRuntime[P] {
	return baseNodeRuntime[P]{
		node:      node,
		sessionId: sessionId,
		store:     store,
	}
}

var (
	_ Entry = (*EntryNode)(nil)
	_ Task  = (*TaskNode)(nil)
	_ Task  = (*Graph)(nil)
	_ Exit  = (*ExitNode)(nil)
)
