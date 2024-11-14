package worker

import (
	"actors/platform/actor/compute"
	"actors/platform/handlers"
	"actors/platform/system"
	"actors/proto"
	"actors/proto/deploy"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type DeployManager struct {
	actors map[string]*actor.PID
}

func (m *DeployManager) deployPython(ctx actor.Context, name string, cmd *deploy.DeployPython) {
	if _, ok := m.actors[name]; ok {
		ctx.Respond(&proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("actor %s already deployed", name),
		})
		return
	}

	handler, err := handlers.NewPyFunc(name, cmd.Parameters, cmd.Venv, cmd.Packages, cmd.PickledCode)
	if err != nil {
		ctx.Respond(&proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("failed deploy python actor %s: %v", name, err),
		})
		return
	}

	actor := compute.NewActor(handler, system.LatencyNotSpecified)
	pid := ctx.Spawn(actor.Props())
	m.actors[name] = pid

	ctx.Respond(&proto.Ack{
		Sender: ctx.Self(),
	})
}

func (m *DeployManager) onCommand(ctx actor.Context, cmd *deploy.Command) {
	switch dep := cmd.Deploy.(type) {
	case *deploy.Command_Python:
		m.deployPython(ctx, cmd.Name, dep.Python)
	}
}

func (m *DeployManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *deploy.Command:
		m.onCommand(ctx, msg)
	}
}

func NewDeployManager() *actor.PID {
	props := actor.PropsFromProducer(func() actor.Actor {
		return &DeployManager{
			actors: make(map[string]*actor.PID),
		}
	})
	return system.ActorSystem().Root.Spawn(props)
}
