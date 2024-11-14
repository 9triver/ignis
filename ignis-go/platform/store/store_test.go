package store

import (
	"testing"
)

type A struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func TestObject(t *testing.T) {
	value := &A{10, "test"}
	store := New()

	obj := store.Add(value, LangGo)
	data, err := obj.Encode()
	if err != nil {
		t.Errorf("Error encoding object: %v", err)
	}

	t.Log("Object data:", data)

	obj2 := &Object{
		ID:       obj.ID,
		Language: obj.Language,
	}
	if err := obj2.Decode(data); err != nil {
		t.Errorf("Error decoding object: %v", err)
	}

	t.Log("Object data:", obj2.Value)
}
