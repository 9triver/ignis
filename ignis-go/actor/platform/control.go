package platform

import (
	"fmt"
	"strings"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/dag"
	"github.com/9triver/ignis/actor/dag/handlers"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils/errors"
)

type Controller struct {
	manager    *python.VenvManager
	controller remote.Controller
	storePID   *actor.PID
	nodes      map[string]dag.Task
	runtimes   map[string]*actor.PID
	remotes    map[string]*actor.PID
}

func (c *Controller) onAppendActor(ctx actor.Context, a *controller.AppendActor) {
	ctx.Logger().Info("control: append actor",
		"name", a.Name,
		"params", a.Params,
	)
	c.remotes[a.Name] = a.PID
	node := dag.NewTaskNode(a.Name, a.Params, func(sessionId string, store *actor.PID) dag.TaskHandler {
		return handlers.FromPID(sessionId, store, a.Params, a.PID)
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
	node := dag.NewTaskNode(f.Name, pyFunc.Params(), func(sessionId string, store *actor.PID) dag.TaskHandler {
		return handlers.FromFunction(sessionId, store, pyFunc)
	})
	c.nodes[f.Name] = node
}

func (c *Controller) onAppendData(ctx actor.Context, data *controller.AppendData) {
	obj := data.Object
	ctx.Logger().Info("control: append data node",
		"id", obj.ID,
		"session", data.SessionID,
	)
	ctx.Send(c.storePID, &store.SaveObject{
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
	sessionId := fmt.Sprintf("%s.%s", arg.SessionID, arg.InstanceID)
	name := fmt.Sprintf("%s::%s", arg.Name, sessionId)
	pid, ok := c.runtimes[name]
	if !ok {
		node, ok := c.nodes[arg.Name]
		if !ok {
			return
		}

		props := node.Props(sessionId, c.storePID)
		pid = ctx.Spawn(props)
		ctx.Send(pid, &messages.Successor{
			ID:    "store",
			PID:   ctx.Self(),
			Param: arg.Name,
		})
		c.runtimes[name] = pid

		if remotePID, ok := c.remotes[arg.Name]; ok {
			ctx.Send(remotePID, &messages.CreateSession{
				SessionID: sessionId,
				Successors: []*messages.Successor{
					{
						ID:    "store",
						PID:   ctx.Self(),
						Param: arg.Name,
					},
				},
			})
		}
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
		save := &store.SaveObject{
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
		ctx.Send(c.storePID, save)
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

func (c *Controller) onReturn(ctx actor.Context, ret *proto.Invoke) {
	splits := strings.SplitN(ret.SessionID, ".", 2)
	sessionId, instanceId := splits[0], splits[1]
	ctx.Logger().Info("control: execution done",
		"name", ret.Param,
		"session", sessionId,
		"instance", instanceId,
	)

	var msg *controller.Message
	if ret.Error != "" {
		msg = controller.NewReturnResult(sessionId, instanceId, ret.Param, nil, errors.New(ret.Error))
	} else {
		msg = controller.NewReturnResult(sessionId, instanceId, ret.Param, ret.Value, nil)
	}
	c.controller.SendChan() <- msg
}

func (c *Controller) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *controller.Message:
		c.onControllerMessage(ctx, msg)
	case *proto.Invoke:
		c.onReturn(ctx, msg)
	}
}

func NewTaskController(store *actor.PID, venvs *python.VenvManager, cm remote.ControllerManager) (remote.Controller, *actor.Props) {
	c := cm.NextController()
	return c, actor.PropsFromProducer(func() actor.Actor {
		return &Controller{
			manager:    venvs,
			controller: c,
			storePID:   store,
			nodes:      make(map[string]dag.Task),
			runtimes:   make(map[string]*actor.PID),
			remotes:    make(map[string]*actor.PID),
		}
	})
}
