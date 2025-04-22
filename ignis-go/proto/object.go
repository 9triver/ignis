package proto

import "github.com/asynkron/protoactor-go/actor"

// Object wraps LocalObject and EncodedObject, and both types support serialization.
// Note that encoding/decoding an object maybe expensive, and Object should only used
// when calling an actor.
type Object interface {
	GetID() string
	GetLanguage() Language
	GetEncoded() (*EncodedObject, error)
	GetValue() (any, error)
	IsEncoded() (*EncodedObject, bool)
	ToStream() (Stream, bool)
}

type Stream interface {
	Object
	ToChan(ctx actor.Context) <-chan Object
}

var (
	_ Object = (*LocalObject)(nil)
	_ Object = (*EncodedObject)(nil)
	_ Object = (*LocalStream)(nil)
)
