package store

import (
	"actors/proto"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
)

type Language = proto.Language

const (
	LangJson   = proto.Language_LANG_JSON
	LangGo     = proto.Language_LANG_GO
	LangPython = proto.Language_LANG_PYTHON
)

type Object struct {
	ID       string
	Value    any
	Language Language
}

func (o *Object) Encode() (*proto.EncodedObject, error) {
	obj := &proto.EncodedObject{
		ID:       o.ID,
		Language: o.Language,
	}

	switch o.Language {
	case LangJson:
		data, err := json.Marshal(o.Value)
		if err != nil {
			return nil, fmt.Errorf("encoder: json marshal error: %w", err)
		}
		obj.Data = data
	case LangGo:
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(o); err != nil {
			return nil, fmt.Errorf("encoder: gob encode error: %w", err)
		}
		obj.Data = buf.Bytes()
	case LangPython:
		if bytes, ok := o.Value.([]byte); !ok {
			return nil, errors.New("encoder: python object is not bytes")
		} else {
			obj.Data = bytes
		}
	default:
		return nil, errors.New("encoder: unsupported language")
	}
	return obj, nil
}

func (o *Object) Decode(encObj *proto.EncodedObject) error {
	switch encObj.Language {
	case LangJson:
		o.Language = LangJson
		return json.Unmarshal(encObj.Data, &o.Value)
	case LangGo:
		o.Language = LangGo
		buf := bytes.NewBuffer(encObj.Data)
		dec := gob.NewDecoder(buf)
		return dec.Decode(o)
	case LangPython:
		o.Language = LangPython
		o.Value = encObj.Data
		return nil
	default:
		return errors.New("decoder: unsupported language")
	}
}
