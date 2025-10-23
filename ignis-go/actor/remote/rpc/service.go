package rpc

import (
	"context"
	"io"

	"google.golang.org/grpc"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils"
	"github.com/sirupsen/logrus"
)

type controllerService struct {
	controller.UnimplementedServiceServer
	controllers map[string]*remote.ControllerImpl
	next        chan *remote.ControllerImpl
}

func (cs *controllerService) Session(stream grpc.BidiStreamingServer[controller.Message, controller.Message]) error {
	conn := utils.GenID()
	c := cs.newConn(conn)
	c.SetSender(stream.Send)
	cs.next <- c
	defer c.Close()

	go c.Run(stream.Context())

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.Produce(msg)
	}
}

func (cs *controllerService) newConn(conn string) *remote.ControllerImpl {
	if c, ok := cs.controllers[conn]; ok {
		return c
	}

	c := remote.NewControllerImpl(conn, remote.RPC)
	cs.controllers[conn] = c
	return c
}

func (cs *controllerService) nextConn() *remote.ControllerImpl {
	return <-cs.next
}

func (cs *controllerService) close() {
	close(cs.next)
}

type executorService struct {
	executor.UnimplementedServiceServer
	executors map[string]*remote.ExecutorImpl
}

func (es *executorService) Session(stream grpc.BidiStreamingServer[executor.Message, executor.Message]) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		es.onReceive(stream, msg)
	}
}

func (es *executorService) onReceive(stream grpc.BidiStreamingServer[executor.Message, executor.Message], msg *executor.Message) {
	c, ok := es.executors[msg.Conn]
	if !ok {
		return
	}

	switch msg.Command.(type) {
	case *executor.Message_Ready:
		c.SetSender(stream.Send)
	default:
		c.Produce(msg)
	}
}

func (es *executorService) newConn(ctx context.Context, conn string) remote.Executor {
	if c, ok := es.executors[conn]; ok {
		return c
	}

	c := remote.NewExecutorImpl(conn, remote.RPC)
	go c.Run(ctx)
	es.executors[conn] = c
	return c
}

func (es *executorService) close() {
	for _, c := range es.executors {
		_ = c.Close()
	}
}

type computeService struct {
	cluster.UnimplementedServiceServer
	computers map[string]*remote.ComputeStreamImpl
}

func (cps *computeService) Session(stream grpc.BidiStreamingServer[cluster.Message, cluster.Message]) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		cps.onReceive(stream, msg)
	}
}

func (cps *computeService) onReceive(stream grpc.BidiStreamingServer[cluster.Message, cluster.Message], msg *cluster.Message) {
	c, ok := cps.computers[msg.ConnID]
	if !ok {
		logrus.Errorf("compute session %s not found", msg.ConnID)
		return
	}

	switch msg.Message.(type) {
	case *cluster.Message_Ready:
		c.SetSender(stream.Send)
	default:
		c.Produce(msg)
	}
}

func (cps *computeService) newConn(ctx context.Context, connId string) *remote.ComputeStreamImpl {
	if c, ok := cps.computers[connId]; ok {
		return c
	}

	c := remote.NewComputeStreamImpl(connId, remote.RPC)
	go c.Run(ctx)
	cps.computers[connId] = c
	return c
}

func (cs *computeService) close() {
	for _, c := range cs.computers {
		_ = c.Close()
	}
}
