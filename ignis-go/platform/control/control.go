package control

import (
	"fmt"
	"strings"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils/errors"
)

type Controller struct {
	manager    *python.VenvManager
	controller remote.Controller
	store      *proto.StoreRef
	nodes      map[string]*task.Node
	runtimes   map[string]*actor.PID
}

func (c *Controller) onAppendActor(ctx actor.Context, a *controller.AppendActor) {
	ctx.Logger().Info("control: append actor",
		"name", a.Name,
		"params", a.Params,
	)
	node := task.NewNode(a.Name, a.Params, func(sessionId string, store *actor.PID) task.Handler {
		return task.HandlerFromPID(sessionId, store, a.Params, a.PID)
	})
	c.nodes[a.Name] = node
}

func (c *Controller) onAppendPyFunc(ctx actor.Context, f *controller.AppendPyFunc) {
	ctx.Logger().Info("control: append python function",
		"name", f.Name,
		"params", f.Params,
	)
	pyFunc, err := functions.NewPy(c.manager, f.Name, f.Params, f.Venv, f.Requirements, f.PickledObject, f.Language)
	if err != nil {
		panic(err)
	}
	node := task.NewNode(f.Name, pyFunc.Params(), func(sessionId string, store *actor.PID) task.Handler {
		return task.HandlerFromFunction(sessionId, store, pyFunc)
	})
	c.nodes[f.Name] = node
}

func (c *Controller) onAppendData(ctx actor.Context, data *controller.AppendData) {
	obj := data.Object
	ctx.Logger().Info("control: append data node",
		"id", obj.ID,
		"session", data.SessionID,
	)
	ctx.Send(c.store.PID, &messages.SaveObject{
		Value: obj,
		Callback: func(ctx actor.Context, ref *proto.Flow) {
			ret := controller.NewReturnResult(data.SessionID, "", obj.ID, ref, nil)
			c.controller.SendChan() <- ret
		},
	})
}

func (c *Controller) onAppendArg(ctx actor.Context, arg *controller.AppendArg) {
	ctx.Logger().Info("control: append arg",
		"name", arg.Name,
		"param", arg.Param,
		"session", arg.SessionID,
		"instance", arg.InstanceID,
	)
	sessionId := fmt.Sprintf("%s::%s::%s", arg.Name, arg.SessionID, arg.InstanceID)
	pid, ok := c.runtimes[sessionId]
	if !ok {
		node, ok := c.nodes[arg.Name]
		if !ok {
			return
		}

		props := node.Props(sessionId, c.store.PID, &proto.ActorRef{
			Store: c.store,
			ID:    "controller",
			PID:   ctx.Self(),
		})
		pid, _ = ctx.SpawnNamed(props, "runtime."+sessionId)
		c.runtimes[sessionId] = pid
	}

	switch v := arg.Value.Object.(type) {
	case *controller.Data_Ref:
		invoke := &proto.Invoke{
			SessionID: sessionId,
			Param:     arg.Param,
			Value:     v.Ref,
		}
		ctx.Send(pid, invoke)
	case *controller.Data_Encoded:
		save := &messages.SaveObject{
			Value: v.Encoded,
			Callback: func(ctx actor.Context, ref *proto.Flow) {
				invoke := &proto.Invoke{
					SessionID: sessionId,
					Param:     arg.Param,
					Value:     ref,
				}
				ctx.Send(pid, invoke)
			},
		}
		ctx.Send(c.store.PID, save)
	}
}

func (c *Controller) onUnknown(ctx actor.Context, msg *controller.Message) {
	ctx.Logger().Info("control: unknown message", "msg", msg)
}

func (c *Controller) onControllerMessage(ctx actor.Context, msg *controller.Message) {
	switch cmd := msg.Command.(type) {
	case *controller.Message_AppendActor:
		c.onAppendActor(ctx, cmd.AppendActor)
	case *controller.Message_AppendPyFunc:
		c.onAppendPyFunc(ctx, cmd.AppendPyFunc)
	case *controller.Message_AppendData:
		c.onAppendData(ctx, cmd.AppendData)
	case *controller.Message_AppendArg:
		c.onAppendArg(ctx, cmd.AppendArg)
	default:
		c.onUnknown(ctx, msg)
	}
}

func (c *Controller) onReturn(ctx actor.Context, ir *proto.InvokeRemote) {
	ret := ir.Invoke
	splits := strings.SplitN(ret.SessionID, "::", 3)
	name, sessionId, instanceId := splits[0], splits[1], splits[2]
	ctx.Logger().Info("control: execution done",
		"name", name,
		"session", sessionId,
		"instance", instanceId,
	)

	var msg *controller.Message
	if ret.Error != "" {
		msg = controller.NewReturnResult(sessionId, instanceId, name, nil, errors.New(ret.Error))
	} else {
		msg = controller.NewReturnResult(sessionId, instanceId, name, ret.Value, nil)
	}
	c.controller.SendChan() <- msg
}

func (c *Controller) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *controller.Message:
		c.onControllerMessage(ctx, msg)
	case *proto.InvokeRemote:
		c.onReturn(ctx, msg)
	}
}

func SpawnTaskController(
	ctx *actor.RootContext,
	store *proto.StoreRef,
	venvs *python.VenvManager,
	cm remote.ControllerManager,
	onClose func(),
) *proto.ActorRef {
	c := cm.NextController()
	props := actor.PropsFromProducer(func() actor.Actor {
		return &Controller{
			manager:    venvs,
			controller: c,
			store:      store,
			nodes:      make(map[string]*task.Node),
			runtimes:   make(map[string]*actor.PID),
		}
	})
	pid, _ := ctx.SpawnNamed(props, "controller")
	go func() {
		defer onClose()
		for msg := range c.RecvChan() {
			ctx.Send(pid, msg)
		}
	}()

	return &proto.ActorRef{
		Store: store,
		ID:    "controller",
		PID:   pid,
	}
}
