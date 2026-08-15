package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aarondever/go-gin-template/config"
)

// Init configures the default slog logger. Extractors run on every
// context-aware call, turning context values into attributes.
func Init(cfg config.LogConfig, extractors ...ContextAttrFunc) {
	var logLevel slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:       logLevel,
		ReplaceAttr: humanizeDuration,
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	if len(extractors) > 0 {
		handler = newContextHandler(handler, extractors)
	}

	slog.SetDefault(slog.New(handler))
}

// humanizeDuration renders durations as "1.523ms", not a nanosecond count.
func humanizeDuration(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindDuration {
		a.Value = slog.StringValue(a.Value.Duration().Round(time.Microsecond).String())
	}
	return a
}

// Debug logs msg at [slog.LevelDebug].
func Debug(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

// DebugContext logs msg at [slog.LevelDebug], using ctx.
func DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// Info logs msg at [slog.LevelInfo].
func Info(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// InfoContext logs msg at [slog.LevelInfo], using ctx.
func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs msg at [slog.LevelWarn].
func Warn(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// WarnContext logs msg at [slog.LevelWarn], using ctx.
func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs msg at [slog.LevelError].
func Error(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// ErrorContext logs msg at [slog.LevelError], using ctx.
func ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}

// Panic logs msg at [slog.LevelError] and then calls [panic](msg).
func Panic(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
	panic(msg)
}

// PanicContext logs msg at [slog.LevelError], using ctx, and then calls [panic](msg).
func PanicContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	panic(msg)
}

// Fatal logs msg at [slog.LevelError] and then calls [os.Exit](1).
func Fatal(msg string, attrs ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
	os.Exit(1)
}

// FatalContext logs msg at [slog.LevelError], using ctx, and then calls [os.Exit](1).
func FatalContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	os.Exit(1)
}

func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String("error", err.Error())
}
