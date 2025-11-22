package cluster

import (
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

type StreamChunk = proto.StreamChunk

func NewFunction(name string, params []string, requirements []string, obj []byte, lang proto.Language) *Message {
	return NewMessage(&Function{
		Name:          name,
		Params:        params,
		Requirements:  requirements,
		PickledObject: obj,
		Language:      lang,
	})
}

func NewObjectRequest(store *proto.StoreRef, id string, target, replyTo string) *Envelope {
	msg := &ObjectRequest{
		ID:      id,
		Target:  target,
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

func NewMessage(msg pb.Message) *Message {
	switch msg := msg.(type) {
	case *ObjectRequest:
		return &Message{
			Type:    MessageType_OBJECT_REQUEST,
			Message: &Message_ObjectRequest{ObjectRequest: msg},
		}
	case *ObjectResponse:
		return &Message{
			Type:    MessageType_OBJECT_RESPONSE,
			Message: &Message_ObjectResponse{ObjectResponse: msg},
		}
	case *StreamChunk:
		return &Message{
			Type:    MessageType_STREAM_CHUNK,
			Message: &Message_StreamChunk{StreamChunk: msg},
		}
	case *Ack:
		return &Message{
			Type:    MessageType_ACK,
			Message: &Message_Ack{Ack: msg},
		}
	case *Ready:
		return &Message{
			Type:    MessageType_READY,
			Message: &Message_Ready{Ready: msg},
		}
	case *proto.Invoke:
		return &Message{
			Type:    MessageType_INVOKE,
			Message: &Message_Invoke{Invoke: msg},
		}
	case *proto.InvokeStart:
		return &Message{
			Type:    MessageType_INVOKE_START,
			Message: &Message_InvokeStart{InvokeStart: msg},
		}
	case *proto.InvokeResponse:
		return &Message{
			Type:    MessageType_INVOKE_RESPONSE,
			Message: &Message_InvokeResponse{InvokeResponse: msg},
		}
	case *Function:
		return &Message{
			Type:    MessageType_FUNCTION,
			Message: &Message_Function{Function: msg},
		}
	default:
		return nil
	}
}

func (msg *Message) Unwrap() pb.Message {
	switch msg := msg.Message.(type) {
	case *Message_ObjectRequest:
		return msg.ObjectRequest
	case *Message_ObjectResponse:
		return msg.ObjectResponse
	case *Message_StreamChunk:
		return msg.StreamChunk
	case *Message_Ack:
		return msg.Ack
	case *Message_Ready:
		return msg.Ready
	case *Message_Invoke:
		return msg.Invoke
	case *Message_InvokeStart:
		return msg.InvokeStart
	case *Message_InvokeResponse:
		return msg.InvokeResponse
	case *Message_Function:
		return msg.Function
	default:
		return nil
	}
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
