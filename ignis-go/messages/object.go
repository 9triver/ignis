package messages

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// Object wraps LocalObject and EncodedObject, and both types support serialization.
// Note that encoding/decoding an object maybe expensive, and Object should only used
// when calling an actor.
type Object interface {
	GetID() string
	GetLanguage() proto.Language
	GetEncoded() (*proto.EncodedObject, error)
	GetValue() (any, error)
}

type LocalObject struct {
	ID       string
	Value    any
	Language proto.Language
}

func (obj *LocalObject) GetID() string {
	return obj.ID
}

func (obj *LocalObject) GetValue() (any, error) {
	return obj.Value, nil
}

func (obj *LocalObject) GetEncoded() (*proto.EncodedObject, error) {
	o := &proto.EncodedObject{
		ID:       obj.ID,
		Language: obj.Language,
	}

	switch obj.Language {
	case proto.LangJson:
		data, err := json.Marshal(obj.Value)
		if err != nil {
			return nil, errors.WrapWith(err, "encoder: json failed")
		}
		o.Data = data
	case proto.LangPython:
		if data, ok := obj.Value.([]byte); !ok {
			return nil, errors.New("encoder: python object must be pickled bytes")
		} else {
			o.Data = data
		}
	case proto.LangGo:
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(obj.Value); err != nil {
			return nil, errors.WrapWith(err, "encoder: gob failed")
		}
		o.Data = buf.Bytes()
	default:
		return nil, errors.New("encoder: unsupported language")
	}
	return o, nil
}

func (obj *LocalObject) GetLanguage() proto.Language {
	return obj.Language
}

func NewLocalObject(value any, language proto.Language) *LocalObject {
	return NewLocalObjectWithID(utils.GenObjectID(), value, language)
}

func NewLocalObjectWithID(id string, value any, language proto.Language) *LocalObject {
	return &LocalObject{
		ID:       id,
		Value:    value,
		Language: language,
	}
}

var (
	_ Object = (*LocalObject)(nil)
	_ Object = (*proto.EncodedObject)(nil)
	_ Object = (*LocalStream)(nil)
)
