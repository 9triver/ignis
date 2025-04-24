package messages

import (
	"reflect"
	"sync"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils"
)

type LocalStream struct {
	store     *proto.StoreRef
	once      sync.Once
	mu        sync.RWMutex
	consumers []chan Object

	id        string
	completed bool
	language  proto.Language
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

func (s *LocalStream) SetRemote(store *proto.StoreRef) {
	s.store = store
}

func (s *LocalStream) GetID() string {
	return s.id
}

func (s *LocalStream) GetLanguage() proto.Language {
	return s.language
}

func (s *LocalStream) GetEncoded() (*proto.EncodedObject, error) {
	return &proto.EncodedObject{
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

func (s *LocalStream) ToChan() <-chan Object {
	ch := make(chan Object)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consumers = append(s.consumers, ch)
	s.once.Do(func() {
		go s.doStart()
	})
	return ch
}

func NewLocalStream(values any, language proto.Language) *LocalStream {
	return NewLocalStreamWithID(utils.GenObjectID(), values, language)
}

func NewLocalStreamWithID(id string, values any, language proto.Language) *LocalStream {
	s := &LocalStream{
		id:       id,
		values:   make(chan Object, configs.ChannelBufferSize),
		language: language,
	}

	if values == nil {
		return s
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
