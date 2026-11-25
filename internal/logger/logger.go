package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aarondever/go-gin-template/config"
)

// Init configures the default slog logger, running extractors on every context-aware call.
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
		AddSource:   true,
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

// Err is the "error" attr, empty for a nil err so the handler drops it.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String("error", err.Error())
}
