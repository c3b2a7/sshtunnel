package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const logTimeFormat = "2006-01-02 15:04:05"

func NewLogger(out io.Writer, level slog.Level) *slog.Logger {
	return slog.New(&formatLogHandler{
		out:   out,
		level: level,
		attrs: map[string]slog.Value{},
		mu:    &sync.Mutex{},
	})
}

func WithModuleLogger(ctx context.Context, name string) context.Context {
	return WithLogger(ctx, FromContext(ctx).With("scope", name))
}

func WithTunnelLogger(ctx context.Context, tunnelID string) context.Context {
	return WithLogger(ctx, FromContext(ctx).With("scope", tunnelID))
}

func Infof(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Log(ctx, slog.LevelInfo, fmt.Sprintf(format, args...))
}

func Debugf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Log(ctx, slog.LevelDebug, fmt.Sprintf(format, args...))
}

func Errorf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Log(ctx, slog.LevelError, fmt.Sprintf(format, args...))
}

type formatLogHandler struct {
	out   io.Writer
	level slog.Level
	attrs map[string]slog.Value
	mu    *sync.Mutex
}

func (h *formatLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *formatLogHandler) Handle(_ context.Context, record slog.Record) error {
	scope := "app"
	if value, ok := h.attrs["scope"]; ok {
		scope = value.String()
	}

	attrs := h.attrsWithoutScope()
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "scope" {
			scope = attr.Value.String()
			return true
		}
		attrs = append(attrs, attr)
		return true
	})

	line := fmt.Sprintf("[%s][%s][%s]: %s", record.Time.Format(logTimeFormat), record.Level.String(), scope, record.Message)
	for _, attr := range attrs {
		line += fmt.Sprintf(" %s=%s", attr.Key, attr.Value.String())
	}
	line += "\n"

	mu := h.mutex()
	mu.Lock()
	defer mu.Unlock()
	_, err := h.out.Write([]byte(line))
	return err
}

func (h *formatLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	for _, attr := range attrs {
		next.attrs[attr.Key] = attr.Value
	}
	return next
}

func (h *formatLogHandler) WithGroup(string) slog.Handler {
	return h
}

func (h *formatLogHandler) clone() *formatLogHandler {
	attrs := make(map[string]slog.Value, len(h.attrs))
	for key, value := range h.attrs {
		attrs[key] = value
	}
	return &formatLogHandler{
		out:   h.out,
		level: h.level,
		attrs: attrs,
		mu:    h.mutex(),
	}
}

func (h *formatLogHandler) attrsWithoutScope() []slog.Attr {
	attrs := make([]slog.Attr, 0, len(h.attrs))
	for key, value := range h.attrs {
		if key == "scope" {
			continue
		}
		attrs = append(attrs, slog.Attr{Key: key, Value: value})
	}
	return attrs
}

func (h *formatLogHandler) mutex() *sync.Mutex {
	if h.mu != nil {
		return h.mu
	}
	h.mu = new(sync.Mutex)
	return h.mu
}
