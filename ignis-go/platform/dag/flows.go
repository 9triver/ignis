package dag

import (
	"actors/platform/system"
	"actors/proto"
	"fmt"
	"sync"
)

type Flow struct {
	ExecutionID string
	Param       string
	Proto       *proto.Flow
	Error       error
}

func (f *Flow) Flow() *proto.Flow {
	return f.Proto
}

func (f *Flow) Parse(name string, obj any) {
	switch obj := obj.(type) {
	case *proto.Flow:
		f.Proto = obj
	case *proto.Error:
		f.Error = fmt.Errorf("node %s: %s", name, obj.Message)
	default:
		f.Error = fmt.Errorf("node %s: invalid object type for flow", name)
	}
}

type ParamFlows map[string]*Flow

func (f ParamFlows) Protos() (map[string]*proto.Flow, error) {
	protoFlows := make(map[string]*proto.Flow)
	for k, v := range f {
		if v.Error != nil {
			return nil, fmt.Errorf("flow %s: %w", k, v.Error)
		}
		protoFlows[k] = v.Proto
	}
	return protoFlows, nil
}

func (f ParamFlows) Ready() bool {
	for _, v := range f {
		if v == nil {
			return false
		}
	}
	return true
}

// ExecutionQueue is a queue to store the execution flows. DAG node sends the flow to the queue,
// once all params of the flow  are ready, the queue will wrap them into an `proto.Execute`
// message and send it back to the DAG node.
type ExecutionQueue struct {
	name   string
	mu     sync.Mutex
	deps   []string
	store  map[string]ParamFlows
	output chan *proto.Execute
}

func (q *ExecutionQueue) Input(flow *Flow) {
	if flow.Param == "" {
		return
	}

	q.handleFlow(flow)
}

func (q *ExecutionQueue) Output() <-chan *proto.Execute {
	return q.output
}

func (q *ExecutionQueue) handleFlow(flow *Flow) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.store[flow.ExecutionID]; !ok {
		q.store[flow.ExecutionID] = make(ParamFlows)
		for _, dep := range q.deps {
			q.store[flow.ExecutionID][dep] = nil
		}
	}

	flows := q.store[flow.ExecutionID]
	flows[flow.Param] = flow
	if flows.Ready() {
		defer delete(q.store, flow.ExecutionID)

		protoFlows, err := flows.Protos()
		if err != nil {
			system.Logger().Error("Aborting execution", "execution", flow.ExecutionID, "error", err)
			return
		}
		system.Logger().Info("Execution Ready", "node", q.name, "execution", flow.ExecutionID)
		q.output <- &proto.Execute{
			ExecutionID: flow.ExecutionID,
			Flows:       protoFlows,
		}
	}
}

func NewExecutionQueue(name string, deps []string) *ExecutionQueue {
	return &ExecutionQueue{
		name:   name,
		deps:   deps,
		store:  make(map[string]ParamFlows),
		output: make(chan *proto.Execute, system.ChannelBufferSize),
	}
}
