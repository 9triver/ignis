package dag

import (
	"actors/platform/actor/compute"
	"actors/proto"

	"actors/platform/handlers"
	"actors/platform/store"
	"actors/platform/system"

	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type DAGNode interface {
	ID() string
	Check() error
	Cleanup() error
	AppendOutput(ch chan<- *Flow)
	IsInter() bool
}

type baseDAGNode struct {
	id      string
	ctx     context.Context
	outputs []chan<- *Flow
}

func (node *baseDAGNode) ID() string {
	return node.id
}

func (node *baseDAGNode) Check() error {
	if len(node.outputs) == 0 {
		return fmt.Errorf("%s: no outputs", node.id)
	}
	return nil
}

func (node *baseDAGNode) Cleanup() error {
	for _, ch := range node.outputs {
		close(ch)
	}
	return nil
}

func (node *baseDAGNode) AppendOutput(ch chan<- *Flow) {
	node.outputs = append(node.outputs, ch)
}

func (node *baseDAGNode) IsInter() bool {
	return false
}

func (node *baseDAGNode) returnFlow(flow *Flow) {
	for _, ch := range node.outputs {
		select {
		case <-node.ctx.Done():
			return
		default:
			ch <- flow
		}
	}
}

type DAGState int

const (
	DAGCreated DAGState = iota
	DAGFrozen
	DAGFinished
)

type DAG struct {
	ctx    context.Context
	cancel context.CancelFunc

	id         string
	state      DAGState
	actorCtx   actor.Context
	entryNodes map[string]EntryNode
	interNodes map[string]InterNode
	outputs    map[string]<-chan *Flow
}

func New(id string, actorCtx actor.Context) *DAG {
	ctx, cancel := context.WithCancel(context.Background())
	return &DAG{
		id:         id,
		state:      DAGCreated,
		ctx:        ctx,
		cancel:     cancel,
		actorCtx:   actorCtx,
		entryNodes: make(map[string]EntryNode),
		interNodes: make(map[string]InterNode),
		outputs:    make(map[string]<-chan *Flow),
	}
}

func (dag *DAG) isEntry(name string) bool {
	_, ok := dag.entryNodes[name]
	return ok
}

func (dag *DAG) isInter(name string) bool {
	_, ok := dag.interNodes[name]
	return ok
}

func (dag *DAG) ID() string {
	return dag.id
}

func (dag *DAG) Outputs() map[string]<-chan *Flow {
	return dag.outputs
}

func (dag *DAG) Contains(name string) bool {
	return dag.isEntry(name) || dag.isInter(name)
}

func (dag *DAG) AddNodeFromFunction(id string, handler handlers.Function, processLatency time.Duration) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}

	actor := compute.NewActor(handler, processLatency)
	pid := dag.actorCtx.Spawn(actor.Props())
	dag.interNodes[id] = NewActorInterNode(id, dag.ctx, dag.actorCtx, pid, handler.Params())
	return nil
}

func (dag *DAG) AddNodeFromActor(id string, deps []string, actor *actor.PID) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}
	dag.interNodes[id] = NewActorInterNode(id, dag.ctx, dag.actorCtx, actor, deps)
	return nil
}

func (dag *DAG) AddNodeFromActors(
	id string,
	deps []string,
	actors ...*actor.PID,
) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}
	dag.interNodes[id] = NewGroupInterNode(id, dag.ctx, dag.actorCtx, actor.NewPIDSet(actors...), deps)
	return nil
}

func (dag *DAG) AddNodeFromActorName(
	id string,
	deps []string,
	actorName string,
) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}

	actorSet, ok := system.HeadRef().ActorSet(actorName)
	if !ok || actorSet.Empty() {
		dag.actorCtx.Logger().Debug("actor set empty")
		return fmt.Errorf("no available actors for %s", actorName)
	}

	dag.interNodes[id] = NewGroupInterNode(id, dag.ctx, dag.actorCtx, actorSet, deps)
	return nil
}

func (dag *DAG) AddDataNode(id string, data any, language store.Language) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}

	store := system.ActorRef().Store()
	dag.entryNodes[id] = NewDataNode(id, dag.ctx, store.Add(data, language))
	return nil
}

func (dag *DAG) AddDataNodeFromEncoded(id string, encObj *proto.EncodedObject) error {
	if dag.Contains(id) {
		return errors.New("node already exists")
	}

	store := system.ActorRef().Store()
	obj, err := store.AddEncoded(encObj)
	if err != nil {
		return err
	}
	dag.entryNodes[id] = NewDataNode(id, dag.ctx, obj)
	return nil
}

func (dag *DAG) AddEdge(from, to, param string) error {
	if dag.isEntry(to) {
		return errors.New("cannot append edge to entry node")
	}

	toNode, ok := dag.interNodes[to]
	if !ok {
		return errors.New("to node not found")
	}

	var fromNode DAGNode
	if dag.isEntry(from) {
		fromNode = dag.entryNodes[from]
	} else if dag.isInter(from) {
		fromNode = dag.interNodes[from]
	} else {
		return errors.New("from node not found")
	}

	edge := make(chan *Flow, system.ChannelBufferSize)
	if err := toNode.SetInput(param, edge); err != nil {
		return err
	}

	fromNode.AppendOutput(edge)
	return nil
}

func (dag *DAG) AddOutputEdge(node string) (<-chan *Flow, error) {
	n, ok := dag.interNodes[node]
	if !ok {
		return nil, errors.New("node not found")
	}
	edge := make(chan *Flow, system.ChannelBufferSize)
	n.AppendOutput(edge)
	dag.outputs[node] = edge
	return edge, nil
}

func (dag *DAG) Destroy() {
	select {
	case <-dag.ctx.Done():
		return
	default:
		dag.state = DAGFinished
		dag.cancel()
		for _, node := range dag.entryNodes {
			node.Cleanup()
		}
		for _, node := range dag.interNodes {
			node.Cleanup()
		}
	}
}

func (dag *DAG) Serve() {
	dag.state = DAGFrozen
	for _, node := range dag.interNodes {
		go node.Serve()
	}
}

func (dag *DAG) Execute(executionId string) {
	for _, node := range dag.entryNodes {
		go node.Execute(executionId)
	}
}
