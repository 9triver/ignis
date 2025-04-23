package remote

import (
	"context"

	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
)

type Protocol int

const (
	IPC Protocol = iota
	RPC
	Websocket
)

type Stream[I, O pb.Message] interface {
	Conn() string
	SendChan() chan<- I
	RecvChan() <-chan O
	Ready() <-chan struct{}
	Protocol() Protocol
}

type Executor Stream[*executor.Message, *executor.Message]

type Controller Stream[*controller.Message, *controller.Message]

type Manager interface {
	Addr() string
	Run(ctx context.Context) error
	Type() Protocol
}

type ExecutorManager interface {
	Manager
	NewExecutor(ctx context.Context, conn string) Executor
}

type ControllerManager interface {
	Manager
	NextController() Controller
}
