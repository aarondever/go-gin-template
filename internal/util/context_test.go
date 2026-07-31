package util

import (
	"context"
	"testing"
)

func TestInjectExtractRequestID(t *testing.T) {
	ctx := InjectRequestID(context.Background(), "req-123")

	got, ok := ExtractRequestID(ctx)
	if !ok {
		t.Fatal("ExtractRequestID() ok = false, want true")
	}
	if got != "req-123" {
		t.Errorf("ExtractRequestID() = %q, want %q", got, "req-123")
	}
}

func TestExtractRequestIDMissing(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "background", ctx: context.Background()},
		{name: "todo", ctx: context.TODO()},
		{
			name: "unrelated value",
			ctx:  context.WithValue(context.Background(), otherKey{}, "req-123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractRequestID(tt.ctx)

			if ok {
				t.Errorf("ExtractRequestID() ok = true, want false")
			}
			if got != "" {
				t.Errorf("ExtractRequestID() = %q, want empty", got)
			}
		})
	}
}

// otherKey is a distinct type, so a value stored under it cannot be mistaken for
// the request ID even though both keys are empty structs.
type otherKey struct{}

// An empty ID is reported as missing rather than as a present empty string.
func TestExtractRequestIDEmptyString(t *testing.T) {
	ctx := InjectRequestID(context.Background(), "")

	got, ok := ExtractRequestID(ctx)

	if ok {
		t.Error("ExtractRequestID() ok = true, want false for an empty ID")
	}
	if got != "" {
		t.Errorf("ExtractRequestID() = %q, want empty", got)
	}
}

// A non-string value under the key fails the type assertion instead of panicking.
func TestExtractRequestIDWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey{}, 42)

	got, ok := ExtractRequestID(ctx)

	if ok {
		t.Error("ExtractRequestID() ok = true, want false")
	}
	if got != "" {
		t.Errorf("ExtractRequestID() = %q, want empty", got)
	}
}

func TestInjectRequestIDOverwrites(t *testing.T) {
	ctx := InjectRequestID(context.Background(), "first")
	ctx = InjectRequestID(ctx, "second")

	got, ok := ExtractRequestID(ctx)

	if !ok {
		t.Fatal("ExtractRequestID() ok = false, want true")
	}
	if got != "second" {
		t.Errorf("ExtractRequestID() = %q, want %q", got, "second")
	}
}

// Injecting must not disturb the parent, which callers may still hold.
func TestInjectRequestIDLeavesParentAlone(t *testing.T) {
	parent := context.Background()
	child := InjectRequestID(parent, "req-123")

	if _, ok := ExtractRequestID(parent); ok {
		t.Error("parent context gained a request ID")
	}
	if _, ok := ExtractRequestID(child); !ok {
		t.Error("child context has no request ID")
	}
}

func TestInjectRequestIDPreservesOtherValues(t *testing.T) {
	ctx := context.WithValue(context.Background(), otherKey{}, "kept")
	ctx = InjectRequestID(ctx, "req-123")

	if got, _ := ExtractRequestID(ctx); got != "req-123" {
		t.Errorf("ExtractRequestID() = %q, want %q", got, "req-123")
	}
	if got := ctx.Value(otherKey{}); got != "kept" {
		t.Errorf("other value = %v, want %q", got, "kept")
	}
}

// The ID has to survive the derived contexts handlers create.
func TestExtractRequestIDThroughDerivedContexts(t *testing.T) {
	base := InjectRequestID(context.Background(), "req-123")

	cancelCtx, cancel := context.WithCancel(base)
	defer cancel()
	valueCtx := context.WithValue(cancelCtx, otherKey{}, "extra")

	for name, ctx := range map[string]context.Context{
		"cancel": cancelCtx,
		"value":  valueCtx,
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := ExtractRequestID(ctx)

			if !ok {
				t.Fatal("ExtractRequestID() ok = false, want true")
			}
			if got != "req-123" {
				t.Errorf("ExtractRequestID() = %q, want %q", got, "req-123")
			}
		})
	}

	// Cancellation does not remove values.
	cancel()
	if _, ok := ExtractRequestID(cancelCtx); !ok {
		t.Error("request ID lost after cancellation")
	}
}

// A child can shadow the ID without affecting the context it was derived from.
func TestInjectRequestIDShadowing(t *testing.T) {
	outer := InjectRequestID(context.Background(), "outer")
	inner := InjectRequestID(outer, "inner")

	if got, _ := ExtractRequestID(outer); got != "outer" {
		t.Errorf("outer ID = %q, want %q", got, "outer")
	}
	if got, _ := ExtractRequestID(inner); got != "inner" {
		t.Errorf("inner ID = %q, want %q", got, "inner")
	}
}

// Contexts are read concurrently by handlers and their goroutines.
func TestExtractRequestIDConcurrent(t *testing.T) {
	ctx := InjectRequestID(context.Background(), "req-123")

	done := make(chan bool, 50)
	for range 50 {
		go func() {
			got, ok := ExtractRequestID(ctx)
			done <- ok && got == "req-123"
		}()
	}
	for range 50 {
		if !<-done {
			t.Error("concurrent ExtractRequestID() returned the wrong value")
		}
	}
}
