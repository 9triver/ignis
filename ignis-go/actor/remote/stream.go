package remote

import (
	"context"
	"time"

	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto/controller"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils/errors"
)

type StreamImpl[I, O pb.Message] struct {
	conn     string
	protocol Protocol
	sender   func(msg I) error
	ready    chan struct{}
	send     chan I
	recv     chan O
}

func (s *StreamImpl[I, O]) Conn() string {
	return s.conn
}

func (s *StreamImpl[I, O]) SendChan() chan<- I {
	return s.send
}

func (s *StreamImpl[I, O]) RecvChan() <-chan O {
	return s.recv
}

func (s *StreamImpl[I, O]) Ready() <-chan struct{} {
	return s.ready
}

func (s *StreamImpl[I, O]) Protocol() Protocol {
	return s.protocol
}

func (s *StreamImpl[I, O]) SetSender(sender func(msg I) error) {
	if s.sender != nil {
		return
	}
	s.sender = sender
	close(s.ready)
}

func (s *StreamImpl[I, O]) Produce(msg O) {
	s.recv <- msg
}

func (s *StreamImpl[I, O]) Run(ctx context.Context) error {
	select {
	case <-time.After(30 * time.Second):
		return errors.New("connection timeout")
	case <-s.Ready():
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-s.send:
			if err := s.sender(msg); err != nil {
				return err
			}
		}
	}
}

func (s *StreamImpl[I, O]) Close() error {
	if s.recv == nil {
		return errors.New("close of a closed stream")
	}
	close(s.recv)
	s.recv = nil
	return nil
}

func newStreamImpl[I, O pb.Message](conn string, protocol Protocol) *StreamImpl[I, O] {
	return &StreamImpl[I, O]{
		conn:     conn,
		protocol: protocol,
		sender:   nil,
		ready:    make(chan struct{}),
		send:     make(chan I, configs.ChannelBufferSize),
		recv:     make(chan O, configs.ChannelBufferSize),
	}
}

type ExecutorImpl = StreamImpl[*executor.Message, *executor.Message]

type ControllerImpl = StreamImpl[*controller.Message, *controller.Message]

func NewExecutorImpl(conn string, protocol Protocol) *ExecutorImpl {
	return newStreamImpl[*executor.Message, *executor.Message](conn, protocol)
}

func NewControllerImpl(conn string, protocol Protocol) *ControllerImpl {
	return newStreamImpl[*controller.Message, *controller.Message](conn, protocol)
}
