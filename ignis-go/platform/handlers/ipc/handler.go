package ipc

import (
	"actors/platform/store"
	"actors/platform/system"
	"actors/platform/utils"
	"actors/proto/ipc"
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"gopkg.in/zeromq/goczmq.v4"
)

type ipcSend struct {
	frame   []byte
	msg     *ipc.RouterMessage
	onError func(error)
}

type Handler struct {
	ctx     context.Context
	send    chan ipcSend
	addr    string
	routees map[string][]byte
	ready   map[string]chan error
	pending map[uint64]*utils.Future[*ipc.Return]
	timeout time.Duration
}

func (h *Handler) onReceive(frame []byte, cmd *ipc.DealerMessage) {
	switch c := cmd.Payload.(type) {
	case *ipc.DealerMessage_Ready:
		h.routees[c.Ready.Name] = frame
		close(h.ready[c.Ready.Name])
		delete(h.ready, c.Ready.Name)
	case *ipc.DealerMessage_Return:
		h.Resolve(c.Return)
	}
}

func (h *Handler) Register(name string) chan error {
	h.ready[name] = make(chan error)
	return h.ready[name]
}

func (h *Handler) Run() <-chan error {
	errors := make(chan error, system.ChannelBufferSize)
	go func() {
		router := goczmq.NewRouterChanneler(h.addr)
		defer router.Destroy()
		defer close(errors)

		sendIPCCall := func(send ipcSend) {
			data, err := proto.Marshal(send.msg)
			if err != nil {
				if send.onError != nil {
					send.onError(err)
				}
				errors <- err
				return
			}
			router.SendChan <- [][]byte{send.frame, data}
		}

		for {
			select {
			case <-h.ctx.Done():
				for _, routee := range h.routees {
					sendIPCCall(ipcSend{
						frame: routee,
						msg:   MakeRouterMessage(MakeExit()),
					})
				}
				return
			case msg := <-router.RecvChan:
				if len(msg) < 2 {
					continue
				}
				routerFrame, data := msg[0], msg[1]
				cmd := &ipc.DealerMessage{}
				if err := proto.Unmarshal(data, cmd); err != nil {
					errors <- err
					continue
				}

				h.onReceive(routerFrame, cmd)
			case send := <-h.send:
				sendIPCCall(send)
			}
		}
	}()

	return errors
}

func (h *Handler) Addr() string {
	return h.addr
}

func (h *Handler) Tell(routee string, cmd *ipc.RouterMessage) error {
	frame, ok := h.routees[routee]
	if !ok {
		return fmt.Errorf("tell: unknown venv %s", routee)
	}

	call := ipcSend{
		frame: frame,
		msg:   cmd,
		onError: func(err error) {
			fmt.Println("tell error", err)
		},
	}
	h.send <- call
	return nil
}

func (h *Handler) Request(routee string, cmd *ipc.Execute) (*store.Object, error) {
	frame, ok := h.routees[routee]
	if !ok {
		return nil, fmt.Errorf("request %d: unknown venv %s", cmd.ID, routee)
	}

	fut := utils.NewFuture[*ipc.Return](h.timeout)
	h.pending[cmd.GetID()] = fut
	defer delete(h.pending, cmd.GetID())

	call := ipcSend{
		frame:   frame,
		msg:     MakeRouterMessage(cmd),
		onError: func(err error) { fut.Reject(err) },
	}
	h.send <- call

	ret, err := fut.Result()
	if err != nil {
		return nil, fmt.Errorf("request %d: %w", cmd.GetID(), err)
	}

	switch c := ret.Result.(type) {
	case *ipc.Return_Error:
		return nil, fmt.Errorf("request %d: %s", cmd.GetID(), c.Error)
	case *ipc.Return_Value:
		obj := &store.Object{}
		if err := obj.Decode(c.Value); err != nil {
			return nil, fmt.Errorf("request %d: %w", cmd.GetID(), err)
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("request %d: unknown return type", cmd.GetID())
	}
}

func (h *Handler) Resolve(response *ipc.Return) {
	fut, ok := h.pending[response.GetID()]
	if !ok {
		return
	}
	fut.Resolve(response)
}

func NewHandler(ctx context.Context, addr string, timeout time.Duration) *Handler {
	h := &Handler{
		ctx:     ctx,
		send:    make(chan ipcSend),
		addr:    addr,
		routees: make(map[string][]byte),
		ready:   make(map[string]chan error),
		pending: make(map[uint64]*utils.Future[*ipc.Return]),
		timeout: timeout,
	}
	return h
}
