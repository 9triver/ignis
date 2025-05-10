package objects

import (
	"reflect"
	"sync"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/utils"
)

type Stream struct {
	once      sync.Once
	mu        sync.RWMutex
	consumers []chan Interface

	id        string
	completed bool
	language  Language
	values    chan Interface
}

func (s *Stream) EnqueueChunk(chunk Interface) {
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

func (s *Stream) GetID() string {
	return s.id
}

func (s *Stream) GetLanguage() Language {
	return s.language
}

func (s *Stream) Encode() (*Remote, error) {
	return &Remote{
		ID:       s.id,
		Language: s.language,
		Stream:   true,
	}, nil
}

func (s *Stream) Value() (any, error) {
	return s.values, nil
}

func (s *Stream) doStart() {
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

func (s *Stream) ToChan() <-chan Interface {
	ch := make(chan Interface)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consumers = append(s.consumers, ch)
	s.once.Do(func() {
		go s.doStart()
	})
	return ch
}

func NewStream(values any, language Language) *Stream {
	return StreamWithID(utils.GenIDWith("stream."), values, language)
}

func StreamWithID(id string, values any, language Language) *Stream {
	s := &Stream{
		id:       id,
		values:   make(chan Interface, configs.ChannelBufferSize),
		language: language,
	}

	if values == nil {
		return s
	}

	go func() {
		defer s.EnqueueChunk(nil)
		ch := reflect.ValueOf(values)
		for {
			v, ok := ch.Recv()
			if !ok {
				return
			}

			if obj, ok := v.Interface().(Interface); ok {
				s.EnqueueChunk(obj)
			} else {
				s.EnqueueChunk(NewLocal(v.Interface(), language))
			}
		}
	}()

	return s
}
