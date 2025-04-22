package proto

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/9triver/ignis/utils/errors"
	"github.com/asynkron/protoactor-go/actor"
)

/** Definition for EncodedObject (generated):
type EncodedObject struct {
	ID       string     // if returned from ipc call, id won't be set
	Data     []byte     // serialized object data, or nil if current object is a stream
	Source   *actor.PID // points to store actor of the object.
	Language Language   // if is JSON, it can be decoded to either Go, Python, or else it can only be decoded to corresponding language.
}
*/

func (obj *EncodedObject) IsEncoded() (*EncodedObject, bool) {
	return obj, true
}

func (obj *EncodedObject) GetEncoded() (*EncodedObject, error) {
	return obj, nil
}

func (obj *EncodedObject) asObject() (any, error) {
	switch obj.Language {
	case Language_LANG_JSON:
		var v any
		if err := json.Unmarshal(obj.Data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case Language_LANG_PYTHON:
		return nil, errors.New("decoding python obj is not supported in Go runtime")
	case Language_LANG_GO:
		var v any
		dec := gob.NewDecoder(bytes.NewReader(obj.Data))
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, errors.New("unknown language")
	}
}

func (obj *EncodedObject) ToChan(ctx actor.Context) <-chan Object {
	values := make(chan Object)
	props := actor.PropsFromFunc(func(c actor.Context) {
		switch o := c.Message().(type) {
		case *StreamEnd:
			close(values)
		case *StreamChunk:
			if o.StreamID != obj.ID {
				return
			}
			values <- o.GetValue()
		}
	})

	flowActor := ctx.Spawn(props)
	ctx.Send(obj.Source, &StreamRequest{
		StreamID: obj.ID,
		ReplyTo:  flowActor,
	})
	return values
}

func (obj *EncodedObject) GetValue() (any, error) {
	if obj.Stream {
		return nil, errors.New("cannot get object directly on stream")
	}
	return obj.asObject()
}

func (obj *EncodedObject) ToStream() (Stream, bool) {
	if obj.Stream {
		return obj, true
	}
	return nil, false
}
