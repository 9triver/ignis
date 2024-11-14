package dag

import (
	"actors/proto"
	"context"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type InterNode interface {
	DAGNode
	SetInput(param string, ch <-chan *Flow) error
	Serve()
}

type baseInterNode struct {
	baseDAGNode
	actorCtx actor.Context
	queue    *ExecutionQueue
	inputs   map[string]<-chan *Flow
}

func (node *baseInterNode) Check() error {
	for name, ch := range node.inputs {
		if ch == nil {
			return fmt.Errorf("%s: input %s not set", node.id, name)
		}
	}
	if len(node.outputs) == 0 {
		return fmt.Errorf("%s: no outputs", node.id)
	}

	return nil
}

func (node *baseInterNode) IsInter() bool {
	return true
}

func (node *baseInterNode) SetInput(param string, ch <-chan *Flow) error {
	if ch, ok := node.inputs[param]; !ok {
		return fmt.Errorf("%s: no specified input param %s", node.id, param)
	} else if ch != nil {
		return fmt.Errorf("%s: already has input param %s", node.id, param)
	}
	node.inputs[param] = ch

	go func() {
		for flow := range ch {
			flow.Param = param
			node.queue.Input(flow)
		}
	}()

	return nil
}

type ActorInterNode struct {
	baseInterNode
	actor *actor.PID
}

func (node *ActorInterNode) handleExecute(execute *proto.Execute) {
	result := &Flow{ExecutionID: execute.ExecutionID}
	pid := node.actorCtx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch obj := c.Message().(type) {
		case *actor.Started:
			return
		case *proto.Flow:
			result.Proto = obj
		case *proto.Error:
			result.Error = fmt.Errorf("node %s: %s", node.id, obj.Message)
		default:
			result.Error = fmt.Errorf("node %s: invalid object type for flow", node.id)
		}
		node.returnFlow(result)
	}))
	execute.ReplyTo = pid
	node.actorCtx.Send(node.actor, execute)
}

func (node *ActorInterNode) Serve() {
	for {
		select {
		case <-node.ctx.Done():
			return
		case execute := <-node.queue.Output():
			node.handleExecute(execute)
		}
	}
}

func NewActorInterNode(
	id string,
	ctx context.Context,
	actorCtx actor.Context,
	actor *actor.PID,
	deps []string,
) InterNode {
	inputs := make(map[string]<-chan *Flow)
	for _, dep := range deps {
		inputs[dep] = nil
	}
	outputs := make([]chan<- *Flow, 0)

	return &ActorInterNode{
		baseInterNode: baseInterNode{
			baseDAGNode: baseDAGNode{
				id:      id,
				ctx:     ctx,
				outputs: outputs,
			},
			queue:    NewExecutionQueue(id, deps),
			actorCtx: actorCtx,
			inputs:   inputs,
		},
		actor: actor,
	}
}

type GroupInterNode struct {
	baseInterNode
	group *ActorGroup
}

func (node *GroupInterNode) handleExecute(selected *actor.PID, execute *proto.Execute) {
	result := &Flow{ExecutionID: execute.ExecutionID}

	if selected == nil {
		result.Error = fmt.Errorf("node %s: no available candidate", node.id)
		node.returnFlow(result)
		return
	}

	pid := node.actorCtx.Spawn(actor.PropsFromFunc(func(c actor.Context) {
		switch obj := c.Message().(type) {
		case *actor.Started:
			return
		case *proto.Flow:
			result.Proto = obj
		case *proto.Error:
			result.Error = fmt.Errorf("node %s: %s", node.id, obj.Message)
		default:
			result.Error = fmt.Errorf("node %s: invalid object type for flow", node.id)
		}
		node.returnFlow(result)
		node.group.TaskDone(selected)
	}))
	execute.ReplyTo = pid
	node.actorCtx.Send(selected, execute)
}

func (node *GroupInterNode) Serve() {
	for {
		select {
		case <-node.ctx.Done():
			return
		case execute := <-node.queue.Output():
			sources := actor.NewPIDSet()
			for _, flow := range execute.Flows {
				if flow.Actor != nil {
					sources.Add(flow.Actor)
				}
			}
			selected := node.group.Select(sources)
			node.group.TaskStart(selected)
			node.handleExecute(selected, execute)
		}
	}
}

func NewGroupInterNode(
	id string,
	ctx context.Context,
	actorCtx actor.Context,
	actors *actor.PIDSet,
	deps []string,
) InterNode {
	inputs := make(map[string]<-chan *Flow)
	for _, dep := range deps {
		inputs[dep] = nil
	}

	outputs := make([]chan<- *Flow, 0)
	return &GroupInterNode{
		baseInterNode: baseInterNode{
			baseDAGNode: baseDAGNode{
				id:      id,
				ctx:     ctx,
				outputs: outputs,
			},
			actorCtx: actorCtx,
			queue:    NewExecutionQueue(id, deps),
			inputs:   inputs,
		},
		group: NewActorGroup(actors),
	}
}
