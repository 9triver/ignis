package rpc

import (
	"context"
	"github.com/9triver/ignis/proto/executor"
	"google.golang.org/grpc"
	"net"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/proto/controller"
)

type ConnectionManager struct {
	addr string
	cs   *controllerService
	es   *executorService
}

func (cm *ConnectionManager) Addr() string {
	return cm.addr
}

func (cm *ConnectionManager) Run(ctx context.Context) error {
	defer cm.cs.close()
	defer cm.es.close()

	lis, err := net.Listen("tcp", cm.addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()

	controller.RegisterServiceServer(server, cm.cs)
	executor.RegisterServiceServer(server, cm.es)

	ech := make(chan error)
	go func() {
		ech <- server.Serve(lis)
		close(ech)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ech:
		return err
	}
}

func (cm *ConnectionManager) NextController() remote.Controller {
	return cm.cs.nextConn()
}

func (cm *ConnectionManager) NewExecutor(ctx context.Context, conn string) remote.Executor {
	return cm.es.newConn(ctx, conn)
}

func (cm *ConnectionManager) Type() remote.Protocol {
	return remote.RPC
}

func NewManager(addr string) *ConnectionManager {
	return &ConnectionManager{
		addr: addr,
		cs: &controllerService{
			controllers: make(map[string]*remote.ControllerImpl),
			next:        make(chan *remote.ControllerImpl),
		},
		es: &executorService{executors: make(map[string]*remote.ExecutorImpl)},
	}
}
