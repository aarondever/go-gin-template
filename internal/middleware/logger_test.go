package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/aarondever/go-gin-template/internal/util"
	"github.com/gin-gonic/gin"
)

// logEntry is a single captured log record, flattened for easy assertions.
type logEntry struct {
	Level     slog.Level
	Message   string
	Attrs     map[string]slog.Value
	RequestID string // pulled off the context the record was logged with
}

func (e logEntry) str(t *testing.T, key string) string {
	t.Helper()
	v, ok := e.Attrs[key]
	if !ok {
		t.Fatalf("log record %q has no attribute %q (attrs: %v)", e.Message, key, e.Attrs)
	}
	return v.String()
}

// logRecorder is a slog.Handler that keeps every record in memory. The middleware
// logs through the package-level slog default, so swapping the default out is the
// seam these tests use. Shared with the ErrorHandler tests.
type logRecorder struct {
	mu      sync.Mutex
	entries []logEntry
}

func (r *logRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *logRecorder) Handle(ctx context.Context, rec slog.Record) error {
	entry := logEntry{
		Level:   rec.Level,
		Message: rec.Message,
		Attrs:   make(map[string]slog.Value, rec.NumAttrs()),
	}
	rec.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value
		return true
	})
	if id, ok := util.ExtractRequestID(ctx); ok {
		entry.RequestID = id
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
	return nil
}

func (r *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *logRecorder) WithGroup(string) slog.Handler      { return r }

func (r *logRecorder) all() []logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]logEntry(nil), r.entries...)
}

// only asserts exactly one record was captured and returns it.
func (r *logRecorder) only(t *testing.T) logEntry {
	t.Helper()
	entries := r.all()
	if len(entries) != 1 {
		t.Fatalf("got %d log records, want 1: %+v", len(entries), entries)
	}
	return entries[0]
}

// captureLogs installs a recording handler as the slog default for the test.
func captureLogs(t *testing.T) *logRecorder {
	t.Helper()
	rec := &logRecorder{}
	orig := slog.Default()
	slog.SetDefault(slog.New(rec))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return rec
}

func TestLoggerLogsRequest(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/users", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/users?page=2&size=10", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	do(engine, req)

	entry := rec.only(t)

	if entry.Level != slog.LevelInfo {
		t.Errorf("level = %v, want %v", entry.Level, slog.LevelInfo)
	}
	if entry.Message != "HTTP Request" {
		t.Errorf("message = %q, want %q", entry.Message, "HTTP Request")
	}

	wantAttrs := map[string]string{
		"method":     http.MethodGet,
		"path":       "/users",
		"query":      "page=2&size=10",
		"ip":         "192.0.2.1", // httptest's default RemoteAddr
		"user_agent": "test-agent/1.0",
	}
	for key, want := range wantAttrs {
		if got := entry.str(t, key); got != want {
			t.Errorf("attr %s = %q, want %q", key, got, want)
		}
	}

	status, ok := entry.Attrs["status"]
	if !ok {
		t.Fatalf("no status attribute (attrs: %v)", entry.Attrs)
	}
	if status.Int64() != http.StatusOK {
		t.Errorf("attr status = %d, want %d", status.Int64(), http.StatusOK)
	}

	latency, ok := entry.Attrs["latency"]
	if !ok {
		t.Fatalf("no latency attribute (attrs: %v)", entry.Attrs)
	}
	if latency.Kind() != slog.KindDuration {
		t.Errorf("attr latency kind = %v, want %v", latency.Kind(), slog.KindDuration)
	}
	if latency.Duration() <= 0 {
		t.Errorf("attr latency = %v, want a positive duration", latency.Duration())
	}
}

func TestLoggerRecordsHandlerStatus(t *testing.T) {
	tests := []struct {
		name   string
		handle gin.HandlerFunc
		want   int64
	}{
		{
			name:   "created",
			handle: func(c *gin.Context) { c.String(http.StatusCreated, "made") },
			want:   http.StatusCreated,
		},
		{
			name:   "bad request",
			handle: func(c *gin.Context) { c.String(http.StatusBadRequest, "nope") },
			want:   http.StatusBadRequest,
		},
		{
			name:   "server error",
			handle: func(c *gin.Context) { c.AbortWithStatus(http.StatusInternalServerError) },
			want:   http.StatusInternalServerError,
		},
		{
			// A handler that writes nothing still reports gin's default.
			name:   "no write defaults to 200",
			handle: func(c *gin.Context) {},
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := captureLogs(t)

			engine := newEngine(Logger())
			engine.GET("/resource", tt.handle)

			do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

			entry := rec.only(t)
			if got := entry.Attrs["status"].Int64(); got != tt.want {
				t.Errorf("attr status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			rec := captureLogs(t)

			engine := newEngine(Logger())
			engine.Handle(method, "/resource", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			do(engine, httptest.NewRequest(method, "/resource", nil))

			if got := rec.only(t).str(t, "method"); got != method {
				t.Errorf("attr method = %q, want %q", got, method)
			}
		})
	}
}

func TestLoggerEmptyQuery(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if got := rec.only(t).str(t, "query"); got != "" {
		t.Errorf("attr query = %q, want empty", got)
	}
}

func TestLoggerSkipPaths(t *testing.T) {
	tests := []struct {
		name    string
		skip    []string
		target  string
		wantLog bool
	}{
		{name: "no skip list", skip: nil, target: "/health", wantLog: true},
		{name: "skipped path", skip: []string{"/health"}, target: "/health", wantLog: false},
		{name: "unskipped path", skip: []string{"/health"}, target: "/users", wantLog: true},
		{
			name:    "second entry in the skip list",
			skip:    []string{"/health", "/metrics"},
			target:  "/metrics",
			wantLog: false,
		},
		{
			// Matching is exact, not by prefix.
			name:    "sub-path of a skipped path is still logged",
			skip:    []string{"/health"},
			target:  "/health/live",
			wantLog: true,
		},
		{
			// The skip list is matched against URL.Path, so a query string
			// does not stop a path from being skipped.
			name:    "skipped path with a query string",
			skip:    []string{"/health"},
			target:  "/health?verbose=1",
			wantLog: false,
		},
		{
			name:    "skip entry that matches nothing",
			skip:    []string{"/nowhere"},
			target:  "/users",
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := captureLogs(t)
			called := false

			engine := newEngine(Logger(tt.skip...))
			handler := func(c *gin.Context) {
				called = true
				c.String(http.StatusOK, "ok")
			}
			engine.GET("/health", handler)
			engine.GET("/health/live", handler)
			engine.GET("/metrics", handler)
			engine.GET("/users", handler)

			w := do(engine, httptest.NewRequest(http.MethodGet, tt.target, nil))

			// Skipping only suppresses the log; the request is served as usual.
			if !called {
				t.Error("handler was not called")
			}
			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			entries := rec.all()
			if tt.wantLog && len(entries) != 1 {
				t.Errorf("got %d log records, want 1: %+v", len(entries), entries)
			}
			if !tt.wantLog && len(entries) != 0 {
				t.Errorf("got %d log records, want none: %+v", len(entries), entries)
			}
		})
	}
}

// Skipping one path must not disable logging for the rest of the requests.
func TestLoggerSkipDoesNotLeakBetweenRequests(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger("/health"))
	engine.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	engine.GET("/users", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do(engine, httptest.NewRequest(http.MethodGet, "/health", nil))
	do(engine, httptest.NewRequest(http.MethodGet, "/users", nil))
	do(engine, httptest.NewRequest(http.MethodGet, "/health", nil))

	entries := rec.all()
	if len(entries) != 1 {
		t.Fatalf("got %d log records, want only the /users one: %+v", len(entries), entries)
	}
	if got := entries[0].str(t, "path"); got != "/users" {
		t.Errorf("attr path = %q, want %q", got, "/users")
	}
}

// The middleware logs with the request context, so a request ID injected
// upstream is carried into the record.
func TestLoggerIncludesRequestIDFromContext(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(RequestID(), Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set(RequestIDHeader, "req-42")
	do(engine, req)

	if got := rec.only(t).RequestID; got != "req-42" {
		t.Errorf("context request ID = %q, want %q", got, "req-42")
	}
}

// Ordering matters: with Logger first, the ID is not on the context yet when
// the deferred log runs, because RequestID replaces c.Request further down.
func TestLoggerWithoutRequestIDMiddleware(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if got := rec.only(t).RequestID; got != "" {
		t.Errorf("context request ID = %q, want empty", got)
	}
}

// Logging happens after c.Next returns, so an unrouted request is logged too.
func TestLoggerLogsUnroutedRequest(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do(engine, httptest.NewRequest(http.MethodGet, "/missing", nil))

	entry := rec.only(t)
	if got := entry.Attrs["status"].Int64(); got != http.StatusNotFound {
		t.Errorf("attr status = %d, want %d", got, http.StatusNotFound)
	}
	if got := entry.str(t, "path"); got != "/missing" {
		t.Errorf("attr path = %q, want %q", got, "/missing")
	}
}

func TestLoggerEmptyUserAgent(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if got := rec.only(t).str(t, "user_agent"); got != "" {
		t.Errorf("attr user_agent = %q, want empty", got)
	}
}

// c.ClientIP honours the forwarding headers gin is configured to trust.
func TestLoggerClientIPFromForwardedHeader(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(Logger())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	if err := engine.SetTrustedProxies([]string{"192.0.2.1"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	do(engine, req)

	if got := rec.only(t).str(t, "ip"); got != "203.0.113.7" {
		t.Errorf("attr ip = %q, want %q", got, "203.0.113.7")
	}
}
