package utils

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/lmittmann/tint"

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
		return slog.New(tint.NewHandler(io.MultiWriter(writers...), &tint.Options{
			AddSource:  true,
			Level:      slog.LevelError,
			TimeFormat: time.DateTime,
		})).With("system", system.ID)
	})
}

func GenID() string {
	return shortuuid.New()
}

func GenIDWith(prefix string) string {
	return prefix + shortuuid.New()
}
