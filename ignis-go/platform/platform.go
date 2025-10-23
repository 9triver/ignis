package platform

import (
	"context"
	"path"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/platform/control"
	"github.com/9triver/ignis/platform/task"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/utils"
)

type Platform struct {
	// Root context of platform
	ctx context.Context
	// Main actor system of the platform
	sys *actor.ActorSystem
	// Control connection manager
	cm remote.ControllerManager
	// Executor connection manager
	em remote.ExecutorManager
	// Deployer
	dp task.Deployer
	// Application infos
	appInfos map[string]*ApplicationInfo
	// Controller actor refs
	controllerActorRefs map[string]*proto.ActorRef
}

func (p *Platform) Run() error {
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	go func() {
		logrus.Infof("Controller Manager listening on %s", p.cm.Addr())
		if err := p.cm.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		logrus.Infof("Executor Manager listening on %s", p.em.Addr())
		if err := p.em.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		logrus.Info("Waiting for new client connection...")
		for {
			ctrlr := p.cm.NextController()
			if ctrlr == nil {
				continue
			}
			logrus.Info("New client is connected")
			msg := <-ctrlr.RecvChan()
			if msg.Type == controller.CommandType_FR_REGISTER_REQUEST {
				req := msg.GetRegisterRequest()
				if req == nil {
					logrus.Error("Register request is nil")
					continue
				}
				appID := req.GetApplicationID()
				if _, ok := p.appInfos[appID]; ok {
					logrus.Errorf("Application ID %s is conflicted", appID)
					continue
				}
				appInfo := &ApplicationInfo{ID: appID}
				actorRef := control.SpawnTaskControllerV2(p.sys.Root, appID, p.dp, appInfo, ctrlr, func() {
					delete(p.appInfos, appID)
				})
				p.appInfos[appID] = appInfo
				p.controllerActorRefs[appID] = actorRef
				logrus.Infof("Application %s is registered", appID)

				ack := controller.NewAck(nil)
				ctrlr.SendChan() <- ack
			} else {
				logrus.Errorf("The first message %s is not register request", msg.Type)
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

func NewPlatform(ctx context.Context, rpcAddr string, dp task.Deployer) *Platform {
	opt := utils.WithLogger()
	ipcAddr := "ipc://" + path.Join(configs.StoragePath, "em-ipc")

	sys := actor.NewActorSystem(opt)
	em := ipc.NewManager(ipcAddr)

	if dp == nil {
		vm, err := functions.NewVenvManager(ctx, em)
		if err != nil {
			panic(err)
		}

		dp = task.NewVenvMgrDeployer(vm)
	}

	return &Platform{
		ctx:                 ctx,
		sys:                 sys,
		cm:                  rpc.NewManager(rpcAddr),
		em:                  em,
		dp:                  dp,
		appInfos:            make(map[string]*ApplicationInfo),
		controllerActorRefs: make(map[string]*proto.ActorRef),
	}
}

// GetControllerActorRef returns the actor ref of the controller of the application.
func (p *Platform) GetApplicationDAG(appID string) *controller.DAG {
	logrus.Infof("GetApplicationDAG: %s", appID)
	logrus.Infof("appInfos: %v", p.appInfos)
	if _, ok := p.appInfos[appID]; !ok {
		return nil
	}
	return p.appInfos[appID].GetDAG()
}

type ApplicationInfo struct {
	ID  string
	dag *controller.DAG
}

// SetDAG sets the DAG of the current application.
func (a *ApplicationInfo) SetDAG(dag *controller.DAG) {
	a.dag = dag
}

// GetDAG returns the DAG of the current application.
func (a *ApplicationInfo) GetDAG() *controller.DAG {
	return a.dag
}
