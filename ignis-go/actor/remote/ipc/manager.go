package ipc

import (
	"context"
	_ "embed"

	pb "google.golang.org/protobuf/proto"
	"gopkg.in/zeromq/goczmq.v4"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/proto/executor"
)

type ConnectionManager struct {
	addr      string
	executors map[string]*remote.ExecutorImpl
}

func (cm *ConnectionManager) Addr() string {
	return cm.addr
}

func (cm *ConnectionManager) onReceive(router *goczmq.Channeler, frame []byte, msg *executor.Message) {
	conn := msg.Conn
	e, ok := cm.executors[conn]
	if !ok {
		return
	}

	switch msg.Command.(type) {
	case *executor.Message_Ready:
		e.SetSender(func(msg *executor.Message) error {
			data, err := pb.Marshal(msg)
			if err != nil {
				return err
			}
			router.SendChan <- [][]byte{frame, data}
			return nil
		})
	case *executor.Message_Return:
		e.Produce(msg)
	}
}

func (cm *ConnectionManager) Run(ctx context.Context) error {
	router := goczmq.NewRouterChanneler(cm.addr)
	defer router.Destroy()

	defer func() {
		for _, e := range cm.executors {
			_ = e.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-router.RecvChan:
			if len(msg) < 2 {
				continue
			}
			frame, data := msg[0], msg[1]
			cmd := &executor.Message{}
			if err := pb.Unmarshal(data, cmd); err != nil {
				continue
			}
			cm.onReceive(router, frame, cmd)
		}
	}
}

func (cm *ConnectionManager) NewExecutor(ctx context.Context, conn string) remote.Executor {
	if e, ok := cm.executors[conn]; ok {
		return e
	}

	e := remote.NewExecutorImpl(conn, remote.IPC)
	go e.Run(ctx)
	cm.executors[conn] = e
	return e
}

func (cm *ConnectionManager) Type() remote.Protocol {
	return remote.IPC
}

func NewManager(addr string) *ConnectionManager {
	return &ConnectionManager{
		addr:      addr,
		executors: make(map[string]*remote.ExecutorImpl),
	}
}

var (
	//go:embed executor_template.py
	PythonExecutorTemplate string
)
