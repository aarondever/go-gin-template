package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// gin's default debug mode writes a warning banner to stderr for every engine
// the tests build.
func init() { gin.SetMode(gin.TestMode) }

// newEngine builds a bare gin engine (no default middleware) with mw installed.
// Shared by the tests for every middleware in this package.
func newEngine(mw ...gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.Use(mw...)
	return engine
}

func do(engine *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

const (
	allowHeaders = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"
	allowMethods = "POST, OPTIONS, GET, PUT, DELETE"
)

// wantCORSHeaders is the full set the middleware writes on every response.
var wantCORSHeaders = map[string]string{
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Allow-Credentials": "true",
	"Access-Control-Allow-Headers":     allowHeaders,
	"Access-Control-Allow-Methods":     allowMethods,
}

func assertCORSHeaders(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	for key, want := range wantCORSHeaders {
		if got := w.Header().Get(key); got != want {
			t.Errorf("header %s = %q, want %q", key, got, want)
		}
	}
}

func TestCORSSetsHeaders(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			called := false
			engine := newEngine(CORS())
			engine.Handle(method, "/resource", func(c *gin.Context) {
				called = true
				c.String(http.StatusOK, "ok")
			})

			w := do(engine, httptest.NewRequest(method, "/resource", nil))

			if !called {
				t.Error("handler was not called")
			}
			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if body := w.Body.String(); body != "ok" {
				t.Errorf("body = %q, want %q", body, "ok")
			}
			assertCORSHeaders(t, w)
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	called := false
	engine := newEngine(CORS())
	engine.Handle(http.MethodOptions, "/resource", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "should not run")
	})

	w := do(engine, httptest.NewRequest(http.MethodOptions, "/resource", nil))

	if called {
		t.Error("handler ran, want the preflight to short-circuit it")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if body := w.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	// The browser needs the headers on the preflight response above all.
	assertCORSHeaders(t, w)
}

// Preflights routinely target paths that have no OPTIONS route registered; gin
// runs the global middleware for those through its no-route chain.
func TestCORSPreflightWithoutRoute(t *testing.T) {
	engine := newEngine(CORS())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := do(engine, httptest.NewRequest(http.MethodOptions, "/resource", nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	assertCORSHeaders(t, w)
}

// The preflight check compares against the canonical upper-case method, so a
// lower-case one is just an unrouted request that still gets the headers.
func TestCORSLowercaseOptionsIsNotPreflight(t *testing.T) {
	engine := newEngine(CORS())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := do(engine, httptest.NewRequest("options", "/resource", nil))

	if w.Code == http.StatusNoContent {
		t.Errorf("status = %d, want the request not to be treated as a preflight", w.Code)
	}
	assertCORSHeaders(t, w)
}

// Aborting must stop the rest of the chain, not just the final handler.
func TestCORSPreflightAbortsLaterMiddleware(t *testing.T) {
	reached := false
	engine := newEngine(CORS(), func(c *gin.Context) {
		reached = true
		c.Next()
	})
	engine.Handle(http.MethodOptions, "/resource", func(c *gin.Context) {})

	w := do(engine, httptest.NewRequest(http.MethodOptions, "/resource", nil))

	if reached {
		t.Error("downstream middleware ran, want the chain aborted")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// Headers are written before the handler runs, so they survive a handler error.
func TestCORSHeadersSurviveHandlerError(t *testing.T) {
	engine := newEngine(CORS())
	engine.GET("/resource", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	assertCORSHeaders(t, w)
}

// The Origin request header is ignored; the response is always a wildcard.
func TestCORSIgnoresRequestOrigin(t *testing.T) {
	engine := newEngine(CORS())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := do(engine, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}
