package compute

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/utils"
)

type ExecInput struct {
	Context   actor.Context
	SessionID string
	Params    map[string]messages.Object
	OnDone    func(obj messages.Object, err error, duration time.Duration)
}

type Executor struct {
	requests chan *ExecInput
	handler  functions.Function
}

func (e *Executor) Deps() utils.Set[string] {
	return e.handler.Params()
}

func (e *Executor) doStart() {
	for req := range e.requests {
		tic := time.Now()
		obj, err := e.handler.Call(req.Params)
		req.OnDone(obj, err, time.Since(tic))
	}
}

func (e *Executor) Requests() chan<- *ExecInput {
	return e.requests
}

func (e *Executor) Close() {
	close(e.requests)
}

func NewExecutor(handler functions.Function) *Executor {
	self := &Executor{
		requests: make(chan *ExecInput, configs.ChannelBufferSize),
		handler:  handler,
	}
	go self.doStart()
	return self
}
