package utils

import (
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
