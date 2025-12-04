package router

import (
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
)

type TCPRouter struct {
	baseRouter
	bootstrap, listener *actor.PID
}

func (r *TCPRouter) Register(store *proto.StoreRef) {
	r.baseRouter.Register(store)
	r.ctx.Logger().Info("router: register bootstrap", "store", store, "bootstrap", r.bootstrap)
	r.ctx.Send(r.bootstrap, store)
}

func NewTCPRouter(ctx Context, bootstrap *actor.PID, host string, port int) *TCPRouter {
	router := &TCPRouter{
		baseRouter: makeBaseRouter(ctx),
	}

	remoter := remote.NewRemote(ctx.ActorSystem(), remote.Configure(host, port))

	remoter.Start()

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *proto.StoreRef:
			c.Logger().Info("bootstrap: register peer", "id", msg.ID)
			router.routes[msg.ID] = msg.PID
		}
	})

	listener, _ := ctx.SpawnNamed(props, "listener")
	router.listener = listener

	if bootstrap == nil {
		bootstrap, _ = ctx.SpawnNamed(actor.PropsFromProducer(func() actor.Actor {
			return &BootstrapActor{}
		}), "bootstrap")
	}

	router.bootstrap = bootstrap
	return router
}

type BootstrapActor struct {
	listeners  []*actor.PID
	candidates []*proto.StoreRef
}

func (a *BootstrapActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.StoreRef:
		ctx.Logger().Info("bootstrap: append candidate", "id", msg.PID)
		a.candidates = append(a.candidates, msg)
		a.listeners = append(a.listeners, actor.NewPID(msg.PID.Address, "listener"))
		for _, pid := range a.listeners {
			for _, candidate := range a.candidates {
				if candidate.PID.Address != pid.Address {
					ctx.Logger().Info("bootstrap: forward candidate", "target", pid, "id", candidate.ID)
					ctx.Send(pid, candidate)
				}
			}
		}
	}
}
