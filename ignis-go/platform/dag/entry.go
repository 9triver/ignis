package dag

import (
	"actors/platform/store"
	"actors/platform/system"
	"actors/proto"
	"context"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type EntryNode interface {
	DAGNode
	Execute(executionID string)
}

type DataNode struct {
	baseDAGNode
	obj *store.Object
}

func (node *DataNode) Execute(executionID string) {
	protoFlow := &proto.Flow{
		Actor:    system.ActorRef().Head(),
		ObjectID: node.obj.ID,
	}
	flow := &Flow{
		ExecutionID: executionID,
		Proto:       protoFlow,
	}

	node.returnFlow(flow)
}

func NewDataNode(id string, ctx context.Context, obj *store.Object) EntryNode {
	return &DataNode{
		baseDAGNode: baseDAGNode{
			id:      id,
			ctx:     ctx,
			outputs: make([]chan<- *Flow, 0),
		},
		obj: obj,
	}
}

type ActorEntryNode struct {
	baseDAGNode
	actorCtx actor.Context
	actor    *actor.PID
}

func (node *ActorEntryNode) Execute(executionID string) {
	flow := &Flow{ExecutionID: executionID}
	fut := node.actorCtx.RequestFuture(node.actor, &proto.Execute{}, system.ExecutionTimeout)
	resp, err := fut.Result()
	if err != nil {
		flow.Error = fmt.Errorf("node %s: error executing actor: %w", node.id, err)
	} else {
		flow.Parse(node.id, resp)
	}

	node.returnFlow(flow)
}

func NewActorEntryNode(
	id string,
	ctx context.Context,
	actorCtx actor.Context,
	actor *actor.PID,
) EntryNode {
	return &ActorEntryNode{
		baseDAGNode: baseDAGNode{
			id:      id,
			ctx:     ctx,
			outputs: make([]chan<- *Flow, 0),
		},
		actorCtx: actorCtx,
		actor:    actor,
	}
}
