package executor

import (
	"errors"

	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

func (ret *Return) Object() (*proto.EncodedObject, error) {
	switch c := ret.Result.(type) {
	case *Return_Error:
		return nil, errors.New(c.Error)
	case *Return_Value:
		return c.Value, nil
	default:
		panic("unreachable")
	}
}

func NewAddHandler(conn, name string, handler []byte, language proto.Language, methods []string) *Message {
	cmd := &AddHandler{
		Name:     name,
		Handler:  handler,
		Language: language,
		Methods:  methods,
	}
	return NewMessage(conn, cmd)
}

func NewRemoveHandler(conn, name string) *Message {
	cmd := &RemoveHandler{
		Name: name,
	}
	return NewMessage(conn, cmd)
}

func NewExecute(conn, corrId, name, method string, args map[string]*proto.EncodedObject) *Message {
	cmd := &Execute{CorrID: corrId,
		Name:   name,
		Method: method,
		Args:   args,
	}
	return NewMessage(conn, cmd)
}

func NewExit(conn string) *Message {
	return NewMessage(conn, &Exit{})
}

func NewReady(conn string) *Message {
	return NewMessage(conn, &Ready{})
}

func NewReturn(conn, corrId string, value *proto.EncodedObject, err error) *Message {
	cmd := &Return{CorrID: corrId}
	if err != nil {
		cmd.Result = &Return_Error{Error: err.Error()}
	} else {
		cmd.Result = &Return_Value{Value: value}
	}
	return NewMessage(conn, cmd)
}

func NewStreamChunk(conn, streamId string, value *proto.EncodedObject, err error) *Message {
	cmd := proto.NewStreamChunk(streamId, value, err)
	return NewMessage(conn, cmd)
}

func NewStreamEnd(conn, streamId string) *Message {
	cmd := proto.NewStreamEnd(streamId)
	return NewMessage(conn, cmd)
}

func NewMessage(conn string, cmd pb.Message) *Message {
	ret := &Message{Conn: conn}
	switch cmd := cmd.(type) {
	case *AddHandler:
		ret.Type = CommandType_R_ADD_HANDLER
		ret.Command = &Message_AddHandler{AddHandler: cmd}
	case *RemoveHandler:
		ret.Type = CommandType_R_REMOVE_HANDLER
		ret.Command = &Message_RemoveHandler{RemoveHandler: cmd}
	case *Execute:
		ret.Type = CommandType_R_EXECUTE
		ret.Command = &Message_Execute{Execute: cmd}
	case *Exit:
		ret.Type = CommandType_R_EXIT
		ret.Command = &Message_Exit{Exit: cmd}
	case *Ready:
		ret.Type = CommandType_D_READY
		ret.Command = &Message_Ready{Ready: cmd}
	case *Return:
		ret.Type = CommandType_D_RETURN
		ret.Command = &Message_Return{Return: cmd}
	case *proto.StreamChunk:
		ret.Type = CommandType_STREAM_CHUNK
		ret.Command = &Message_StreamChunk{StreamChunk: cmd}
	case *proto.StreamEnd:
		ret.Type = CommandType_STREAM_END
		ret.Command = &Message_StreamEnd{StreamEnd: cmd}
	default:
		ret.Type = CommandType_UNSPECIFIED
	}
	return ret
}
