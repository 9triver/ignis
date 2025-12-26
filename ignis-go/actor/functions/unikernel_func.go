package functions

import (
	"github.com/9triver/ignis/actor/functions/mirage"
	"github.com/9triver/ignis/transport/ws"
	"github.com/9triver/ignis/utils"
)

func NewUnikernel(
	manager *ws.Manager,
	name string,
	params []string,
	handlers string,
) (*RemoteFunction, error) {
	m := mirage.New(name, manager, handlers, "unix")
	if err := m.Build(); err != nil {
		return nil, err
	}

	connId := utils.GenIDWith("unikernel-")

	go m.Run(connId)

	f := NewRemote(manager, name, params, connId)
	return f, nil
}
