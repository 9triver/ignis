package utils

import (
	"io"
	"log/slog"
	"os"

	"github.com/asynkron/protoactor-go/actor"
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
