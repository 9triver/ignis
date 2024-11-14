package dag

import (
	protoDAG "actors/proto/dag"

	"github.com/asynkron/protoactor-go/actor"
)

const (
	CommandCreate       = protoDAG.CommandType_COMMAND_Create
	CommandAppendNode   = protoDAG.CommandType_COMMAND_AppendNode
	CommandAppendEdge   = protoDAG.CommandType_COMMAND_AppendEdge
	CommandAppendOutput = protoDAG.CommandType_COMMAND_AppendOutput
	CommandServe        = protoDAG.CommandType_COMMAND_Serve
	CommandDestroy      = protoDAG.CommandType_COMMAND_Execute
)

func MakeCommand(dag string, cmd interface{}, replyTo *actor.PID) *protoDAG.Command {
	ret := &protoDAG.Command{DAG: dag, ReplyTo: replyTo}
	switch cmd := cmd.(type) {
	case *protoDAG.Create:
		ret.Type = CommandCreate
		ret.Command = &protoDAG.Command_Create{Create: cmd}
	case *protoDAG.AppendNode:
		ret.Type = CommandAppendNode
		ret.Command = &protoDAG.Command_AppendNode{AppendNode: cmd}
	case *protoDAG.AppendEdge:
		ret.Type = CommandAppendEdge
		ret.Command = &protoDAG.Command_AppendEdge{AppendEdge: cmd}
	case *protoDAG.AppendOutput:
		ret.Type = CommandAppendOutput
		ret.Command = &protoDAG.Command_AppendOutput{AppendOutput: cmd}
	case *protoDAG.Execute:
		ret.Type = CommandDestroy
		ret.Command = &protoDAG.Command_Execute{Execute: cmd}
	case *protoDAG.Serve:
		ret.Type = CommandServe
		ret.Command = &protoDAG.Command_Serve{Serve: cmd}
	}
	return ret
}
