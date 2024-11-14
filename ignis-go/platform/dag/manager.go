package dag

import (
	"actors/platform/system"
	"actors/platform/utils"
	"actors/proto"
	protoDAG "actors/proto/dag"
	"errors"

	"github.com/asynkron/protoactor-go/actor"
)

type Manager struct {
	tasks   map[string]*DAG
	results map[string]*protoDAG.ExecutionResult
}

func (m *Manager) respond(ctx actor.Context, err error, replyTo *actor.PID) {
	if replyTo == nil {
		return
	}

	if err != nil {
		ctx.Send(replyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: err.Error(),
		})
	} else {
		ctx.Send(replyTo, &proto.Ack{Sender: ctx.Self()})
	}
}

func (m *Manager) onCreate(ctx actor.Context, dag string, replyTo *actor.PID) {
	var err error

	if _, ok := m.tasks[dag]; ok {
		err = errors.New("duplicated task")
	} else {
		m.tasks[dag] = New(dag, ctx)
	}

	m.respond(ctx, err, replyTo)
}

func (m *Manager) onAppendNode(ctx actor.Context, name string, req *protoDAG.AppendNode, replyTo *actor.PID) {
	var err error

	dag, ok := m.tasks[name]
	if !ok || dag.state != DAGCreated {
		err = errors.New("task not found or not in Created state")
		m.respond(ctx, err, replyTo)
		return
	}

	switch opt := req.Option.(type) {
	case *protoDAG.AppendNode_Actor:
		err = dag.AddNodeFromActor(req.Name, req.Params, opt.Actor)
	case *protoDAG.AppendNode_Group:
		err = dag.AddNodeFromActorName(req.Name, req.Params, opt.Group)
	case *protoDAG.AppendNode_Value:
		err = dag.AddDataNodeFromEncoded(req.Name, opt.Value)
	}

	m.respond(ctx, err, replyTo)
}

func (m *Manager) onAppendEdge(ctx actor.Context, name string, req *protoDAG.AppendEdge, replyTo *actor.PID) {
	var err error

	dag, ok := m.tasks[name]
	if !ok || dag.state != DAGCreated {
		err = errors.New("task not found or not in Created state")
	} else {
		err = dag.AddEdge(req.From, req.To, req.Param)
	}

	m.respond(ctx, err, replyTo)
}

func (m *Manager) onAppendOutput(ctx actor.Context, name string, req *protoDAG.AppendOutput, replyTo *actor.PID) {
	var err error

	dag, ok := m.tasks[name]
	if !ok || dag.state != DAGCreated {
		err = errors.New("task not found or not in Created state")
	} else {
		_, err = dag.AddOutputEdge(req.Node)
	}

	m.respond(ctx, err, replyTo)
}

func (m *Manager) onExecute(ctx actor.Context, name string, replyTo *actor.PID) {
	dag, ok := m.tasks[name]
	if !ok || dag.state != DAGFrozen {
		ctx.Send(replyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: "task not found or not frozen",
		})
		return
	}

	executionId := utils.GenExecutionID()
	dag.Execute(executionId)

	go func() {
		results := make(map[string]*Flow)
		objects := make(map[string]*proto.EncodedObject)
		for node, ch := range dag.outputs {
			results[node] = <-ch
		}
		var resErr error
		for node, res := range results {
			if res.Error != nil {
				resErr = res.Error
			} else if obj, err := system.FlowData(ctx, res.Proto); err != nil {
				resErr = err
			} else if encObj, err := obj.Encode(); err != nil {
				resErr = err
			} else {
				objects[node] = encObj
			}
			if resErr != nil {
				break
			}
		}

		result := &protoDAG.ExecutionResult{ExecutionID: executionId, DAG: name}
		if resErr != nil {
			result.Error = resErr.Error()
		} else {
			result.Results = objects
		}

		m.results[executionId] = result
		ctx.Send(replyTo, result)
	}()
}

func (m *Manager) onServe(ctx actor.Context, name string, replyTo *actor.PID) {
	dag, ok := m.tasks[name]
	if !ok || dag.state != DAGCreated {
		ctx.Send(replyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: "task not found or not in Created state",
		})
		return
	}

	go dag.Serve()
	dag.state = DAGFrozen
	ctx.Send(replyTo, &proto.Ack{Sender: ctx.Self()})
}

func (m *Manager) onCommand(ctx actor.Context, cmd *protoDAG.Command) {
	switch c := cmd.Command.(type) {
	case *protoDAG.Command_Create:
		m.onCreate(ctx, cmd.DAG, cmd.ReplyTo)
	case *protoDAG.Command_AppendNode:
		m.onAppendNode(ctx, cmd.DAG, c.AppendNode, cmd.ReplyTo)
	case *protoDAG.Command_AppendEdge:
		m.onAppendEdge(ctx, cmd.DAG, c.AppendEdge, cmd.ReplyTo)
	case *protoDAG.Command_AppendOutput:
		m.onAppendOutput(ctx, cmd.DAG, c.AppendOutput, cmd.ReplyTo)
	case *protoDAG.Command_Execute:
		m.onExecute(ctx, cmd.DAG, cmd.ReplyTo)
	case *protoDAG.Command_Serve:
		m.onServe(ctx, cmd.DAG, cmd.ReplyTo)
	}
}

func (m *Manager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protoDAG.Command:
		m.onCommand(ctx, msg)
	default:
		ctx.Logger().Error("Unknown message")
	}
}

func NewManager() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Manager{
			tasks:   make(map[string]*DAG),
			results: make(map[string]*protoDAG.ExecutionResult),
		}
	})
}
