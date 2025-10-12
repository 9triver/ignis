package control

import (
	"fmt"
	"strings"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/compute"
	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils/errors"
)

type Controller struct {
	manager    *functions.VenvManager
	controller remote.Controller
	store      *proto.StoreRef

	nodes    map[string]*task.Node
	groups   map[string]*task.ActorGroup
	runtimes map[string]*task.Runtime
}

func (c *Controller) onAppendActor(ctx actor.Context, a *controller.AppendActor) {
	ctx.Logger().Info("control: append actor",
		"name", a.Name,
		"params", a.Params,
	)

	info := &task.ActorInfo{
		Ref: a.Ref,
	}
	if group, ok := c.groups[a.Name]; ok {
		group.Push(info)
		return
	}

	group := task.NewGroup(a.Name, info)
	c.groups[a.Name] = group

	node := task.NodeFromActorGroup(a.Name, a.Params, group)
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

	if f.Replicas == 0 {
		f.Replicas = 1
	}
	group := task.NewGroup(f.Name)
	c.groups[f.Name] = group

	for i := range f.Replicas {
		name := fmt.Sprintf("%s-%d", f.Name, i)
		props := compute.NewActor(name, pyFunc, c.store.PID)
		pid := ctx.Spawn(props)
		info := &proto.ActorInfo{
			Ref: &proto.ActorRef{
				ID:    name,
				PID:   pid,
				Store: c.store,
			},
		}
		group.Push(info)
	}

	node := task.NodeFromActorGroup(f.Name, f.Params, group)
	c.nodes[f.Name] = node
}

func (c *Controller) onAppendPyClass(ctx actor.Context, obj *controller.AppendPyClass) {
	ctx.Logger().Info("control: append python class object",
		"name", obj.Name,
		"methods", obj.Methods,
	)

	var methods []string
	var params [][]string

	for _, method := range obj.Methods {
		methods = append(methods, method.Name)
		params = append(params, method.Params)
	}

	pyMethods, err := functions.NewPyClass(c.manager, obj.Name, methods, params, obj.Venv, obj.Requirements, obj.PickledObject, obj.Language)
	if err != nil {
		return
	}

	for _, method := range pyMethods {
		name := fmt.Sprintf("%s.%s", obj.Name, method.Name)
		group := task.NewGroup(name)
		c.groups[name] = group
		props := compute.NewActor(name, method, c.store.PID)
		pid := ctx.Spawn(props)
		info := &proto.ActorInfo{
			Ref: &proto.ActorRef{
				ID:    name,
				PID:   pid,
				Store: c.store,
			},
		}
		group.Push(info)
	}
}

func (c *Controller) onAppendData(ctx actor.Context, data *controller.AppendData) {
	obj := data.Object
	ctx.Logger().Info("control: append data node",
		"id", obj.ID,
		"session", data.SessionID,
	)
	ctx.Send(c.store.PID, &store.SaveObject{
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
	rt, ok := c.runtimes[sessionId]
	if !ok {
		node, ok := c.nodes[arg.Name]
		if !ok {
			return
		}

		rt = node.Runtime(sessionId, c.store.PID, &proto.ActorRef{
			Store: c.store,
			ID:    "controller",
			PID:   ctx.Self(),
		})
		c.runtimes[sessionId] = rt
	}

	switch v := arg.Value.Object.(type) {
	case *controller.Data_Ref:
		if err := rt.Invoke(ctx, arg.Param, v.Ref); err != nil {
			ctx.Logger().Error("control: invoke error",
				"name", arg.Name,
				"param", arg.Param,
				"session", arg.SessionID,
				"instance", arg.InstanceID,
			)
			msg := controller.NewReturnResult(arg.SessionID, arg.InstanceID, arg.Name, nil, err)
			c.controller.SendChan() <- msg
		}
	case *controller.Data_Encoded:
		save := &store.SaveObject{
			Value: v.Encoded,
			Callback: func(ctx actor.Context, ref *proto.Flow) {
				if err := rt.Invoke(ctx, arg.Param, ref); err != nil {
					ctx.Logger().Error("control: invoke error",
						"name", arg.Name,
						"param", arg.Param,
						"session", arg.SessionID,
						"instance", arg.InstanceID,
					)
					msg := controller.NewReturnResult(arg.SessionID, arg.InstanceID, arg.Name, nil, err)
					c.controller.SendChan() <- msg
				}
			},
		}
		ctx.Send(c.store.PID, save)
	}
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
	}
}

func (c *Controller) onReturn(ctx actor.Context, ir *proto.InvokeResponse) {
	splits := strings.SplitN(ir.SessionID, "::", 3)
	name, sessionId, instanceId := splits[0], splits[1], splits[2]
	if ir.Error != "" {
		ctx.Logger().Info("control: execution failed",
			"name", name,
			"session", sessionId,
			"instance", instanceId,
			"err", ir.Error,
		)
		msg := controller.NewReturnResult(sessionId, instanceId, name, nil, errors.New(ir.Error))
		c.controller.SendChan() <- msg
		return
	}

	ctx.Logger().Info("control: execution done",
		"name", name,
		"session", sessionId,
		"instance", instanceId,
	)

	if group, ok := c.groups[name]; ok {
		group.Push(ir.Info)
	}

	msg := controller.NewReturnResult(sessionId, instanceId, name, ir.Result, nil)
	c.controller.SendChan() <- msg
}

func (c *Controller) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *controller.Message:
		c.onControllerMessage(ctx, msg)
	case *proto.InvokeResponse:
		c.onReturn(ctx, msg)
	}
}

func SpawnTaskController(
	ctx *actor.RootContext,
	store *proto.StoreRef,
	venvs *functions.VenvManager,
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
			groups:     make(map[string]*task.ActorGroup),
			runtimes:   make(map[string]*task.Runtime),
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
