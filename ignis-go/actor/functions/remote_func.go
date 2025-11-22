package functions

import (
	"time"

	"github.com/9triver/ignis/actor/functions/remote"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
)

type WSChunk struct {
	CorrId string `json:"corr_id"`
	Topic  string `json:"topic"`
	Value  any    `json:"value"`
	Error  string `json:"error"`
}

type RemoteFunction struct {
	FuncDec
	conn *remote.Connection
}

func (f *RemoteFunction) Name() string {
	return f.name
}

func (f *RemoteFunction) Params() []string {
	return f.params
}

func (f *RemoteFunction) Call(params map[string]object.Interface) (object.Interface, error) {
	return f.conn.Execute(f.name, params).Result()
}

func (f *RemoteFunction) TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error) {
	start := time.Now()
	obj, err := f.Call(params)
	return time.Since(start), obj, err
}

func (f *RemoteFunction) Language() proto.Language {
	return proto.Language_LANG_JSON
}

func NewRemote(
	manager *remote.Manager,
	name string,
	params []string,
	connId string,
) *RemoteFunction {
	conn := manager.GetConnection(connId)
	return &RemoteFunction{
		FuncDec: FuncDec{name, params},
		conn:    conn,
	}
}
