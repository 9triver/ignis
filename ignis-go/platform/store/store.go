package store

import (
	"actors/platform/utils"
	"actors/proto"
	"sync"
)

// TODO: rewrite in actor to avoid using Mutex
type Store struct {
	mu      sync.RWMutex
	objects map[string]*Object
}

func (s *Store) Get(id string) (*Object, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[id]
	return obj, ok
}

func (s *Store) GetAll() []*Object {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var objs []*Object
	for _, obj := range s.objects {
		objs = append(objs, obj)
	}
	return objs
}

func (s *Store) Add(obj any, lang Language) *Object {
	id := utils.GenObjectID()
	return s.Set(id, obj, lang)
}

func (s *Store) AddObject(obj *Object) *Object {
	return s.SetObj(utils.GenObjectID(), obj)
}

func (s *Store) AddEncoded(encObj *proto.EncodedObject) (*Object, error) {
	obj := &Object{}
	if err := obj.Decode(encObj); err != nil {
		return nil, err
	}
	return s.AddObject(obj), nil
}

func (s *Store) Set(id string, value any, lang Language) *Object {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := &Object{
		ID:       id,
		Value:    value,
		Language: lang,
	}
	s.objects[id] = obj
	return obj
}

func (s *Store) SetObj(id string, obj *Object) *Object {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj.ID = id
	s.objects[id] = obj
	return obj
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.objects, id)
}

func New() *Store {
	return &Store{
		objects: make(map[string]*Object),
	}
}
