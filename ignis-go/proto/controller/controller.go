package controller

import (
	pb "google.golang.org/protobuf/proto"

	"github.com/9triver/ignis/proto"
)

func NewAck(err error) *Message {
	ack := &Ack{}
	if err != nil {
		ack.Error = err.Error()
	}
	return NewMessage(ack)
}

func NewReady() *Message {
	return NewMessage(&Ready{})
}

func NewAppendData(sessionId string, object *proto.EncodedObject) *Message {
	cmd := &AppendData{SessionID: sessionId, Object: object}
	return NewMessage(cmd)
}

func NewAppendPyFunc(name string, params []string, venv string, requirements []string, pickedObj []byte, language proto.Language) *Message {
	cmd := &AppendPyFunc{
		Name:          name,
		Params:        params,
		Venv:          venv,
		Requirements:  requirements,
		PickledObject: pickedObj,
		Language:      language,
	}
	return NewMessage(cmd)
}

func NewAppendActor(name string, params []string, ref *proto.ActorRef) *Message {
	cmd := &AppendActor{
		Name:   name,
		Params: params,
		Ref:    ref,
	}
	return NewMessage(cmd)
}

func NewAppendArg(sessionId, instanceId, name, param string, flow *proto.Flow, encoded *proto.EncodedObject) *Message {
	data := &Data{}
	if flow != nil {
		data.Type = Data_OBJ_REF
		data.Object = &Data_Ref{Ref: flow}
	} else if encoded != nil {
		data.Type = Data_OBJ_ENCODED
		data.Object = &Data_Encoded{Encoded: encoded}
	}

	cmd := &AppendArg{
		SessionID:  sessionId,
		InstanceID: instanceId,
		Name:       name,
		Param:      param,
		Value:      data,
	}
	return NewMessage(cmd)
}

func NewAppendArgFromRef(sessionId, instanceId, name, param string, flow *proto.Flow) *Message {
	return NewAppendArg(sessionId, instanceId, name, param, flow, nil)
}

func NewAppendArgFromEncoded(sessionId, instanceId, name, param string, encoded *proto.EncodedObject) *Message {
	return NewAppendArg(sessionId, instanceId, name, param, nil, encoded)
}

func NewReturnResult(sessionId, instanceId, name string, value *proto.Flow, err error) *Message {
	cmd := &ReturnResult{
		SessionID:  sessionId,
		InstanceID: instanceId,
		Name:       name,
	}
	if err != nil {
		cmd.Result = &ReturnResult_Error{Error: err.Error()}
	} else {
		cmd.Result = &ReturnResult_Value{Value: &Data{
			Type:   Data_OBJ_REF,
			Object: &Data_Ref{Ref: value},
		}}
	}
	return NewMessage(cmd)
}

func NewMessage(cmd pb.Message) *Message {
	ret := &Message{}

	switch cmd := cmd.(type) {
	case *Ack:
		ret.Type = CommandType_ACK
		ret.Command = &Message_Ack{Ack: cmd}
	case *Ready:
		ret.Type = CommandType_FR_READY
		ret.Command = &Message_Ready{Ready: cmd}
	case *AppendData:
		ret.Type = CommandType_FR_APPEND_DATA
		ret.Command = &Message_AppendData{AppendData: cmd}
	case *AppendActor:
		ret.Type = CommandType_FR_APPEND_ACTOR
		ret.Command = &Message_AppendActor{AppendActor: cmd}
	case *AppendPyFunc:
		ret.Type = CommandType_FR_APPEND_PY_FUNC
		ret.Command = &Message_AppendPyFunc{AppendPyFunc: cmd}
	case *AppendArg:
		ret.Type = CommandType_FR_APPEND_ARG
		ret.Command = &Message_AppendArg{AppendArg: cmd}
	case *ReturnResult:
		ret.Type = CommandType_BK_RETURN_RESULT
		ret.Command = &Message_ReturnResult{ReturnResult: cmd}
	default:
		ret.Type = CommandType_UNSPECIFIED
	}
	return ret
}
