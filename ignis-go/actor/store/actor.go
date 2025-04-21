package store

import (
	"fmt"
	"github.com/9triver/ignis/utils"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
)

type Actor struct {
	name    string
	objects map[string]proto.Object
}

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    proto.Object                             // object or stream
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

func (s *Actor) Name() string {
	return s.name
}

func (s *Actor) responseObject(ctx actor.Context, req *proto.ObjectRequest, obj proto.Object) {
	var reply any
	if utils.IsSameSystem(ctx.Self(), req.ReplyTo) {
		reply = obj
	} else if encoded, err := obj.GetEncoded(); err != nil {
		reply = &proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("request %s: %s", req.ID, err.Error()),
		}
	} else {
		reply = encoded
	}
	ctx.Send(req.ReplyTo, reply)
}

func (s *Actor) onObjectRequest(ctx actor.Context, req *proto.ObjectRequest) {
	ctx.Logger().Info("request object",
		"id", req.ID,
	)

	obj, ok := s.objects[req.ID]
	if !ok {
		ctx.Send(req.ReplyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("request %s: no such object", req.ID),
		})
		return
	}

	s.responseObject(ctx, req, obj)
}

func (s *Actor) onStreamRequest(ctx actor.Context, req *proto.StreamRequest) {
	ctx.Logger().Info("request stream",
		"id", req.StreamID,
	)

	obj, ok := s.objects[req.StreamID]
	if !ok {
		ctx.Send(req.ReplyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("request %s: no such stream", req.StreamID),
		})
		return
	}
	stream, ok := obj.ToStream()
	if !ok {
		ctx.Send(req.ReplyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("request %s: no such stream", req.StreamID),
		})
		return
	}

	objects := stream.ToChan(ctx)
	go func() {
		defer ctx.Send(req.ReplyTo, &proto.EndOfStream{})
		for obj := range objects {
			var msg any
			if encoded, err := obj.GetEncoded(); err != nil {
				msg = &proto.Error{
					Sender:  ctx.Self(),
					Message: fmt.Sprintf("request %s: %s", req.StreamID, err.Error()),
				}
			} else {
				msg = &proto.StreamChunk{StreamID: req.StreamID, Object: encoded}
			}
			ctx.Send(req.ReplyTo, msg)
		}
	}()
}

func (s *Actor) onSaveObject(ctx actor.Context, save *SaveObject) {
	obj := save.Value
	ctx.Logger().Info("save object",
		"id", obj.GetID(),
	)

	if stream, ok := obj.(*proto.LocalStream); ok {
		stream.SetRemote(ctx.Self())
	}

	s.objects[obj.GetID()] = obj

	if save.Callback != nil {
		save.Callback(ctx, &proto.Flow{ObjectID: obj.GetID(), Source: ctx.Self()})
	}
}

func (s *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.ObjectRequest:
		s.onObjectRequest(ctx, msg)
	case *proto.StreamRequest:
		s.onStreamRequest(ctx, msg)
	case *SaveObject:
		s.onSaveObject(ctx, msg)
	}
}

func New() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Actor{
			objects: make(map[string]proto.Object),
		}
	})
}
