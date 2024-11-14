package utils

import (
	"errors"
	"time"
)

type Future[T any] struct {
	cond    chan struct{}
	done    bool
	value   T
	err     error
	timeout <-chan struct{}
}

func NewFuture[T any](timeout time.Duration) *Future[T] {
	timeCh := make(chan struct{})
	go func() {
		<-time.After(timeout)
		close(timeCh)
	}()

	return &Future[T]{
		cond:    make(chan struct{}),
		timeout: timeCh,
	}
}

func (f *Future[T]) wait() {
	select {
	case <-f.cond:
	case <-f.timeout:
		f.Reject(errors.New("future: timed out"))
	}
}

func (f *Future[T]) Result() (T, error) {
	f.wait()
	return f.value, f.err
}

func (f *Future[T]) Resolve(value T) {
	if f.done {
		return
	}

	f.value = value
	f.done = true
	close(f.cond)
}

func (f *Future[T]) Reject(err error) {
	if f.done {
		return
	}

	f.err = err
	f.done = true
	close(f.cond)
}
