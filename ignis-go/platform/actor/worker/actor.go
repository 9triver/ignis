package worker

import (
	"actors/platform/store"
	"actors/platform/system"
	"actors/platform/utils"
	"actors/proto"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type Actor struct {
	name          string
	store         *store.Store
	head          *actor.PID
	pid           *actor.PID
	deployManager *actor.PID
}

func (node *Actor) SetPID(pid *actor.PID) {
	node.pid = pid
	if node.head == nil {
		node.head = pid
	}
}

func (node *Actor) Name() string {
	return node.name
}

func (node *Actor) Head() *actor.PID {
	return node.head
}

func (node *Actor) Self() *actor.PID {
	return node.pid
}

func (node *Actor) Store() *store.Store {
	return node.store
}

func (node *Actor) DeployManager() *actor.PID {
	return node.deployManager
}

func (node *Actor) onStarted(ctx actor.Context) {
	ctx.Send(node.head, &proto.WorkerRegister{
		Sender: ctx.Self(),
	})
}

func (node *Actor) OnLinkPing(ctx actor.Context, ping *proto.LinkPing) {
	pong := &proto.LinkPong{
		SchedulerTime: ping.SchedulerTime,
		Scheduler:     ping.Scheduler,
		Source:        ctx.Self(),
		DummyPayload:  nil,
	}
	ctx.Send(ping.Target, pong)
}

func (node *Actor) OnLinkPong(ctx actor.Context, pong *proto.LinkPong) {
	res := &proto.LinkResponse{
		SchedulerTime: pong.SchedulerTime,
		Source:        pong.Source,
		Target:        ctx.Self(),
	}
	ctx.Send(pong.Scheduler, res)
}

func (node *Actor) OnObjectRequest(ctx actor.Context, req *proto.ObjectRequest) {
	obj, ok := node.store.Get(req.ID)
	if !ok {
		ctx.Respond(&proto.Error{
			Message: "object not found",
			Sender:  req.Sender,
		})
		return
	}

	if utils.IsSameSystem(ctx.Self(), req.Sender) {
		ctx.Respond(obj)
		return
	}

	encoded, err := obj.Encode()
	if err != nil {
		ctx.Respond(&proto.Error{
			Message: "error encoding object: " + err.Error(),
			Sender:  req.Sender,
		})
		return
	}

	ctx.Respond(encoded)
}

func (node *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		node.onStarted(ctx)
	case *proto.LinkPing:
		node.OnLinkPing(ctx, msg)
	case *proto.LinkPong:
		node.OnLinkPong(ctx, msg)
	case *proto.ActorReady, *proto.ActorStopped:
		ctx.Forward(node.head)
	case *proto.ObjectRequest:
		node.OnObjectRequest(ctx, msg)
	}
}

func NewActor(head *actor.PID) *Actor {
	return &Actor{
		store:         store.New(),
		head:          head,
		deployManager: NewDeployManager(),
	}
}

func Start(localAddr, remoteAddr string, remotePort int) error {
	sys := system.ActorSystem()
	logger := sys.Logger()

	headWorker := actor.NewPID(fmt.Sprintf("%s:%d", remoteAddr, remotePort), system.SystemName)

	worker := NewActor(headWorker)
	pid, err := system.Start(worker, remoteAddr, 0)
	if err != nil {
		return fmt.Errorf("failed to start worker: %v", err)
	}
	worker.SetPID(pid)

	logger.Info("Worker connecting", "local", worker.Self(), "head", worker.Head())
	return nil
}
