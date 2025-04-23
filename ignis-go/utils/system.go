package utils

import (
	"io"
	"log/slog"
	"os"
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
