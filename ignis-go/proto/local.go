package proto

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/9triver/ignis/configs"
	"reflect"
	"sync"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type LocalObject struct {
	ID       string
	Value    any
	Language Language
}

func (obj *LocalObject) GetID() string {
	return obj.ID
}

func (obj *LocalObject) IsEncoded() (*EncodedObject, bool) {
	return nil, false
}

func (obj *LocalObject) GetValue() (any, error) {
	return obj.Value, nil
}

func (obj *LocalObject) GetEncoded() (*EncodedObject, error) {
	o := &EncodedObject{
		ID:       obj.ID,
		Language: obj.Language,
	}

	switch obj.Language {
	case LangJson:
		data, err := json.Marshal(obj.Value)
		if err != nil {
			return nil, errors.WrapWith(err, "encoder: json failed")
		}
		o.Data = data
	case LangPython:
		if data, ok := obj.Value.([]byte); !ok {
			return nil, errors.New("encoder: python object must be pickled bytes")
		} else {
			o.Data = data
		}
	case LangGo:
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

func (obj *LocalObject) GetLanguage() Language {
	return obj.Language
}

func (obj *LocalObject) ToStream() (Stream, bool) {
	return nil, false
}

func NewLocalObject(value any, language Language) *LocalObject {
	return &LocalObject{
		ID:       utils.GenObjectID(),
		Value:    value,
		Language: language,
	}
}

type LocalStream struct {
	store     *actor.PID
	once      sync.Once
	mu        sync.RWMutex
	consumers []chan Object

	id        string
	completed bool
	language  Language
	values    chan Object
}

func (s *LocalStream) EnqueueChunk(chunk Object) {
	if s.completed {
		return
	}

	if chunk == nil {
		s.completed = true
		close(s.values)
		return
	}

	s.values <- chunk
}

func (s *LocalStream) SetRemote(store *actor.PID) {
	s.store = store
}

func (s *LocalStream) GetID() string {
	return s.id
}

func (s *LocalStream) IsEncoded() (*EncodedObject, bool) {
	return nil, false
}

func (s *LocalStream) GetLanguage() Language {
	return s.language
}

func (s *LocalStream) GetEncoded() (*EncodedObject, error) {
	//if s.store == nil {
	//	return nil, errors.New("stream remote not set")
	//}

	return &EncodedObject{
		ID:       s.id,
		Source:   s.store,
		Language: s.language,
		Stream:   true,
	}, nil
}

func (s *LocalStream) GetValue() (any, error) {
	return s.values, nil
}

func (s *LocalStream) doStart() {
	defer func() {
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, ch := range s.consumers {
			close(ch)
		}
	}()

	for obj := range s.values {
		s.mu.RLock()
		for _, ch := range s.consumers {
			ch <- obj
		}
		s.mu.RUnlock()
	}
}

func (s *LocalStream) ToChan(actor.Context) <-chan Object {
	ch := make(chan Object)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consumers = append(s.consumers, ch)
	s.once.Do(func() {
		go s.doStart()
	})
	return ch
}

func (s *LocalStream) ToStream() (Stream, bool) {
	return s, true
}

func NewLocalStream(values any, language Language) *LocalStream {
	s := &LocalStream{
		id:       utils.GenObjectID(),
		values:   make(chan Object, configs.ChannelBufferSize),
		language: language,
	}

	go func() {
		ch := reflect.ValueOf(values)
		for {
			v, ok := ch.Recv()
			if !ok {
				s.EnqueueChunk(nil)
				return
			}

			if obj, ok := v.Interface().(Object); ok {
				s.EnqueueChunk(obj)
			} else {
				s.EnqueueChunk(NewLocalObject(v.Interface(), language))
			}
		}
	}()

	return s
}
