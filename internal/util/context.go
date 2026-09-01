package util

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceID returns the current span's trace id, or "" if the context carries
// no valid span.
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}
