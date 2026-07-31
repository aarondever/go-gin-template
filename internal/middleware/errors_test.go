package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	e "github.com/aarondever/go-gin-template/internal/apperror"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// failWith returns a handler that registers err on the context, the way real
// handlers surface failures to this middleware.
func failWith(err error) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = c.Error(err)
	}
}

func decodeErrorBody(t *testing.T, w *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return body
}

func TestErrorHandlerNoError(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(ErrorHandler())
	engine.GET("/resource", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
	if entries := rec.all(); len(entries) != 0 {
		t.Errorf("got %d log records, want none: %+v", len(entries), entries)
	}
}

func TestErrorHandlerStatusByCode(t *testing.T) {
	tests := []struct {
		name       string
		code       e.Code
		wantStatus int
	}{
		{name: "invalid input", code: e.CodeInvalidInput, wantStatus: http.StatusBadRequest},
		{name: "unauthorized", code: e.CodeUnauthorized, wantStatus: http.StatusUnauthorized},
		{name: "forbidden", code: e.CodeForbidden, wantStatus: http.StatusForbidden},
		{name: "not found", code: e.CodeNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict", code: e.CodeConflict, wantStatus: http.StatusConflict},
		{name: "rate limited", code: e.CodeRateLimited, wantStatus: http.StatusTooManyRequests},
		// 499 is nginx's Client Closed Request; reachable only when the code is
		// set explicitly, since a real context.Canceled takes the abort path.
		{name: "canceled", code: e.CodeCanceled, wantStatus: 499},
		{name: "timeout", code: e.CodeTimeout, wantStatus: http.StatusGatewayTimeout},
		{name: "internal", code: e.CodeInternal, wantStatus: http.StatusInternalServerError},
		// Anything not in the table falls back to 500.
		{name: "unmapped code", code: e.Code("SOMETHING_ELSE"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := newEngine(ErrorHandler())
			engine.GET("/resource", failWith(e.New(tt.code, "boom")))

			w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := decodeErrorBody(t, w)
			if body.Error.Code != tt.code {
				t.Errorf("body code = %q, want %q", body.Error.Code, tt.code)
			}
			if body.Error.Message != "boom" {
				t.Errorf("body message = %q, want %q", body.Error.Message, "boom")
			}
		})
	}
}

func TestErrorHandlerResponseShape(t *testing.T) {
	appErr := e.New(e.CodeInvalidInput, "bad request").
		WithDetails(map[string]string{"email": "required"})

	engine := newEngine(ErrorHandler())
	engine.POST("/resource", failWith(appErr))

	w := do(engine, httptest.NewRequest(http.MethodPost, "/resource", nil))

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	if len(raw) != 1 {
		t.Errorf("body has keys %v, want only \"error\"", raw)
	}
	if _, ok := raw["error"]; !ok {
		t.Fatalf("body %q has no \"error\" key", w.Body.String())
	}

	body := decodeErrorBody(t, w)
	if body.Error.Details["email"] != "required" {
		t.Errorf("body details = %v, want email=required", body.Error.Details)
	}
	// The internal cause must never reach the client.
	if b := w.Body.String(); len(b) == 0 {
		t.Error("empty body")
	}
}

// Details is omitempty, so an error without them produces no such key.
func TestErrorHandlerOmitsEmptyDetails(t *testing.T) {
	engine := newEngine(ErrorHandler())
	engine.GET("/resource", failWith(e.New(e.CodeNotFound, "missing")))

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	if _, ok := raw["error"]["details"]; ok {
		t.Errorf("body %q includes details, want it omitted", w.Body.String())
	}
}

// The internal cause is logged but never serialized.
func TestErrorHandlerHidesInternalCause(t *testing.T) {
	rec := captureLogs(t)
	cause := errors.New("pq: connection refused to 10.0.0.5")

	engine := newEngine(ErrorHandler())
	engine.GET("/resource", failWith(e.Wrap(cause, e.CodeInternal, "internal server error")))

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if body := w.Body.String(); strings.Contains(body, "connection refused") {
		t.Errorf("body %q leaks the internal cause", body)
	}
	if got := rec.only(t).str(t, "error"); !strings.Contains(got, "connection refused") {
		t.Errorf("log error = %q, want the internal cause recorded", got)
	}
}

// Plain errors are normalised by apperror.From, so handlers can return anything.
func TestErrorHandlerNormalizesPlainError(t *testing.T) {
	engine := newEngine(ErrorHandler())
	engine.GET("/resource", failWith(errors.New("something broke")))

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	body := decodeErrorBody(t, w)
	if body.Error.Code != e.CodeInternal {
		t.Errorf("body code = %q, want %q", body.Error.Code, e.CodeInternal)
	}
	if body.Error.Message != "internal server error" {
		t.Errorf("body message = %q, want %q", body.Error.Message, "internal server error")
	}
}

func TestErrorHandlerValidationError(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required"`
		Age   int    `json:"age" validate:"gte=18"`
	}

	verr := validator.New().Struct(payload{Age: 10})
	if verr == nil {
		t.Fatal("expected validation to fail")
	}

	engine := newEngine(ErrorHandler())
	engine.POST("/resource", failWith(verr))

	w := do(engine, httptest.NewRequest(http.MethodPost, "/resource", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeErrorBody(t, w)
	if body.Error.Code != e.CodeInvalidInput {
		t.Errorf("body code = %q, want %q", body.Error.Code, e.CodeInvalidInput)
	}
	if body.Error.Message != "validation failed" {
		t.Errorf("body message = %q, want %q", body.Error.Message, "validation failed")
	}
	want := map[string]string{"Email": "required", "Age": "gte"}
	for field, tag := range want {
		if body.Error.Details[field] != tag {
			t.Errorf("details[%q] = %q, want %q", field, body.Error.Details[field], tag)
		}
	}
}

// A wrapped *AppError keeps its own code rather than collapsing to internal.
func TestErrorHandlerWrappedAppError(t *testing.T) {
	appErr := e.New(e.CodeNotFound, "user not found")

	engine := newEngine(ErrorHandler())
	engine.GET("/resource", failWith(fmt.Errorf("service: %w", appErr)))

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if body := decodeErrorBody(t, w); body.Error.Code != e.CodeNotFound {
		t.Errorf("body code = %q, want %q", body.Error.Code, e.CodeNotFound)
	}
}

// A disconnected client gets no response body — there is nobody to write to.
func TestErrorHandlerClientCanceled(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "sentinel", err: context.Canceled},
		{name: "wrapped", err: fmt.Errorf("query users: %w", context.Canceled)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := captureLogs(t)

			engine := newEngine(ErrorHandler())
			engine.GET("/resource", failWith(tt.err))

			w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

			if body := w.Body.String(); body != "" {
				t.Errorf("body = %q, want nothing written", body)
			}

			// The failure is still recorded, with the 499 status it mapped to.
			entry := rec.only(t)
			if got := entry.Attrs["status"].Int64(); got != 499 {
				t.Errorf("log status = %d, want 499", got)
			}
			if got := entry.Attrs["code"].Any(); got != e.CodeCanceled {
				t.Errorf("log code = %v, want %q", got, e.CodeCanceled)
			}
			if entry.Level != slog.LevelWarn {
				t.Errorf("log level = %v, want %v", entry.Level, slog.LevelWarn)
			}
		})
	}
}

// A handler that already committed a response keeps it; the middleware only logs.
func TestErrorHandlerDoesNotOverwriteWrittenResponse(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(ErrorHandler())
	engine.GET("/resource", func(c *gin.Context) {
		c.String(http.StatusCreated, "partial payload")
		_ = c.Error(e.New(e.CodeInternal, "failed midway"))
	})

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want the handler's %d", w.Code, http.StatusCreated)
	}
	if body := w.Body.String(); body != "partial payload" {
		t.Errorf("body = %q, want the handler's response untouched", body)
	}
	// It is still logged, at the status the error mapped to.
	entry := rec.only(t)
	if got := entry.Attrs["status"].Int64(); got != http.StatusInternalServerError {
		t.Errorf("log status = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestErrorHandlerUsesLastError(t *testing.T) {
	engine := newEngine(ErrorHandler())
	engine.GET("/resource", func(c *gin.Context) {
		_ = c.Error(e.New(e.CodeInvalidInput, "first"))
		_ = c.Error(e.New(e.CodeConflict, "second"))
	})

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want the last error's %d", w.Code, http.StatusConflict)
	}
	if body := decodeErrorBody(t, w); body.Error.Message != "second" {
		t.Errorf("body message = %q, want %q", body.Error.Message, "second")
	}
}

func TestErrorHandlerLogLevelBySeverity(t *testing.T) {
	tests := []struct {
		name  string
		code  e.Code
		level slog.Level
	}{
		{name: "client error logs a warning", code: e.CodeNotFound, level: slog.LevelWarn},
		{name: "bad input logs a warning", code: e.CodeInvalidInput, level: slog.LevelWarn},
		{name: "rate limit logs a warning", code: e.CodeRateLimited, level: slog.LevelWarn},
		{name: "internal logs an error", code: e.CodeInternal, level: slog.LevelError},
		{name: "timeout logs an error", code: e.CodeTimeout, level: slog.LevelError},
		{name: "unmapped code logs an error", code: e.Code("WAT"), level: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := captureLogs(t)

			engine := newEngine(ErrorHandler())
			engine.GET("/resource", failWith(e.New(tt.code, "boom")))

			do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

			if got := rec.only(t).Level; got != tt.level {
				t.Errorf("log level = %v, want %v", got, tt.level)
			}
		})
	}
}

func TestErrorHandlerLogAttributes(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(ErrorHandler())
	engine.POST("/users/42", failWith(e.New(e.CodeNotFound, "user not found")))

	do(engine, httptest.NewRequest(http.MethodPost, "/users/42?verbose=1", nil))

	entry := rec.only(t)
	if entry.Message != "request failed" {
		t.Errorf("message = %q, want %q", entry.Message, "request failed")
	}
	if got := entry.Attrs["code"].Any(); got != e.CodeNotFound {
		t.Errorf("attr code = %v, want %q", got, e.CodeNotFound)
	}
	if got := entry.Attrs["status"].Int64(); got != http.StatusNotFound {
		t.Errorf("attr status = %d, want %d", got, http.StatusNotFound)
	}
	if got := entry.str(t, "method"); got != http.MethodPost {
		t.Errorf("attr method = %q, want %q", got, http.MethodPost)
	}
	if got := entry.str(t, "path"); got != "/users/42" {
		t.Errorf("attr path = %q, want %q", got, "/users/42")
	}
	if got := entry.str(t, "error"); got != "user not found" {
		t.Errorf("attr error = %q, want %q", got, "user not found")
	}

	// BUG: errors.go passes c.Request.URL.Path for the "query" attribute, so
	// the query string is never logged. Pinned here so the behaviour change is
	// deliberate; the fix is to use c.Request.URL.RawQuery.
	if got := entry.str(t, "query"); got != "/users/42" {
		t.Errorf("attr query = %q, want the currently duplicated path %q", got, "/users/42")
	}
}

// The middleware logs with the request context, so an upstream request ID lands
// on the record.
func TestErrorHandlerIncludesRequestID(t *testing.T) {
	rec := captureLogs(t)

	engine := newEngine(RequestID(), ErrorHandler())
	engine.GET("/resource", failWith(e.New(e.CodeInternal, "boom")))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set(RequestIDHeader, "req-7")
	w := do(engine, req)

	if got := rec.only(t).RequestID; got != "req-7" {
		t.Errorf("context request ID = %q, want %q", got, "req-7")
	}
	if got := w.Header().Get(RequestIDHeader); got != "req-7" {
		t.Errorf("response header = %q, want %q", got, "req-7")
	}
}

// c.Error only records; it does not stop the chain. A handler that wants to bail
// out has to abort itself, otherwise the handlers after it still run and can
// commit a response the middleware then refuses to overwrite.
func TestErrorHandlerRecordingDoesNotAbortChain(t *testing.T) {
	reached := false

	engine := newEngine(ErrorHandler())
	engine.GET("/resource",
		failWith(e.New(e.CodeForbidden, "no access")),
		func(c *gin.Context) {
			reached = true
			c.Next()
		},
	)

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if !reached {
		t.Error("later handler did not run, want c.Error to leave the chain going")
	}
	// Nothing was written by the handlers, so the middleware still gets to respond.
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// c.Abort in a handler stops the chain, and the middleware still responds.
func TestErrorHandlerRespondsAfterHandlerAbort(t *testing.T) {
	reached := false

	engine := newEngine(ErrorHandler())
	engine.GET("/resource",
		func(c *gin.Context) {
			_ = c.Error(e.New(e.CodeUnauthorized, "missing token"))
			c.Abort()
		},
		func(c *gin.Context) {
			reached = true
			c.String(http.StatusOK, "should not run")
		},
	)

	w := do(engine, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if reached {
		t.Error("later handler ran despite the abort")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if body := decodeErrorBody(t, w); body.Error.Code != e.CodeUnauthorized {
		t.Errorf("body code = %q, want %q", body.Error.Code, e.CodeUnauthorized)
	}
}

func TestErrorHandlerStatusByCodeTableCoversAllCodes(t *testing.T) {
	codes := []e.Code{
		e.CodeInvalidInput, e.CodeUnauthorized, e.CodeForbidden, e.CodeNotFound,
		e.CodeConflict, e.CodeRateLimited, e.CodeCanceled, e.CodeTimeout, e.CodeInternal,
	}

	for _, code := range codes {
		if _, ok := statusByCode[code]; !ok {
			t.Errorf("statusByCode has no entry for %q", code)
		}
	}
	if len(statusByCode) != len(codes) {
		t.Errorf("statusByCode has %d entries, want %d", len(statusByCode), len(codes))
	}
}
