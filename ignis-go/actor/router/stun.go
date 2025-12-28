package router

import (
	"errors"

	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/transport/stun"
	"github.com/pion/logging"
	pb "google.golang.org/protobuf/proto"
)

type STUNRouter struct {
	baseRouter
	sender stun.Sender
	source string
}

func (r *STUNRouter) Send(targetId string, msg any) {
	r.mu.RLock()
	pid, ok := r.routes[targetId]
	r.mu.RUnlock()

	// find local
	if ok {
		r.ctx.Send(pid, msg)
		return
	}

	// if not found, broadcast to remote peer
	pbMsg, ok := msg.(pb.Message)
	if !ok {
		r.ctx.Logger().Error(
			"router: forward to peer failed",
			"router", "stun",
			"err", errors.New("msg not serializable"),
		)
		return
	}

	envelope := cluster.NewEnvelope(r.source, targetId, pbMsg)
	data, err := pb.Marshal(envelope)
	if err != nil {
		r.ctx.Logger().Error(
			"router: forward to peer failed",
			"router", "stun",
			"err", err,
		)
	}

	r.ctx.Logger().Info(
		"router: forwarding message",
		"target", targetId,
	)
	r.sender.Send(targetId, data)
}

func (r *STUNRouter) handleEnvelope(envelope *cluster.Envelope) {
	targetId := envelope.Target
	pid, ok := r.routes[targetId]
	if !ok {
		return
	}

	r.ctx.Logger().Info(
		"router: receive message",
		"target", targetId,
	)

	var send any
	switch msg := envelope.Message.(type) {
	case *cluster.Envelope_ObjectRequest:
		send = msg.ObjectRequest
	case *cluster.Envelope_ObjectResponse:
		send = msg.ObjectResponse
	case *cluster.Envelope_StreamChunk:
		send = msg.StreamChunk
	default:
		r.ctx.Logger().Info(
			"router: receive message",
			"target", targetId,
		)
		return
	}

	r.ctx.Send(pid, send)
}

func NewSTUNRouter(ctx Context, source, signalingServer string) *STUNRouter {
	router := &STUNRouter{
		baseRouter: makeBaseRouter(ctx),
		source:     source,
	}

	onMessage := func(data []byte) {
		envelope := &cluster.Envelope{}
		if err := pb.Unmarshal(data, envelope); err != nil {
			return
		}
		router.handleEnvelope(envelope)
	}

	sender := stun.NewNatSender(source, signalingServer, onMessage, logging.LogLevelError)

	sender.Listen(nil)
	go func() {
		for {
			select {
			case <-sender.Ctx.Done():
				return
			default:
			}
			sender.Accept()
		}
	}()

	router.sender = sender
	return router
}
