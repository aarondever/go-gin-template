package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aarondever/go-gin-template/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// captureRequestID returns a handler that records the ID visible on the request
// context, which is where downstream code reads it from.
func captureRequestID(got *string, found *bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		*got, *found = util.ExtractRequestID(c.Request.Context())
		c.String(http.StatusOK, "ok")
	}
}

func TestRequestIDGenerates(t *testing.T) {
	var ctxID string
	var found bool

	engine := newEngine(RequestID())
	engine.GET("/resource", captureRequestID(&ctxID, &found))

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if !found {
		t.Fatal("no request ID on the request context")
	}
	header := w.Header().Get(RequestIDHeader)
	if header == "" {
		t.Fatalf("response header %s is empty", RequestIDHeader)
	}
	if header != ctxID {
		t.Errorf("header ID %q and context ID %q differ, want them identical", header, ctxID)
	}

	// The generated ID comes from util.NewID, which mints UUIDv7.
	parsed, err := uuid.Parse(ctxID)
	if err != nil {
		t.Fatalf("generated ID %q is not a UUID: %v", ctxID, err)
	}
	if v := parsed.Version(); v != 7 {
		t.Errorf("UUID version = %d, want 7", v)
	}
}

func TestRequestIDReusesIncomingHeader(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
	}{
		{name: "uuid", incoming: "3f2b8c1e-0d4a-7b6c-9e1f-2a3b4c5d6e7f"},
		{name: "opaque upstream value", incoming: "edge-proxy-01/abc123"},
		// Nothing validates the header, so any non-empty value is trusted.
		{name: "not an id at all", incoming: "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctxID string
			var found bool

			engine := newEngine(RequestID())
			engine.GET("/resource", captureRequestID(&ctxID, &found))

			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			req.Header.Set(RequestIDHeader, tt.incoming)
			w := do(engine, req)

			if !found {
				t.Fatal("no request ID on the request context")
			}
			if ctxID != tt.incoming {
				t.Errorf("context ID = %q, want the incoming %q", ctxID, tt.incoming)
			}
			if got := w.Header().Get(RequestIDHeader); got != tt.incoming {
				t.Errorf("response header = %q, want the incoming %q", got, tt.incoming)
			}
		})
	}
}

func TestRequestIDGeneratesWhenHeaderEmpty(t *testing.T) {
	var ctxID string
	var found bool

	engine := newEngine(RequestID())
	engine.GET("/resource", captureRequestID(&ctxID, &found))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set(RequestIDHeader, "")
	do(engine, req)

	if !found || ctxID == "" {
		t.Fatal("want a generated request ID when the header is present but empty")
	}
	if _, err := uuid.Parse(ctxID); err != nil {
		t.Errorf("generated ID %q is not a UUID: %v", ctxID, err)
	}
}

// The header lookup is case-insensitive, as HTTP header lookups always are.
func TestRequestIDHeaderLookupIsCaseInsensitive(t *testing.T) {
	var ctxID string
	var found bool

	engine := newEngine(RequestID())
	engine.GET("/resource", captureRequestID(&ctxID, &found))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("x-request-id", "from-lowercase")
	do(engine, req)

	if ctxID != "from-lowercase" {
		t.Errorf("context ID = %q, want %q", ctxID, "from-lowercase")
	}
}

func TestRequestIDIsUniquePerRequest(t *testing.T) {
	engine := newEngine(RequestID())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	seen := make(map[string]bool, 20)
	for range 20 {
		w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

		id := w.Header().Get(RequestIDHeader)
		if id == "" {
			t.Fatal("empty request ID")
		}
		if seen[id] {
			t.Fatalf("request ID %q was reused across requests", id)
		}
		seen[id] = true
	}
}

// Replacing c.Request must not disturb anything else on it.
func TestRequestIDPreservesRequest(t *testing.T) {
	var gotPath, gotQuery, gotHeader string

	engine := newEngine(RequestID())
	engine.GET("/resource", func(c *gin.Context) {
		gotPath = c.Request.URL.Path
		gotQuery = c.Request.URL.RawQuery
		gotHeader = c.GetHeader("Authorization")
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/resource?page=2", nil)
	req.Header.Set("Authorization", "Bearer token")
	do(engine, req)

	if gotPath != "/resource" {
		t.Errorf("path = %q, want %q", gotPath, "/resource")
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want %q", gotQuery, "page=2")
	}
	if gotHeader != "Bearer token" {
		t.Errorf("Authorization = %q, want %q", gotHeader, "Bearer token")
	}
}

// The ID must be on the context before later middleware runs, not only the handler.
func TestRequestIDVisibleToDownstreamMiddleware(t *testing.T) {
	var mwID string
	var found bool

	engine := newEngine(RequestID(), func(c *gin.Context) {
		mwID, found = util.ExtractRequestID(c.Request.Context())
		c.Next()
	})
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if !found {
		t.Fatal("downstream middleware saw no request ID")
	}
	if mwID != w.Header().Get(RequestIDHeader) {
		t.Errorf("middleware ID %q differs from the response header %q",
			mwID, w.Header().Get(RequestIDHeader))
	}
}

// The header is set before the handler runs, so it is present even when the
// handler aborts or fails.
func TestRequestIDHeaderSetOnErrorResponse(t *testing.T) {
	engine := newEngine(RequestID())
	engine.GET("/resource", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if w.Header().Get(RequestIDHeader) == "" {
		t.Errorf("response header %s is empty on an error response", RequestIDHeader)
	}
}

func TestRequestIDHeaderConstant(t *testing.T) {
	if RequestIDHeader != "X-Request-ID" {
		t.Errorf("RequestIDHeader = %q, want %q", RequestIDHeader, "X-Request-ID")
	}
}
