package logger

import (
	"context"
	"log/slog"
	"os"
)

const (
	envDev  = "dev"
	envProd = "production"
)

var fallbackLogger = slog.New(slog.NewJSONHandler(
	os.Stdout,
	&slog.HandlerOptions{
		Level: slog.LevelDebug,
	}),
)

func NewLogger(env string) (log *slog.Logger) {
	switch env {
	case envDev:
		log = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelDebug,
			}),
		)
	case envProd:
		log = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			}),
		)
	}
	return
}

func ErrorAttr(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}

type ctxKeyType string

const (
	ctxKey ctxKeyType = "logger"
)

func FromCtx(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey).(*slog.Logger); ok {
		return logger
	}
	return fallbackLogger
}
