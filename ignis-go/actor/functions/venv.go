package functions

import (
	"context"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/9triver/ignis/actor/remote"
	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/messages"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/proto/executor"
	"github.com/9triver/ignis/utils"
	"github.com/9triver/ignis/utils/errors"
)

type VirtualEnv struct {
	mu      sync.Mutex
	ctx     context.Context
	handler remote.Executor
	started bool
	futures map[string]utils.Future[messages.Object]
	streams map[string]*messages.LocalStream

	Name     string   `json:"name"`
	Exec     string   `json:"exec"`
	Packages []string `json:"packages"`
}

func (v *VirtualEnv) Interpreter() string {
	return v.Exec
}

func (v *VirtualEnv) RunPip(args ...string) (*exec.Cmd, context.CancelFunc) {
	args = append([]string{"-m", "pip"}, args...)
	cmdCtx, cancel := context.WithTimeout(v.ctx, 300*time.Second)
	return exec.CommandContext(cmdCtx, v.Exec, args...), cancel
}

func (v *VirtualEnv) AddPackages(p ...string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	pkgSet := utils.MakeSetFromSlice(v.Packages)
	for _, pkg := range p {
		if pkgSet.Contains(pkg) {
			continue
		}

		if err := func() error {
			cmd, cancel := v.RunPip("install", pkg)
			defer cancel()

			if err := cmd.Run(); err != nil {
				return errors.WrapWith(err, "venv %s: failed installing package %s", v.Name, p)
			}
			return nil
		}(); err != nil {
			return err
		}

		v.Packages = append(v.Packages, pkg)
		pkgSet.Add(pkg)
	}
	return nil
}

func (v *VirtualEnv) Execute(name, method string, args map[string]messages.Object) utils.Future[messages.Object] {
	fut := utils.NewFuture[messages.Object](configs.ExecutionTimeout)
	encoded := make(map[string]*proto.EncodedObject)
	for param, obj := range args {
		enc, err := obj.GetEncoded()
		if err != nil {
			fut.Reject(err)
			return fut
		}
		encoded[param] = enc
	}

	corrId := utils.GenID()
	v.futures[corrId] = fut

	msg := executor.NewExecute(v.Name, corrId, name, method, encoded)
	v.handler.SendChan() <- msg

	for _, arg := range args {
		if stream, ok := arg.(*messages.LocalStream); ok {
			chunks := stream.ToChan()
			go func() {
				defer func() {
					v.handler.SendChan() <- executor.NewStreamEnd(v.Name, stream.GetID())
				}()
				for chunk := range chunks {
					encoded, err := chunk.GetEncoded()
					v.handler.SendChan() <- executor.NewStreamChunk(v.Name, stream.GetID(), encoded, err)
				}
			}()
		}
	}
	return fut
}

func (v *VirtualEnv) Send(msg *executor.Message) {
	v.handler.SendChan() <- msg
}

func (v *VirtualEnv) onReturn(ret *executor.Return) {
	fut, ok := v.futures[ret.CorrID]
	if !ok {
		return
	}
	defer delete(v.futures, ret.CorrID)

	obj, err := ret.Object()
	if err != nil {
		fut.Reject(err)
		return
	}

	var o messages.Object
	if obj.Stream { // return a stream from python
		values := make(chan messages.Object)
		ls := messages.NewLocalStream(values, obj.GetLanguage())
		v.streams[ret.CorrID] = ls
		o = ls
	} else {
		o = obj
	}
	fut.Resolve(o)
}

func (v *VirtualEnv) onStreamChunk(chunk *proto.StreamChunk) {
	stream, ok := v.streams[chunk.StreamID]
	if !ok {
		return
	}

	if chunk.EoS {
		stream.EnqueueChunk(nil)
	} else {
		stream.EnqueueChunk(chunk.GetValue())
	}
}

func (v *VirtualEnv) onReceive(msg *executor.Message) {
	switch cmd := msg.Command.(type) {
	case *executor.Message_StreamChunk:
		v.onStreamChunk(cmd.StreamChunk)
	case *executor.Message_Return:
		v.onReturn(cmd.Return)
	}
}

func (v *VirtualEnv) Run(addr string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.started {
		return
	}
	v.started = true
	go func() {
		cmd := exec.CommandContext(v.ctx, v.Exec, path.Join(venvPath, v.Name, venvStart), "--remote", addr, "--venv", v.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return
		}
	}()
	go func() {
		for msg := range v.handler.RecvChan() {
			v.onReceive(msg)
		}
	}()
	<-v.handler.Ready()
}
