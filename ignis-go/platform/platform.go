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
	"github.com/9triver/ignis/monitor"
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
	// Monitor system
	monitor monitor.Monitor
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

				// 检查应用是否已注册
				if info, _ := p.monitor.GetApplicationInfo(ctx, appID); info != nil {
					logrus.Errorf("Application ID %s is conflicted", appID)
					continue
				}

				// 注册应用
				if err := p.monitor.RegisterApplication(ctx, appID, &monitor.ApplicationMetadata{
					AppID: appID,
					Name:  appID,
				}); err != nil {
					logrus.Errorf("Failed to register application %s: %v", appID, err)
					continue
				}

				actorRef := control.SpawnTaskControllerV2(p.sys.Root, appID, p.dp, p.monitor, ctx, ctrlr, func() {
					// 应用关闭时注销
					if err := p.monitor.UnregisterApplication(context.Background(), appID); err != nil {
						logrus.Errorf("Failed to unregister application %s: %v", appID, err)
					}
				})

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

// NewPlatform 创建一个新的 Platform 实例
// mon: 监控实例，如果为 nil 则使用默认的空操作监控
func NewPlatform(ctx context.Context, rpcAddr string, dp task.Deployer, mon monitor.Monitor) *Platform {
	if mon == nil {
		// 使用默认的空操作监控
		mon = monitor.DefaultMonitor()
	}

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
		monitor:             mon,
		controllerActorRefs: make(map[string]*proto.ActorRef),
	}
}

// GetMonitor 返回 Platform 的监控实例
func (p *Platform) GetMonitor() monitor.Monitor {
	return p.monitor
}
