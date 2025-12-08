package control

import (
	"fmt"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions/python"
	"github.com/9triver/ignis/actor/router"
	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/transport"
	"github.com/9triver/ignis/utils/errors"
)

type Controller struct {
	id         string
	manager    *python.VenvManager
	controller transport.Controller
	store      *proto.StoreRef
	appID      string
	appInfo    ApplicationInfo

	nodes    map[string]*task.Node
	groups   map[string]*task.ActorGroup
	runtimes map[string]*task.Runtime

	deployer task.Deployer
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

	// af, err := c.deployer.DeployPyFunc(context.TODO(), c.appID, f)

	// if err != nil {
	// 	ctx.Logger().Error("control: deploy python function error",
	// 		"name", f.Name,
	// 		"err", err,
	// 	)
	// 	return
	// }

	if f.Replicas == 0 {
		f.Replicas = 1
	}

	infos, err := c.deployer.DeployPyFunc(ctx, c.appID, f, c.store)
	if err != nil {
		ctx.Logger().Error("control: deploy python function error",
			"name", f.Name,
			"err", err,
		)
		return
	}

	ctx.Logger().Info("2222")

	group := task.NewGroup(f.Name)
	c.groups[f.Name] = group

	ctx.Logger().Info("33333")

	for _, info := range infos {
		group.Push(info)
	}

	ctx.Logger().Info("11111")

	node := task.NodeFromActorGroup(f.Name, f.Params, group)
	c.nodes[f.Name] = node

	ctx.Logger().Info("44444")
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

func (c *Controller) getOrCreateRuntime(ctx actor.Context, name, sessionId, instanceId string) (*task.Runtime, error) {
	runtimeId := fmt.Sprintf("%s::%s::%s", name, sessionId, instanceId)
	rt, ok := c.runtimes[runtimeId]
	if !ok {
		node, ok := c.nodes[name]
		if !ok {
			return nil, errors.Format("actor node %s not found", name)
		}

		rt = node.Runtime(runtimeId, c.store.PID, c.id)
		c.runtimes[runtimeId] = rt
	}

	return rt, nil
}

func (c *Controller) onAppendArg(ctx actor.Context, arg *controller.AppendArg) {
	ctx.Logger().Info("control: append arg",
		"name", arg.Name,
		"param", arg.Param,
		"session", arg.SessionID,
		"instance", arg.InstanceID,
	)

	rt, err := c.getOrCreateRuntime(ctx, arg.Name, arg.SessionID, arg.InstanceID)
	if err != nil {
		ctx.Logger().Error("control: get runtime error",
			"name", arg.Name,
			"session", arg.SessionID,
			"instance", arg.InstanceID,
			"err", err,
		)
		return
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

func (c *Controller) onInvoke(ctx actor.Context, invoke *controller.Invoke) {
	ctx.Logger().Info("control: invoke",
		"name", invoke.Name,
		"session", invoke.SessionID,
		"instance", invoke.InstanceID,
	)

	rt, err := c.getOrCreateRuntime(ctx, invoke.Name, invoke.SessionID, invoke.InstanceID)
	if err != nil {
		ctx.Logger().Error("control: get runtime error",
			"name", invoke.Name,
			"session", invoke.SessionID,
			"instance", invoke.InstanceID,
			"err", err,
		)
		return
	}

	rt.Start(ctx)
}

func (c *Controller) onRequestObject(ctx actor.Context, requestObject *controller.RequestObject) {
	ctx.Logger().Info("control: request object",
		"id", requestObject.ID,
	)
	store.GetObject(ctx, c.store.PID, &proto.Flow{
		ID: requestObject.ID,
		Source: &proto.StoreRef{
			ID:  c.store.ID,
			PID: c.store.PID,
		},
	}).OnDone(func(obj object.Interface, duration time.Duration, err error) {
		if err != nil {
			ctx.Logger().Error("control: request object error",
				"id", requestObject.ID,
				"err", err,
			)
			return
		}

		encoded, err := obj.Encode()
		if err != nil {
			ctx.Logger().Error("control: encode object error",
				"id", requestObject.ID,
				"err", err,
			)
		}
		msg := controller.NewResponseObject(requestObject.ID, encoded, err)
		c.controller.SendChan() <- msg
	})
}

func (c *Controller) onControllerMessage(ctx actor.Context, msg *controller.Message) {
	switch cmd := msg.Command.(type) {
	case *controller.Message_AppendPyFunc:
		c.onAppendPyFunc(ctx, cmd.AppendPyFunc)
	case *controller.Message_AppendData:
		c.onAppendData(ctx, cmd.AppendData)
	case *controller.Message_AppendArg:
		c.onAppendArg(ctx, cmd.AppendArg)
	case *controller.Message_Invoke:
		c.onInvoke(ctx, cmd.Invoke)
	case *controller.Message_RequestObject:
		c.onRequestObject(ctx, cmd.RequestObject)
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

	// Automatically mark the corresponding ControlNode and its output DataNode as done when task execution completes
	if c.appInfo != nil {
		// Find the node IDs by function name
		controlNodeID, outputDataNodeID := c.findNodeIDsByFunctionName(name)
		if controlNodeID != "" {
			ctx.Logger().Info("control: auto-marking ControlNode done",
				"functionName", name,
				"controlNodeID", controlNodeID,
				"sessionId", sessionId,
			)
			c.appInfo.MarkNodeDone(controlNodeID)

			// Also mark the output DataNode as done if it exists
			if outputDataNodeID != "" {
				ctx.Logger().Info("control: auto-marking output DataNode done",
					"functionName", name,
					"outputDataNodeID", outputDataNodeID,
					"sessionId", sessionId,
				)
				c.appInfo.MarkNodeDone(outputDataNodeID)
			}
		} else {
			ctx.Logger().Warn("control: could not find node ID for function",
				"functionName", name,
				"sessionId", sessionId,
			)
		}
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
	venvs *python.VenvManager,
	cm transport.ControllerManager,
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

type ApplicationInfo interface {
	GetDAG() *controller.DAG
	SetDAG(dag *controller.DAG)
	MarkNodeDone(nodeID string)
}

// For iarnet
func SpawnTaskControllerV2(ctx *actor.RootContext, appID string, deployer task.Deployer,
	appInfo ApplicationInfo, c transport.Controller, onClose func()) *proto.ActorRef {

	store := store.Spawn(ctx, nil, "store-"+appID)

	controllerId := "controller-" + appID
	props := actor.PropsFromProducer(func() actor.Actor {
		return &Controller{
			id:         controllerId,
			controller: c,
			appID:      appID,
			appInfo:    appInfo,
			deployer:   deployer,
			store:      store,
			nodes:      make(map[string]*task.Node),
			groups:     make(map[string]*task.ActorGroup),
			runtimes:   make(map[string]*task.Runtime),
		}
	})

	pid, _ := ctx.SpawnNamed(props, controllerId)
	logrus.Infof("control: spawn controller %s with pid %s", controllerId, pid)

	go func() {
		defer onClose()
		for msg := range c.RecvChan() {
			ctx.Send(pid, msg)
		}
	}()

	router.Register(controllerId, pid)

	return &proto.ActorRef{
		// Store: store,
		ID:  controllerId,
		PID: pid,
	}
}

// findNodeIDsByFunctionName finds the ControlNode ID and its output DataNode ID by function name
func (c *Controller) findNodeIDsByFunctionName(functionName string) (controlNodeID, outputDataNodeID string) {
	if c.appInfo == nil {
		return "", ""
	}

	dag := c.appInfo.GetDAG()
	if dag == nil {
		return "", ""
	}

	// Find the ControlNode with the matching function name
	for _, node := range dag.Nodes {
		if node.Type == "ControlNode" && node.GetControlNode() != nil {
			controlNode := node.GetControlNode()
			if controlNode.FunctionName == functionName {
				controlNodeID = controlNode.Id
				outputDataNodeID = controlNode.DataNode // This is the output DataNode ID
				return controlNodeID, outputDataNodeID
			}
		}
	}

	return "", ""
}
