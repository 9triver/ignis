package utils

import (
	"github.com/asynkron/protoactor-go/actor"
	"io"
	"log/slog"
	"os"

	"github.com/lithammer/shortuuid/v4"
)

func WithLogger(logPaths ...string) actor.ConfigOption {
	writers := []io.Writer{os.Stderr}
	for _, log := range logPaths {
		w, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		writers = append(writers, w)
	}

	return actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})).With("system", system.ID)
	})
}

func GenID() string {
	return shortuuid.New()
}

func GenIDWith(prefix string) string {
	return prefix + shortuuid.New()
}
