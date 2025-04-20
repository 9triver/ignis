package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type Actor struct {
	name    string
	objects map[string]proto.Object
	streams map[string]*LocalStream
}

// SaveObject is sent to store when actor generates new return objects from functions
type SaveObject struct {
	Value    proto.Object                             // real object
	Callback func(ctx actor.Context, ref *proto.Flow) // called when object saving is completed
}

type SaveChunk struct {
	StreamID string
	Value    proto.Object
	IsEoS    bool
	Callback func(ctx actor.Context, ref *proto.Flow, isEos bool)
}

// SaveStream is sent to store when a stream handler is called
type SaveStream struct {
	StreamID string                                       // id of the created stream
	Callback func(ctx actor.Context, stream *LocalStream) // called when stream terminates
}

func (s *Actor) Name() string {
	return s.name
}

func (s *Actor) SerializeToFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.WrapWith(err, "%s: serialization failed", s.name)
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(s.objects)
	if err != nil {
		return errors.WrapWith(err, "%s: serialization failed", s.name)
	}

	buf := bufio.NewWriter(f)
	_, err = buf.Write(data)
	if err != nil {
		return errors.WrapWith(err, "%s: serialization failed", s.name)
	}

	err = buf.Flush()
	if err != nil {
		return errors.WrapWith(err, "%s: serialization failed", s.name)
	}
	return nil
}

func (s *Actor) LoadFromFile(path string) error {
	// TODO: unimplemented
	panic("unimplemented")
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

func (s *Actor) responseStream(ctx actor.Context, req *proto.ObjectRequest, stream *LocalStream) {
	for chunk := range stream.values {
		s.responseObject(ctx, req, NewLocalObject(chunk, stream.language))
	}
}

func (s *Actor) onObjectRequest(ctx actor.Context, req *proto.ObjectRequest) {
	ctx.Logger().Info("request object",
		"id", req.ID,
	)

	obj, ok1 := s.objects[req.ID]
	stream, ok2 := s.streams[req.ID]
	if !ok1 && !ok2 {
		ctx.Send(req.ReplyTo, &proto.Error{
			Sender:  ctx.Self(),
			Message: fmt.Sprintf("request %s: no such object", req.ID),
		})
	}

	if ok1 {
		s.responseObject(ctx, req, obj)
	} else {
		s.responseStream(ctx, req, stream)
	}
}

func (s *Actor) onSaveObject(ctx actor.Context, obj *SaveObject) {
	ctx.Logger().Info("save object",
		"id", obj.Value.GetID(),
	)

	s.objects[obj.Value.GetID()] = obj.Value
	if obj.Callback != nil {
		obj.Callback(ctx, &proto.Flow{ObjectID: obj.Value.GetID(), Source: ctx.Self()})
	}
}

func (s *Actor) onSaveStream(ctx actor.Context, stream *SaveStream) {
	ctx.Logger().Info("store: save stream",
		"id", stream.StreamID,
	)
	if _, ok := s.streams[stream.StreamID]; ok {
		return
	}
	s.streams[stream.StreamID] = &LocalStream{
		id:        stream.StreamID,
		completed: false,
		store:     ctx.Self(),
		chunks:    nil,
		values:    make(chan any),
	}
}

func (s *Actor) onSaveChunk(ctx actor.Context, chunk *SaveChunk) {
	ctx.Logger().Info("store: save chunk",
		"id", chunk.StreamID,
	)
	stream, ok := s.streams[chunk.StreamID]
	if !ok {
		return
	}
	stream.EnqueueChunk(chunk.Value, chunk.IsEoS)
	if chunk.Callback != nil {
		chunk.Callback(ctx, &proto.Flow{ObjectID: chunk.StreamID, Source: ctx.Self()}, chunk.IsEoS)
	}
}

func (s *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *proto.ObjectRequest:
		s.onObjectRequest(ctx, msg)
	case *SaveStream:
		s.onSaveStream(ctx, msg)
	case *SaveChunk:
		s.onSaveChunk(ctx, msg)
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
