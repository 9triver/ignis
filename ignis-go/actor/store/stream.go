package store

import (
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

type LocalStream struct {
	id          string
	completed   bool
	store       *actor.PID
	chunks      []proto.Object
	language    proto.Language
	values      chan any
	subscribers *actor.PIDSet
}

func (s *LocalStream) EnqueueChunk(chunk proto.Object, isEos bool) {
	if s.completed {
		return
	}
	s.chunks = append(s.chunks, chunk)
	s.values <- chunk
	if isEos {
		s.completed = true
		close(s.values)
	}
}

func (s *LocalStream) GetID() string {
	return s.id
}

func (s *LocalStream) IsEncoded() bool {
	return false
}

func (s *LocalStream) GetLanguage() proto.Language {
	return s.language
}

func (s *LocalStream) GetEncoded() (*proto.EncodedObject, error) {
	return &proto.EncodedObject{
		ID:       s.id,
		Data:     nil,
		Source:   s.store,
		Language: s.language,
	}, nil
}

func (s *LocalStream) GetValue(ctx actor.Context) (any, error) {
	return s.values, nil
}

func (s *LocalStream) IsStream() bool {
	return true
}
