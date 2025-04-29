package utils

import (
	"reflect"
	"testing"
	"time"

	"github.com/9triver/ignis/utils/errors"
)

func TestFuture(t *testing.T) {
	f := newCtxFuture[int](3 * time.Second)
	go func() {
		<-time.After(4 * time.Second)
		f.Reject(errors.New("failed"))
	}()

	v, err := f.Result()
	if err != nil {
		t.Logf("future timeout: %s\n", err)
	} else {
		t.Log(v)
	}
}

func TestMapStruct(t *testing.T) {
	type B struct {
		B int
	}
	type A struct {
		A int
		B *B
	}

	m := map[string]any{
		"A": 1,
		"B": map[string]any{
			"B": 2,
		},
	}

	a, err := MapToStruct2(reflect.TypeFor[A](), m)
	t.Log(a, err)
}
