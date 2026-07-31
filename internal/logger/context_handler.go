package logger

import (
	"context"
	"log/slog"

	"github.com/aarondever/go-gin-template/internal/util"
)

// ContextAttrFunc derives log attributes from a request context. Register
// implementations with [Init] so every context-aware log call picks them up.
type ContextAttrFunc func(ctx context.Context) []slog.Attr

// contextHandler wraps a slog.Handler and adds the attributes produced by the
// registered extractors. The built-in slog handlers ignore the context they are
// handed, so values such as a request ID have to be pulled out here.
type contextHandler struct {
	slog.Handler
	extractors []ContextAttrFunc
}

func newContextHandler(h slog.Handler, extractors []ContextAttrFunc) *contextHandler {
	return &contextHandler{Handler: h, extractors: extractors}
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, extract := range h.extractors {
		if attrs := extract(ctx); len(attrs) > 0 {
			r.AddAttrs(attrs...)
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newContextHandler(h.Handler.WithAttrs(attrs), h.extractors)
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return newContextHandler(h.Handler.WithGroup(name), h.extractors)
}

// WithRequestID exposes the request ID on ctx as a log attribute. Wire it into
// logger.Init so every context-aware log call carries it.
func WithRequestID(ctx context.Context) []slog.Attr {
	requestID, ok := util.ExtractRequestID(ctx)
	if !ok {
		return nil
	}
	return []slog.Attr{slog.String("request_id", requestID)}
}
