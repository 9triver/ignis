package ipc

import (
	"actors/platform/store"
	"actors/platform/utils"
	"actors/proto"
	"actors/proto/ipc"

	pb "google.golang.org/protobuf/proto"
)

func MakeAddHandler(name string, handler []byte, language store.Language, methods []string) *ipc.AddHandler {
	return &ipc.AddHandler{
		Name:     name,
		Handler:  handler,
		Language: language,
		Methods:  methods,
	}
}

func MakeRemoveHandler(name string) *ipc.RemoveHandler {
	return &ipc.RemoveHandler{
		Name: name,
	}
}

func MakeExecute(name, method string, args map[string]*proto.EncodedObject) *ipc.Execute {
	return &ipc.Execute{
		ID:     utils.GenNextID(),
		Name:   name,
		Method: method,
		Args:   args,
	}
}

func MakeExit() *ipc.Exit {
	return &ipc.Exit{}
}

func MakeRouterMessage(cmd pb.Message) *ipc.RouterMessage {
	ret := &ipc.RouterMessage{}
	switch cmd := cmd.(type) {
	case *ipc.AddHandler:
		ret.Command = ipc.RouterCommand_ROUTER_ADD_HANDLER
		ret.Payload = &ipc.RouterMessage_AddHandler{
			AddHandler: cmd,
		}
	case *ipc.RemoveHandler:
		ret.Command = ipc.RouterCommand_ROUTER_REMOVE_HANDLER
		ret.Payload = &ipc.RouterMessage_RemoveHandler{
			RemoveHandler: cmd,
		}
	case *ipc.Execute:
		ret.Command = ipc.RouterCommand_ROUTER_EXECUTE
		ret.Payload = &ipc.RouterMessage_Execute{
			Execute: cmd,
		}
	case *ipc.Exit:
		ret.Command = ipc.RouterCommand_ROUTER_EXIT
		ret.Payload = &ipc.RouterMessage_Exit{
			Exit: cmd,
		}
	default:
		ret.Command = ipc.RouterCommand_ROUTER_UNSPECIFIED
	}
	return ret
}
