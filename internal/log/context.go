package log

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger = NewLogger(os.Stderr, slog.LevelInfo)

type loggerContextKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return defaultLogger
}
