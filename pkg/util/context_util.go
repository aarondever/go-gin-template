package util

import (
	"context"
)

type requestIDKey struct{}

// InjectRequestID stores a request ID in the context.
func InjectRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// ExtractRequestID returns the request ID from ctx, or false if not found.
func ExtractRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey{}).(string)
	return requestID, ok && requestID != ""
}
