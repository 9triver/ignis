package utils

import (
	"io"
	"log/slog"
	"os"

	"github.com/asynkron/protoactor-go/actor"

	"github.com/9triver/ignis/configs"
	"github.com/9triver/ignis/proto"
	"github.com/9triver/ignis/utils/errors"
)

func Logger(logPaths ...string) *slog.Logger {
	writers := []io.Writer{os.Stderr}
	for _, log := range logPaths {
		w, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		writers = append(writers, w)
	}

	return slog.New(slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func IsSameSystem(pids ...*actor.PID) bool {
	if len(pids) == 0 {
		return true
	}

	address := pids[0].Address
	for _, pid := range pids {
		if pid == nil {
			continue
		}
		if pid.Address != address {
			return false
		}
	}

	return true
}

func GetObject(ctx actor.Context, flow *proto.Flow) Future[proto.Object] {
	fut := NewFuture[proto.Object](configs.FlowTimeout)
	if flow == nil {
		fut.Reject(errors.New("flow is nil"))
		return fut
	}

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *proto.Error:
			fut.Reject(errors.Format("flow %s failed: %s", flow.ObjectID, msg.Message))
		case proto.Object:
			if msg.GetID() != flow.ObjectID {
				fut.Reject(errors.Format("flow %s failed: unexpected ID %s", flow.ObjectID, msg.GetID()))
				return
			}
			fut.Resolve(msg)
		}
	})
	flowActor := ctx.Spawn(props)
	fut.OnDone(func(future proto.Object, err error) { ctx.Stop(flowActor) })

	ctx.Send(flow.Source, &proto.ObjectRequest{
		ID:      flow.ObjectID,
		ReplyTo: flowActor,
	})

	return fut
}
