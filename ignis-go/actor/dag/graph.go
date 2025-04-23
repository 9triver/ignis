package dag

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type graphEdge struct {
	from, to, param string
}

type ChildTerminated struct {
	PID   *actor.PID
	Error error
}

type Graph struct {
	id      string
	nodes   utils.Map[string, Node]
	entries utils.Set[string]
	edges   utils.Set[*graphEdge]
}

func (g *Graph) ID() string {
	return g.id
}

func (g *Graph) Inputs() utils.Set[string] {
	return g.entries
}

func (g *Graph) Type() NodeType {
	return GraphNodeType
}

func (g *Graph) AppendNode(node Task) {
	if g.nodes.Contains(node.ID()) {
		return
	}
	g.nodes.Put(node.ID(), node)
}

func (g *Graph) AddEntry(node Entry) {
	if g.nodes.Contains(node.ID()) {
		return
	}
	g.nodes.Put(node.ID(), node)
	g.entries.Add(node.ID())
}

func (g *Graph) AddEdge(from, to, param string) {
	if !g.entries.Contains(from) && !g.nodes.Contains(to) {
		return
	}

	if g.entries.Contains(to) || !g.nodes.Contains(to) {
		return
	}

	g.edges.Add(&graphEdge{from, to, param})
}

func (g *Graph) newRuntime(root bool, sessionId string, store *actor.PID) *GraphRuntime {
	return &GraphRuntime{
		baseNodeRuntime: makeBaseNodeRuntime(g, sessionId, store),
		root:            root,
		actors:          utils.MakeMap[string, *actor.PID](),
	}
}

func (g *Graph) RootProps(sessionId string, store *actor.PID) *actor.Props {
	if sessionId == "" {
		sessionId = utils.GenSessionID()
	}
	return g.props(true, sessionId, store)
}

func (g *Graph) Props(sessionId string, store *actor.PID) *actor.Props {
	return g.props(false, sessionId, store)
}

func (g *Graph) props(root bool, sessionId string, store *actor.PID) *actor.Props {
	rt := g.newRuntime(root, sessionId, store)
	return actor.PropsFromProducer(func() actor.Actor {
		return rt
	}, actor.WithOnInit(func(ctx actor.Context) {
		for nodeId, node := range g.nodes {
			props := node.Props(sessionId, store)
			rt.actors[nodeId], _ = ctx.SpawnNamed(props, fmt.Sprintf("actor-%s::%s::%s", g.id, sessionId, nodeId))
		}

		for edge := range g.edges {
			from := rt.actors[edge.from]
			to := rt.actors[edge.to]
			ctx.Send(from, &messages.Successor{ID: edge.to, Param: edge.param, PID: to})
		}
	}))
}

func New(id string, nodes ...Node) *Graph {
	g := &Graph{
		id:      id,
		nodes:   utils.MakeMap[string, Node](),
		entries: utils.MakeSet[string](),
		edges:   utils.MakeSet[*graphEdge](),
	}

	for _, node := range nodes {
		g.nodes.Put(node.ID(), node)
		if node.Type() == EntryNodeType {
			g.entries.Add(node.ID())
		}
	}

	return g
}

type GraphRuntime struct {
	baseNodeRuntime[*Graph]
	root   bool
	actors utils.Map[string, *actor.PID]
}

func (rt *GraphRuntime) closeWith(ctx actor.Context, err error) {
	ctx.Logger().Info("actor terminating",
		"name", rt.node.ID(),
		"session", rt.sessionId,
		"error", err,
	)

	// if rt is not root graph, send back error to parent
	if !rt.root {
		ctx.Send(ctx.Parent(), &ChildTerminated{
			PID:   ctx.Self(),
			Error: err,
		})
	}

	ctx.Stop(ctx.Self())
}

func (rt *GraphRuntime) onChildTerminated(ctx actor.Context, term *ChildTerminated) {
	if term.Error == nil {
		return
	}
	ctx.Logger().Error("graph failed",
		"node", term.PID,
		"reason", term.Error,
		"graph", rt.node.id,
		"session", rt.sessionId,
	)
	rt.closeWith(ctx, term.Error)
}

func (rt *GraphRuntime) onInvoke(ctx actor.Context, _ *proto.Invoke) {
	ctx.Logger().Warn(
		"ignoring invoke",
		"reason", "graph node does not support invoke currently",
		"graph", rt.node.id,
		"session", rt.sessionId,
	)
}

func (rt *GraphRuntime) onInvokeEmpty(ctx actor.Context) {
	ctx.Logger().Info("graph: receive invoke",
		"id", rt.node.id,
		"session", rt.sessionId,
	)
	for entry := range rt.node.entries {
		pid, ok := rt.actors.Get(entry)
		if !ok {
			continue
		}
		ctx.Send(pid, &proto.InvokeEmpty{})
	}
}

func (rt *GraphRuntime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
	case *messages.Successor:
		rt.onAddEdge(ctx, msg)
	case *proto.Invoke:
		rt.onInvoke(ctx, msg)
	case *proto.InvokeEmpty:
		rt.onInvokeEmpty(ctx)
	case *ChildTerminated:
		rt.onChildTerminated(ctx, msg)
	}
}
