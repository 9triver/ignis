package utils

import (
	"context"
	"errors"
	"time"
)

type FutureDoneCallback[T any] func(future T, err error)

type Future[T any] interface {
	Result() (T, error)
	Resolve(value T)
	Reject(err error)
	OnDone(callback FutureDoneCallback[T])
}

func NewFuture[T any](timeout time.Duration, callbacks ...FutureDoneCallback[T]) Future[T] {
	return newCtxFuture(timeout, callbacks...)
}

var ErrFutureTimeout = errors.New("future: timeout")

type ctxFutureImpl[T any] struct {
	ctx     context.Context
	cancel  context.CancelFunc
	done    bool
	value   T
	err     error
	onDones []FutureDoneCallback[T]
}

func (f *ctxFutureImpl[T]) onDone() {
	f.done = true
	f.cancel()
	for _, callback := range f.onDones {
		callback(f.value, f.err)
	}
}

func (f *ctxFutureImpl[T]) Result() (T, error) {
	<-f.ctx.Done()
	if !f.done { // timeout
		return f.value, ErrFutureTimeout
	}
	return f.value, f.err
}

func (f *ctxFutureImpl[T]) Resolve(value T) {
	select {
	case <-f.ctx.Done():
	default:
		f.value = value
		f.onDone()
	}
}

func (f *ctxFutureImpl[T]) Reject(err error) {
	select {
	case <-f.ctx.Done():
	default:
		f.err = err
		f.onDone()
	}
}

func (f *ctxFutureImpl[T]) OnDone(callback FutureDoneCallback[T]) {
	if f.done {
		callback(f.value, f.err)
		return
	}
	f.onDones = append(f.onDones, callback)
}

func newCtxFuture[T any](timeout time.Duration, callbacks ...FutureDoneCallback[T]) *ctxFutureImpl[T] {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	return &ctxFutureImpl[T]{
		ctx:     ctx,
		cancel:  cancel,
		onDones: callbacks,
	}
}
