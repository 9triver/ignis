package proto

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/9triver/ignis/utils/errors"
)

func NewStreamChunk(streamId string, target string, value *EncodedObject, err error) *StreamChunk {
	chunk := &StreamChunk{StreamID: streamId, Target: target, EoS: false}
	if err != nil {
		chunk.Error = err.Error()
	} else {
		chunk.Value = value
	}
	return chunk
}

func NewStreamEnd(streamId string, target string) *StreamChunk {
	return &StreamChunk{StreamID: streamId, Target: target, EoS: true}
}

/** Definition for EncodedObject (generated):
type EncodedObject struct {
	ID       string     // if returned from ipc call, id won't be set
	Data     []byte     // serialized object data, or nil if current object is a stream
	Source   *actor.PID // points to store actor of the object.
	Language Language   // if is JSON, it can be decoded to either Go, Python, or else it can only be decoded to corresponding language.
}
*/

func (obj *EncodedObject) Encode() (*EncodedObject, error) {
	return obj, nil
}

func (obj *EncodedObject) Value() (any, error) {
	if obj.Stream {
		return nil, errors.New("cannot get object directly on stream")
	}
	switch obj.Language {
	case Language_LANG_JSON:
		var v any
		if err := json.Unmarshal(obj.Data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case Language_LANG_PYTHON:
		return nil, errors.New("decoding python obj is not supported in Go runtime")
	case Language_LANG_GO:
		var v any
		dec := gob.NewDecoder(bytes.NewReader(obj.Data))
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, errors.New("unknown language")
	}
}

func (ref *StoreRef) Equals(other *StoreRef) bool {
	if other == nil {
		return false
	}
	return ref.ID == other.ID
}

func (ref *StoreRef) Addr() string {
	return fmt.Sprintf("store.%s", ref.ID)
}

func (ref *ActorRef) Equals(other *ActorRef) bool {
	if other == nil {
		return false
	}
	return ref.ID == other.ID && ref.Store.Equals(other.Store)
}

func (ref *ActorRef) Addr() string {
	return fmt.Sprintf("actor.%s@%s", ref.ID, ref.Store.Addr())
}

func (sr *InvokeStart) GetTarget() string {
	return sr.Info.Ref.ID
}
