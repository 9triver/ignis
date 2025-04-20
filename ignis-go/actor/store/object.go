package store

import (
	"encoding/json"
	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type LocalObject struct {
	ID       string
	Value    any
	Language proto.Language
}

func (obj *LocalObject) GetID() string {
	return obj.ID
}

func (obj *LocalObject) IsEncoded() bool {
	return false
}

func (obj *LocalObject) GetValue(ctx actor.Context) (any, error) {
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
		if bytes, ok := obj.Value.([]byte); !ok {
			return nil, errors.New("encoder: python object must be pickled bytes")
		} else {
			o.Data = bytes
		}
	default:
		return nil, errors.New("encoder: unsupported language")
	}
	return o, nil
}

func (obj *LocalObject) GetLanguage() proto.Language {
	return obj.Language
}

func (obj *LocalObject) IsStream() bool {
	return false
}

var _ proto.Object = (*LocalObject)(nil)

func NewLocalObject(value any, language proto.Language) *LocalObject {
	return &LocalObject{
		ID:       utils.GenObjectID(),
		Value:    value,
		Language: language,
	}
}
