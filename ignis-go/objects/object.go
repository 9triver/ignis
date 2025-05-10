package objects

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

// Interface wraps Local, Stream and proto.EncodedObject, and all these types support serialization.
// Note that encoding/decoding an object maybe expensive, and Interface should only be used
// when calling an actor function.
type Interface interface {
	GetID() string
	GetLanguage() Language
	Encode() (*Remote, error)
	Value() (any, error)
}

var (
	_ Interface = (*Local)(nil)
	_ Interface = (*Remote)(nil)
	_ Interface = (*Stream)(nil)
)

type (
	Remote   = proto.EncodedObject
	Language = proto.Language
)

const (
	LangUnknown = proto.Language_LANG_UNKNOWN
	LangJson    = proto.Language_LANG_JSON
	LangGo      = proto.Language_LANG_GO
	LangPython  = proto.Language_LANG_PYTHON
)

type Local struct {
	id       string
	value    any
	language Language
}

func (obj *Local) GetID() string {
	return obj.id
}

func (obj *Local) Value() (any, error) {
	return obj.value, nil
}

func (obj *Local) Encode() (*Remote, error) {
	o := &Remote{
		ID:       obj.id,
		Language: obj.language,
	}

	switch obj.language {
	case LangJson:
		data, err := json.Marshal(obj.value)
		if err != nil {
			return nil, errors.WrapWith(err, "encoder: json failed")
		}
		o.Data = data
	case LangPython:
		if data, ok := obj.value.([]byte); !ok {
			return nil, errors.New("encoder: python object must be pickled bytes")
		} else {
			o.Data = data
		}
	case LangGo:
		buf := &bytes.Buffer{}
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(obj.value); err != nil {
			return nil, errors.WrapWith(err, "encoder: gob failed")
		}
		o.Data = buf.Bytes()
	default:
		return nil, errors.New("encoder: unsupported language")
	}
	return o, nil
}

func (obj *Local) GetLanguage() Language {
	return obj.language
}

func NewLocal(value any, language Language) *Local {
	return LocalWithID(utils.GenIDWith("obj."), value, language)
}

func LocalWithID(id string, value any, language Language) *Local {
	return &Local{
		id:       id,
		value:    value,
		language: language,
	}
}
