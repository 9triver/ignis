package cluster

import (
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type StreamChunk = proto.StreamChunk

func NewObjectRequest(store *proto.StoreRef, id string, replyTo *proto.StoreRef) *Envelope {
	msg := &ObjectRequest{
		ID:      id,
		ReplyTo: replyTo,
	}
	return NewEnvelope(store, msg)
}

func NewObjectResponse(store *proto.StoreRef, id string, value *proto.EncodedObject, err error) *Envelope {
	msg := &ObjectResponse{
		ID:    id,
		Value: value,
		Error: err.Error(),
	}
	return NewEnvelope(store, msg)
}

func NewStreamChunk(store *proto.StoreRef, chunk *proto.StreamChunk) *Envelope {
	return NewEnvelope(store, chunk)
}

func NewEnvelope(store *proto.StoreRef, msg pb.Message) *Envelope {
	e := &Envelope{
		Store: store,
	}
	switch msg := msg.(type) {
	case *ObjectRequest:
		e.Type = MessageType_OBJECT_REQUEST
		e.Message = &Envelope_ObjectRequest{ObjectRequest: msg}
	case *ObjectResponse:
		e.Type = MessageType_OBJECT_RESPONSE
		e.Message = &Envelope_ObjectResponse{ObjectResponse: msg}
	case *StreamChunk:
		e.Type = MessageType_STREAM_CHUNK
		e.Message = &Envelope_StreamChunk{StreamChunk: msg}
	default:
		e.Type = MessageType_UNSPECIFIED
	}
	return e
}

func (e *Envelope) Unwrap() pb.Message {
	switch msg := e.Message.(type) {
	case *Envelope_ObjectRequest:
		return msg.ObjectRequest
	case *Envelope_ObjectResponse:
		return msg.ObjectResponse
	case *Envelope_StreamChunk:
		return msg.StreamChunk
	default:
		return nil
	}
}
