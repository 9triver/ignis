package store

import (
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/cluster"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type Actor struct {
	id   string
	ref  *proto.StoreRef
	stub remote.Stub
	// localObjects saves objects generated from current system
	localObjects map[string]messages.Object
	// remoteObjects saves objects from remote stores
	remoteObjects map[string]utils.Future[messages.Object]
}

func (s *Actor) onObjectRequest(ctx actor.Context, req *cluster.ObjectRequest) {
	ctx.Logger().Info("store: responding object",
		"id", req.ID,
		"replyTo", req.ReplyTo.ID,
	)

	reply := &cluster.ObjectResponse{ID: req.ID}
	obj, ok := s.localObjects[req.ID]
	if !ok {
		reply.Error = errors.Format("store: object %s not found", req.ID).Error()
	} else if encoded, err := obj.GetEncoded(); err != nil {
		reply.Error = errors.WrapWith(err, "store: object %s encode failed", req.ID).Error()
	} else {
		reply.Value = encoded
	}
	s.stub.SendTo(req.ReplyTo, reply)

	// if requested object is a stream, send all chunks
	if stream, ok := obj.(*messages.LocalStream); ok {
		go func() {
			defer s.stub.SendTo(req.ReplyTo, proto.NewStreamEnd(req.ID))
			objects := stream.ToChan()
			for obj := range objects {
				encoded, err := obj.GetEncoded()
				msg := proto.NewStreamChunk(req.ID, encoded, err)
				s.stub.SendTo(req.ReplyTo, msg)
			}
		}()
	}
}

func (s *Actor) onObjectResponse(ctx actor.Context, resp *cluster.ObjectResponse) {
	if resp.Error != "" {
		ctx.Logger().Error("store: object response error",
			"id", resp.ID,
			"error", resp.Error,
		)
		return
	}

	ctx.Logger().Info("store: receive object response",
		"id", resp.ID,
	)

	fut, ok := s.remoteObjects[resp.ID]
	if !ok {
		return
	}

	encoded := resp.Value
	if !encoded.Stream {
		fut.Resolve(encoded)
	} else {
		ls := messages.NewLocalStream(nil, encoded.Language)
		fut.Resolve(ls)
	}
}

func (s *Actor) onStreamChunk(ctx actor.Context, chunk *proto.StreamChunk) {
	ctx.Logger().Info("store: receive stream chunk",
		"stream", chunk.StreamID,
		"isEos", chunk.EoS,
	)
	fut, ok := s.remoteObjects[chunk.StreamID]
	if !ok {
		return
	}

	obj, err := fut.Result()
	if err != nil {
		return
	}

	stream, ok := obj.(*messages.LocalStream)
	if !ok {
		return
	}

	if chunk.EoS {
		stream.EnqueueChunk(nil)
	} else {
		stream.EnqueueChunk(chunk.Value)
	}
}

func (s *Actor) onSaveObject(ctx actor.Context, save *messages.SaveObject) {
	obj := save.Value
	ctx.Logger().Info("store: save object",
		"id", obj.GetID(),
	)

	s.localObjects[obj.GetID()] = obj

	if save.Callback != nil {
		save.Callback(ctx, &proto.Flow{ObjectID: obj.GetID(), Source: s.ref})
	}
}

func (s *Actor) getLocalObject(flow *proto.Flow) (messages.Object, error) {
	obj, ok := s.localObjects[flow.ObjectID]
	if !ok {
		return nil, errors.Format("store: flow %s not found", flow.ObjectID)
	}
	return obj, nil
}

func (s *Actor) requestRemoteObject(flow *proto.Flow) utils.Future[messages.Object] {
	if fut, ok := s.remoteObjects[flow.ObjectID]; ok {
		return fut
	}

	fut := utils.NewFuture[messages.Object](configs.FlowTimeout)
	s.remoteObjects[flow.ObjectID] = fut

	remoteRef := flow.Source
	s.stub.SendTo(remoteRef, &cluster.ObjectRequest{
		ID:      flow.ObjectID,
		ReplyTo: s.ref,
	})
	return fut
}

func (s *Actor) onFlowRequest(ctx actor.Context, req *messages.RequestObject) {
	ctx.Logger().Info("store: local flow request",
		"id", req.Flow.ObjectID,
		"store", req.Flow.Source.ID,
	)

	if req.Flow.Source.ID == s.ref.ID {
		obj, err := s.getLocalObject(req.Flow)
		ctx.Send(req.ReplyTo, &messages.ObjectResponse{
			Value: obj,
			Error: err,
		})
	} else {
		s.requestRemoteObject(req.Flow).OnDone(func(obj messages.Object, err error) {
			ctx.Send(req.ReplyTo, &messages.ObjectResponse{
				Value: obj,
				Error: err,
			})
		})
	}
}

func (s *Actor) onForward(ctx actor.Context, forward messages.ForwardMessage) {
	target := forward.GetTarget()
	if target.Store.Equals(s.ref) { // target is local
		ctx.Send(target.PID, forward)
	} else {
		ctx.Logger().Info("store: forwarding request")
		s.stub.SendTo(target.Store, forward)
	}
}

func (s *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	// Requests from remote stores
	case *cluster.ObjectRequest:
		s.onObjectRequest(ctx, msg)
	// Responses from remote stores
	case *cluster.ObjectResponse:
		s.onObjectResponse(ctx, msg)
	case *cluster.StreamChunk:
		s.onStreamChunk(ctx, msg)
	// Local actor requests
	case *messages.SaveObject:
		s.onSaveObject(ctx, msg)
	case *messages.RequestObject:
		s.onFlowRequest(ctx, msg)
	// forward messages
	case messages.ForwardMessage:
		s.onForward(ctx, msg)
	}
}

func New(stub remote.Stub, id string) *actor.Props {
	s := &Actor{
		id:            id,
		stub:          stub,
		localObjects:  make(map[string]messages.Object),
		remoteObjects: make(map[string]utils.Future[messages.Object]),
	}
	return actor.PropsFromProducer(func() actor.Actor {
		return s
	}, actor.WithOnInit(func(ctx actor.Context) {
		s.ref = &proto.StoreRef{ID: id, PID: ctx.Self()}
	}))
}

func Spawn(ctx actor.SpawnerContext, stub remote.Stub, id string) *proto.StoreRef {
	s := &Actor{
		id:            id,
		stub:          stub,
		localObjects:  make(map[string]messages.Object),
		remoteObjects: make(map[string]utils.Future[messages.Object]),
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return s
	})
	pid, _ := ctx.SpawnNamed(props, "store."+id)
	ref := &proto.StoreRef{
		ID:  id,
		PID: pid,
	}
	s.ref = ref
	return ref
}

// GetObject can be called by actors within the same system
// It returns a future that resolves to the object or rejects with an error
func GetObject(ctx actor.Context, store *actor.PID, flow *proto.Flow) utils.Future[messages.Object] {
	fut := utils.NewFuture[messages.Object](configs.FlowTimeout)
	if flow == nil {
		fut.Reject(errors.New("flow is nil"))
		return fut
	}

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *messages.ObjectResponse:
			if msg.Error != nil {
				fut.Reject(errors.WrapWith(msg.Error, "flow %s fetch failed", flow.ObjectID))
				return
			}

			if msg.Value.GetID() != flow.ObjectID {
				err := errors.Format("flow %s received unexpected ID %s", flow.ObjectID, msg.Value.GetID())
				fut.Reject(err)
				return
			}

			fut.Resolve(msg.Value)
		}
	})

	flowActor := ctx.Spawn(props)
	ctx.Send(store, &messages.RequestObject{
		ReplyTo: flowActor,
		Flow:    flow,
	})

	return fut
}
