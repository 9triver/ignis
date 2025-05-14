package store

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/objects"
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
	localObjects map[string]objects.Interface
	// remoteObjects saves objects from remote stores
	remoteObjects map[string]utils.Future[objects.Interface]
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
	} else if encoded, err := obj.Encode(); err != nil {
		reply.Error = errors.WrapWith(err, "store: object %s encode failed", req.ID).Error()
	} else {
		reply.Value = encoded
	}
	s.stub.SendTo(req.ReplyTo, reply)

	// if requested object is a stream, send all chunks
	if stream, ok := obj.(*objects.Stream); ok {
		go func() {
			defer s.stub.SendTo(req.ReplyTo, proto.NewStreamEnd(req.ID))
			for obj := range stream.ToChan() {
				encoded, err := obj.Encode()
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
		ls := objects.NewStream(nil, encoded.Language)
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

	stream, ok := obj.(*objects.Stream)
	if !ok {
		return
	}

	if chunk.EoS {
		stream.EnqueueChunk(nil)
	} else {
		stream.EnqueueChunk(chunk.Value)
	}
}

func (s *Actor) onSaveObject(ctx actor.Context, save *SaveObject) {
	obj := save.Value
	ctx.Logger().Info("store: save object",
		"id", obj.GetID(),
	)

	s.localObjects[obj.GetID()] = obj

	if save.Callback != nil {
		save.Callback(ctx, &proto.Flow{ID: obj.GetID(), Source: s.ref})
	}
}

func (s *Actor) getLocalObject(flow *proto.Flow) (objects.Interface, error) {
	obj, ok := s.localObjects[flow.ID]
	if !ok {
		return nil, errors.Format("store: flow %s not found", flow.ID)
	}
	return obj, nil
}

func (s *Actor) requestRemoteObject(flow *proto.Flow) utils.Future[objects.Interface] {
	if fut, ok := s.remoteObjects[flow.ID]; ok {
		return fut
	}

	fut := utils.NewFuture[objects.Interface](configs.FlowTimeout)
	s.remoteObjects[flow.ID] = fut

	remoteRef := flow.Source
	s.stub.SendTo(remoteRef, &cluster.ObjectRequest{
		ID:      flow.ID,
		ReplyTo: s.ref,
	})
	return fut
}

func (s *Actor) onFlowRequest(ctx actor.Context, req *RequestObject) {
	ctx.Logger().Info("store: local flow request",
		"id", req.Flow.ID,
		"store", req.Flow.Source.ID,
	)

	if req.Flow.Source.Equals(s.ref) {
		obj, err := s.getLocalObject(req.Flow)
		req.Callback(ctx, obj, err)
	} else {
		s.requestRemoteObject(req.Flow).OnDone(func(obj objects.Interface, duration time.Duration, err error) {
			req.Callback(ctx, obj, err)
		})
	}
}

func (s *Actor) onForward(ctx actor.Context, forward forwardMessage) {
	target := forward.GetTarget()
	if target.Store.Equals(s.ref) { // target is local
		ctx.Send(target.PID, forward)
	} else {
		ctx.Logger().Info("store: forwarding request")
		s.stub.SendTo(target.Store, forward)
	}
}

func (s *Actor) onClose(ctx actor.Context) {
	ctx.Logger().Info("store: closing")
	for _, fut := range s.remoteObjects {
		fut.Reject(errors.New("store: closing"))
	}
	if err := s.stub.Close(); err != nil {
		ctx.Logger().Error("store: error closing stub", "err", err)
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
	case *SaveObject:
		s.onSaveObject(ctx, msg)
	case *RequestObject:
		s.onFlowRequest(ctx, msg)
	// forward messages
	case forwardMessage:
		s.onForward(ctx, msg)
	case *actor.Stopped:
		s.onClose(ctx)
	}
}

func Spawn(ctx *actor.RootContext, spawnStub remote.StubSpawnFunc, id string) *proto.StoreRef {
	stub := spawnStub(ctx)

	s := &Actor{
		id:            id,
		stub:          stub,
		localObjects:  make(map[string]objects.Interface),
		remoteObjects: make(map[string]utils.Future[objects.Interface]),
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

	go func() {
		for msg := range stub.RecvChan() {
			ctx.Send(pid, msg.Unwrap())
		}
	}()

	return ref
}

// GetObject can be called by actors within the same system
// It returns a future that resolves to the object or rejects with an error
func GetObject(ctx actor.Context, store *actor.PID, flow *proto.Flow) utils.Future[objects.Interface] {
	fut := utils.NewFuture[objects.Interface](configs.FlowTimeout)
	if flow == nil {
		fut.Reject(errors.New("flow is nil"))
		return fut
	}

	ctx.Send(store, &RequestObject{
		Flow: flow,
		Callback: func(ctx actor.Context, obj objects.Interface, err error) {
			if err != nil {
				fut.Reject(errors.WrapWith(err, "flow %s fetch failed", flow.ID))
				return
			}

			if obj.GetID() != flow.ID {
				err := errors.Format("flow %s received unexpected ID %s", flow.ID, obj.GetID())
				fut.Reject(err)
				return
			}

			fut.Resolve(obj)
		},
	})

	return fut
}
