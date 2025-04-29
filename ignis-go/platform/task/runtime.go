package task

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
)

// Runtime serves as a bridge between graph definition and graph execution actors.
// A node runtime holds an actor context and is accessible to node definition (referred by `P`)
type Runtime struct {
	sessionId string
	replyTo   *proto.ActorRef
	node      *Node
	handler   Handler
}

func (rt *Runtime) SessionID() string {
	return rt.sessionId
}

func (rt *Runtime) closeWith(ctx actor.Context, err error) {
	ctx.Logger().Info("node: actor terminating",
		"name", rt.node.ID(),
		"session", rt.sessionId,
		"error", err,
	)
	if err != nil {
		println(errors.Stacktrace(err))
	}
	ctx.Stop(ctx.Self())
}

func (rt *Runtime) onInvoke(ctx actor.Context, invoke *proto.Invoke) {
	ctx.Logger().Info("task: receive invoke",
		"id", rt.node.id,
		"param", invoke.Param,
		"session", rt.sessionId,
	)
	ready, err := rt.handler.Invoke(ctx, invoke.Param, invoke.Value)
	if err != nil {
		rt.closeWith(ctx, err)
		return
	}

	if ready {
		err := rt.handler.Start(ctx, rt.replyTo)
		rt.closeWith(ctx, err)
	}
}

func (rt *Runtime) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.Invoke:
		rt.onInvoke(ctx, msg)
	}
}
