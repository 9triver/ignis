package proto

import (
	"encoding/json"
	"errors"
	"github.com/asynkron/protoactor-go/actor"
)

// Object wraps LocalObject and EncodedObject, and both types support serialization.
// Note that encoding/decoding an object maybe expensive, and Object should only used
// when calling an actor.
type Object interface {
	GetID() string
	IsEncoded() bool
	GetLanguage() Language
	GetEncoded() (*EncodedObject, error)
	GetValue(ctx actor.Context) (any, error)
	IsStream() bool
}

func (obj *EncodedObject) IsEncoded() bool {
	return true
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
	default:
		return nil, errors.New("unknown language")
	}
}

func (obj *EncodedObject) asStream(ctx actor.Context) (any, error) {
	values := make(chan any)
	props := actor.PropsFromFunc(func(c actor.Context) {
		switch o := c.Message().(type) {
		case *Error:
			values <- errors.New(o.Message)
		case Object:
			v, err := o.GetValue(c)
			if err != nil {
				values <- v
			} else {
				values <- err
			}
		}
	})
	flowActor := ctx.Spawn(props)
	ctx.Send(obj.Source, &ObjectRequest{
		ID:      obj.ID,
		ReplyTo: flowActor,
	})
	return values, nil
}

func (obj *EncodedObject) GetValue(ctx actor.Context) (any, error) {
	if !obj.IsStream() {
		return obj.asObject()
	}
	return obj.asStream(ctx)
}

func (obj *EncodedObject) IsStream() bool {
	return obj.Data == nil && obj.Source != nil
}

var _ Object = (*EncodedObject)(nil)

const (
	LangUnknown = Language_LANG_UNKNOWN
	LangJson    = Language_LANG_JSON
	LangGo      = Language_LANG_GO
	LangPython  = Language_LANG_PYTHON
)
