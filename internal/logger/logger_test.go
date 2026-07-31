package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// captureStdout swaps os.Stdout for a pipe while fn runs. Init reads os.Stdout
// at call time, so anything that needs capturing must be initialised inside fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	defer func() {
		os.Stdout = orig
		_ = w.Close()
		_ = r.Close()
	}()

	fn()

	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	out := <-done
	_ = r.Close()
	return out
}

// Init replaces the process-wide default logger, so put the old one back.
func restoreDefaultLogger(t *testing.T) {
	t.Helper()
	orig := slog.Default()
	t.Cleanup(func() { slog.SetDefault(orig) })
}

// initAndLog runs Init inside the capture so the handler writes to the pipe.
func initAndLog(t *testing.T, cfg Config, extractors []ContextAttrFunc, log func()) string {
	t.Helper()
	restoreDefaultLogger(t)
	return captureStdout(t, func() {
		Init(cfg, extractors...)
		log()
	})
}

func decodeJSONLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("decode %q: %v", line, err)
	}
	return rec
}

func TestInitLevels(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		enabled map[slog.Level]bool
	}{
		{
			name:  "debug",
			level: "debug",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: true, slog.LevelInfo: true,
				slog.LevelWarn: true, slog.LevelError: true,
			},
		},
		{
			name:  "info",
			level: "info",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: false, slog.LevelInfo: true,
				slog.LevelWarn: true, slog.LevelError: true,
			},
		},
		{
			name:  "warn",
			level: "warn",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: false, slog.LevelInfo: false,
				slog.LevelWarn: true, slog.LevelError: true,
			},
		},
		{
			name:  "error",
			level: "error",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: false, slog.LevelInfo: false,
				slog.LevelWarn: false, slog.LevelError: true,
			},
		},
		{
			name:  "mixed case is normalised",
			level: "DeBuG",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: true, slog.LevelInfo: true,
			},
		},
		{
			// Anything unrecognised falls back to info.
			name:  "unknown falls back to info",
			level: "verbose",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: false, slog.LevelInfo: true,
			},
		},
		{
			name:  "empty falls back to info",
			level: "",
			enabled: map[slog.Level]bool{
				slog.LevelDebug: false, slog.LevelInfo: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreDefaultLogger(t)
			Init(Config{Level: tt.level, Format: "json"})

			for level, want := range tt.enabled {
				got := slog.Default().Enabled(context.Background(), level)
				if got != want {
					t.Errorf("Enabled(%v) = %v, want %v", level, got, want)
				}
			}
		})
	}
}

func TestInitFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		isJSON bool
	}{
		{name: "json", format: "json", isJSON: true},
		{name: "json mixed case", format: "JSON", isJSON: true},
		{name: "text", format: "text", isJSON: false},
		// Only "json" selects the JSON handler; everything else is text.
		{name: "unknown falls back to text", format: "yaml", isJSON: false},
		{name: "empty falls back to text", format: "", isJSON: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := initAndLog(t, Config{Level: "info", Format: tt.format}, nil, func() {
				Info("hello", slog.String("key", "value"))
			})

			if tt.isJSON {
				rec := decodeJSONLine(t, strings.TrimSpace(out))
				if rec["msg"] != "hello" {
					t.Errorf("msg = %v, want %q", rec["msg"], "hello")
				}
				if rec["key"] != "value" {
					t.Errorf("key = %v, want %q", rec["key"], "value")
				}
				return
			}

			if json.Valid([]byte(strings.TrimSpace(out))) {
				t.Errorf("output = %q, want text format", out)
			}
			if !strings.Contains(out, `msg=hello`) || !strings.Contains(out, `key=value`) {
				t.Errorf("output = %q, want text attributes", out)
			}
		})
	}
}

func TestInitAppliesExtractors(t *testing.T) {
	type ctxKey struct{}

	extractor := func(ctx context.Context) []slog.Attr {
		v, ok := ctx.Value(ctxKey{}).(string)
		if !ok {
			return nil
		}
		return []slog.Attr{slog.String("request_id", v)}
	}
	second := func(context.Context) []slog.Attr {
		return []slog.Attr{slog.String("service", "api")}
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "abc-123")
	out := initAndLog(t, Config{Level: "info", Format: "json"}, []ContextAttrFunc{extractor, second}, func() {
		InfoContext(ctx, "with context")
		Info("without context")
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), out)
	}

	withCtx := decodeJSONLine(t, lines[0])
	if withCtx["request_id"] != "abc-123" {
		t.Errorf("request_id = %v, want %q", withCtx["request_id"], "abc-123")
	}
	if withCtx["service"] != "api" {
		t.Errorf("service = %v, want %q", withCtx["service"], "api")
	}

	// The background context carries no value, so the extractor contributes nothing.
	withoutCtx := decodeJSONLine(t, lines[1])
	if _, ok := withoutCtx["request_id"]; ok {
		t.Errorf("request_id = %v, want it absent", withoutCtx["request_id"])
	}
	if withoutCtx["service"] != "api" {
		t.Errorf("service = %v, want %q", withoutCtx["service"], "api")
	}
}

func TestHumanizeDuration(t *testing.T) {
	tests := []struct {
		name string
		attr slog.Attr
		want slog.Value
	}{
		{
			name: "rounds to microsecond",
			attr: slog.Duration("took", 1523456*time.Nanosecond),
			want: slog.StringValue("1.523ms"),
		},
		{
			name: "sub-millisecond stays visible",
			attr: slog.Duration("took", 250*time.Microsecond),
			want: slog.StringValue("250µs"),
		},
		{
			name: "zero",
			attr: slog.Duration("took", 0),
			want: slog.StringValue("0s"),
		},
		{
			name: "negative",
			attr: slog.Duration("took", -2*time.Second),
			want: slog.StringValue("-2s"),
		},
		{
			name: "non-duration is untouched",
			attr: slog.Int("count", 3),
			want: slog.IntValue(3),
		},
		{
			name: "string is untouched",
			attr: slog.String("msg", "1s"),
			want: slog.StringValue("1s"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanizeDuration(nil, tt.attr)

			if got.Key != tt.attr.Key {
				t.Errorf("Key = %q, want %q", got.Key, tt.attr.Key)
			}
			if !got.Value.Equal(tt.want) {
				t.Errorf("Value = %v (%v), want %v (%v)",
					got.Value, got.Value.Kind(), tt.want, tt.want.Kind())
			}
		})
	}
}

func TestDurationIsHumanizedInOutput(t *testing.T) {
	out := initAndLog(t, Config{Level: "info", Format: "json"}, nil, func() {
		Info("request done", slog.Duration("latency", 1523456*time.Nanosecond))
	})

	rec := decodeJSONLine(t, strings.TrimSpace(out))
	if rec["latency"] != "1.523ms" {
		t.Errorf("latency = %v, want %q", rec["latency"], "1.523ms")
	}
}

func TestLevelFunctions(t *testing.T) {
	tests := []struct {
		name  string
		log   func()
		level string
	}{
		{name: "Debug", level: "DEBUG", log: func() { Debug("m", slog.String("k", "v")) }},
		{name: "Info", level: "INFO", log: func() { Info("m", slog.String("k", "v")) }},
		{name: "Warn", level: "WARN", log: func() { Warn("m", slog.String("k", "v")) }},
		{name: "Error", level: "ERROR", log: func() { Error("m", slog.String("k", "v")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := initAndLog(t, Config{Level: "debug", Format: "json"}, nil, tt.log)

			rec := decodeJSONLine(t, strings.TrimSpace(out))
			if rec["level"] != tt.level {
				t.Errorf("level = %v, want %q", rec["level"], tt.level)
			}
			if rec["msg"] != "m" {
				t.Errorf("msg = %v, want %q", rec["msg"], "m")
			}
			if rec["k"] != "v" {
				t.Errorf("k = %v, want %q", rec["k"], "v")
			}
		})
	}
}

func TestContextLevelFunctions(t *testing.T) {
	type ctxKey struct{}
	extractor := func(ctx context.Context) []slog.Attr {
		v, ok := ctx.Value(ctxKey{}).(string)
		if !ok {
			return nil
		}
		return []slog.Attr{slog.String("trace", v)}
	}
	ctx := context.WithValue(context.Background(), ctxKey{}, "t-1")

	tests := []struct {
		name  string
		log   func(context.Context)
		level string
	}{
		{name: "DebugContext", level: "DEBUG", log: func(c context.Context) { DebugContext(c, "m") }},
		{name: "InfoContext", level: "INFO", log: func(c context.Context) { InfoContext(c, "m") }},
		{name: "WarnContext", level: "WARN", log: func(c context.Context) { WarnContext(c, "m") }},
		{name: "ErrorContext", level: "ERROR", log: func(c context.Context) { ErrorContext(c, "m") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Level: "debug", Format: "json"}
			out := initAndLog(t, cfg, []ContextAttrFunc{extractor}, func() { tt.log(ctx) })

			rec := decodeJSONLine(t, strings.TrimSpace(out))
			if rec["level"] != tt.level {
				t.Errorf("level = %v, want %q", rec["level"], tt.level)
			}
			if rec["trace"] != "t-1" {
				t.Errorf("trace = %v, want %q", rec["trace"], "t-1")
			}
		})
	}
}

func TestLevelFunctionsRespectThreshold(t *testing.T) {
	out := initAndLog(t, Config{Level: "error", Format: "json"}, nil, func() {
		Debug("debug")
		Info("info")
		Warn("warn")
		Error("error")
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want only the error: %q", len(lines), out)
	}
	if rec := decodeJSONLine(t, lines[0]); rec["msg"] != "error" {
		t.Errorf("msg = %v, want %q", rec["msg"], "error")
	}
}

func TestPanic(t *testing.T) {
	tests := []struct {
		name string
		log  func()
	}{
		{name: "Panic", log: func() { Panic("boom", slog.String("k", "v")) }},
		{name: "PanicContext", log: func() { PanicContext(context.Background(), "boom", slog.String("k", "v")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recovered any
			out := initAndLog(t, Config{Level: "info", Format: "json"}, nil, func() {
				defer func() { recovered = recover() }()
				tt.log()
			})

			if recovered != "boom" {
				t.Errorf("recover() = %v, want %q", recovered, "boom")
			}
			rec := decodeJSONLine(t, strings.TrimSpace(out))
			if rec["level"] != "ERROR" {
				t.Errorf("level = %v, want %q", rec["level"], "ERROR")
			}
			if rec["msg"] != "boom" {
				t.Errorf("msg = %v, want %q", rec["msg"], "boom")
			}
			if rec["k"] != "v" {
				t.Errorf("k = %v, want %q", rec["k"], "v")
			}
		})
	}
}

// Fatal calls os.Exit, so it is exercised by re-running this test binary as a
// child process and inspecting its exit status and output.
func TestFatal(t *testing.T) {
	const envVar = "LOGGER_TEST_FATAL_FUNC"

	if fn := os.Getenv(envVar); fn != "" {
		Init(Config{Level: "info", Format: "json"})
		if fn == "FatalContext" {
			FatalContext(context.Background(), "fatal msg", slog.String("k", "v"))
		} else {
			Fatal("fatal msg", slog.String("k", "v"))
		}
		return // unreachable unless os.Exit was skipped
	}

	for _, fn := range []string{"Fatal", "FatalContext"} {
		t.Run(fn, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$", "-test.v")
			cmd.Env = append(os.Environ(), envVar+"="+fn)
			out, err := cmd.CombinedOutput()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("child exited with %v, want exit status 1; output: %s", err, out)
			}
			if code := exitErr.ExitCode(); code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(string(out), `"msg":"fatal msg"`) {
				t.Errorf("output = %s, want the fatal message logged as JSON", out)
			}
			if !strings.Contains(string(out), `"level":"ERROR"`) {
				t.Errorf("output = %s, want level ERROR", out)
			}
		})
	}
}

func TestErr(t *testing.T) {
	t.Run("nil returns an empty attr", func(t *testing.T) {
		got := Err(nil)

		if !got.Equal(slog.Attr{}) {
			t.Errorf("Err(nil) = %v, want the zero Attr", got)
		}
	})

	t.Run("non-nil returns a string attr", func(t *testing.T) {
		got := Err(errors.New("boom"))

		if got.Key != "error" {
			t.Errorf("Key = %q, want %q", got.Key, "error")
		}
		if got.Value.Kind() != slog.KindString || got.Value.String() != "boom" {
			t.Errorf("Value = %v, want %q", got.Value, "boom")
		}
	})

	// An empty Attr is dropped by slog, so a nil error adds no field.
	t.Run("empty attr is omitted from output", func(t *testing.T) {
		out := initAndLog(t, Config{Level: "info", Format: "json"}, nil, func() {
			Info("no error", Err(nil))
			Info("has error", Err(errors.New("boom")))
		})

		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2: %q", len(lines), out)
		}
		if _, ok := decodeJSONLine(t, lines[0])["error"]; ok {
			t.Errorf("line = %q, want no error field", lines[0])
		}
		if got := decodeJSONLine(t, lines[1])["error"]; got != "boom" {
			t.Errorf("error = %v, want %q", got, "boom")
		}
	})
}
