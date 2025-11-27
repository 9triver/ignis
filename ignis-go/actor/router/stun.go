package router

import (
	"errors"

	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/transport/stun"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/pion/logging"
	pb "google.golang.org/protobuf/proto"
)

type STUNRouter struct {
	baseRouter
	sender stun.Sender
	source string
	peers  []string
}

func (r *STUNRouter) Send(targetId string, msg any) {
	// find local
	if pid, ok := r.routeTable[targetId]; ok {
		r.ctx.Send(pid, msg)
		return
	}

	// if not found, broadcast to all remote peers
	pbMsg, ok := msg.(pb.Message)
	if !ok {
		r.ctx.Logger().Error(
			"router: forward to peer failed",
			"router", "stun",
			"err", errors.New("msg not serializable"),
		)
		return
	}

	envelope := cluster.NewEnvelope(targetId, pbMsg)
	data, err := pb.Marshal(envelope)
	if err != nil {
		r.ctx.Logger().Error(
			"router: forward to peer failed",
			"router", "stun",
			"err", err,
		)
	}

	for _, peer := range r.peers {
		r.sender.Send(peer, data)
	}
}

func (r *STUNRouter) handleEnvelope(envelope *Envelope) {
	targetId := envelope.Target
	pid, ok := r.routeTable[targetId]
	if !ok {
		return
	}

	var send any
	switch msg := envelope.Message.(type) {
	case *cluster.Envelope_ObjectRequest:
		send = msg.ObjectRequest
	case *cluster.Envelope_ObjectResponse:
		send = msg.ObjectResponse
	case *cluster.Envelope_StreamChunk:
		send = msg.StreamChunk
	default:
		return
	}

	r.ctx.Send(pid, send)
}

type Envelope = cluster.Envelope

func NewSTUNRouter(
	ctx Context,
	source, signalingServer string,
	peers ...string,
) *STUNRouter {
	router := &STUNRouter{
		baseRouter: baseRouter{
			ctx:        ctx,
			routeTable: make(map[string]*actor.PID),
		},
		source: source,
		peers:  peers,
	}

	onMessage := func(data []byte) {
		envelope := &Envelope{}
		if err := pb.Unmarshal(data, envelope); err != nil {
			return
		}
		router.handleEnvelope(envelope)
	}

	sender := stun.NewNatSender(source, signalingServer, onMessage, logging.LogLevelWarn)

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
