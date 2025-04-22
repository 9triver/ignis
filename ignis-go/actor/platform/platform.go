package platform

import (
	"context"
	"log/slog"
	"path"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/actor/remote/ipc"
	"github.com/9triver/ignis/actor/remote/rpc"
	"github.com/9triver/ignis/configs"
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
}

func (p *Platform) Run() error {
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	go func() {
		if err := p.cm.Run(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := p.em.Run(ctx); err != nil {
			panic(err)
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

func NewPlatform(ctx context.Context) *Platform {
	opt := actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {
		return utils.Logger()
	})
	rpcAddr := "127.0.0.1:8080"
	ipcAddr := "ipc://" + path.Join(configs.StoragePath, "em-ipc")

	return &Platform{
		ctx: ctx,
		sys: actor.NewActorSystem(opt),
		cm:  rpc.NewManager(rpcAddr),
		em:  ipc.NewManager(ipcAddr),
	}
}
