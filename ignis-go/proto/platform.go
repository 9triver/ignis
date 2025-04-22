package proto

import (
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
	"github.com/asynkron/protoactor-go/actor"
)

const (
	LangUnknown = Language_LANG_UNKNOWN
	LangJson    = Language_LANG_JSON
	LangGo      = Language_LANG_GO
	LangPython  = Language_LANG_PYTHON
)

type Successor struct {
	ID    string
	Param string
	PID   *actor.PID
}

type CreateSession struct {
	SessionID  string
	Successors []*Successor
}

func NewStreamChunk(streamId string, value *EncodedObject, err error) *StreamChunk {
	cmd := &StreamChunk{StreamID: streamId}
	if err != nil {
		cmd.Chunk = &StreamChunk_Error{Error: err.Error()}
	} else {
		cmd.Chunk = &StreamChunk_Value{Value: value}
	}
	return cmd
}

func NewStreamEnd(streamId string) *StreamEnd {
	return &StreamEnd{StreamID: streamId}
}

func (flow *Flow) Get(ctx actor.Context) utils.Future[Object] {
	fut := utils.NewFuture[Object](configs.FlowTimeout)
	if flow == nil {
		fut.Reject(errors.New("flow is nil"))
		return fut
	}

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *Error:
			fut.Reject(errors.Format("flow %s failed: %s", flow.ObjectID, msg.Message))
		case Object:
			if msg.GetID() != flow.ObjectID {
				fut.Reject(errors.Format("flow %s failed: unexpected ID %s", flow.ObjectID, msg.GetID()))
				return
			}
			fut.Resolve(msg)
		}
	})

	flowActor := ctx.Spawn(props)
	ctx.Send(flow.Source, &ObjectRequest{
		ID:      flow.ObjectID,
		ReplyTo: flowActor,
	})

	return fut
}
