package log

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestNewLoggerFiltersByLevel(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewLogger(&out, slog.LevelWarn))
	ctx = WithModuleLogger(ctx, "main")

	Infof(ctx, "info hidden")
	if out.Len() != 0 {
		t.Fatalf("NewLogger() wrote %q below warn level", out.String())
	}

	Errorf(ctx, "error visible")
	if !strings.Contains(out.String(), "error visible") {
		t.Fatalf("NewLogger() output = %q, want error message", out.String())
	}
}

func TestNewLoggerAllowsInfoLevel(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewLogger(&out, slog.LevelInfo))
	ctx = WithModuleLogger(ctx, "main")
	Infof(ctx, "visible")
	if !strings.Contains(out.String(), "visible") {
		t.Fatalf("NewLogger() output = %q, want info message", out.String())
	}
}

func TestNewLoggerAllowsDebugLevel(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewLogger(&out, slog.LevelDebug))
	ctx = WithModuleLogger(ctx, "main")
	Debugf(ctx, "debug visible")
	if !strings.Contains(out.String(), "debug visible") {
		t.Fatalf("NewLogger() output = %q, want debug message", out.String())
	}
}

func TestLoggerFormatsModuleMessage(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewLogger(&out, slog.LevelWarn))
	ctx = WithModuleLogger(ctx, "main")

	Errorf(ctx, "failed: %s", "boom")

	want := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]\[ERROR\]\[main\]: failed: boom\n$`)
	if !want.MatchString(out.String()) {
		t.Fatalf("logger output = %q, want module log format", out.String())
	}
}

func TestLoggerFormatsTunnelMessage(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewLogger(&out, slog.LevelInfo))
	ctx = WithTunnelLogger(ctx, fmt.Sprintf("tunnel-%d", 2))

	Infof(ctx, "connected")

	want := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]\[INFO\]\[tunnel-2\]: connected\n$`)
	if !want.MatchString(out.String()) {
		t.Fatalf("logger output = %q, want tunnel log format", out.String())
	}
}
