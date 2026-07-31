package apperror

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestError(t *testing.T) {
	cause := errors.New("boom")

	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "without cause",
			err:  New(CodeNotFound, "user not found"),
			want: "user not found",
		},
		{
			name: "with cause",
			err:  Wrap(cause, CodeInternal, "load user"),
			want: "load user: boom",
		},
		{
			name: "empty message with cause",
			err:  Wrap(cause, CodeInternal, ""),
			want: ": boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(CodeUnauthorized, "missing token")

	if err.Code != CodeUnauthorized {
		t.Errorf("Code = %q, want %q", err.Code, CodeUnauthorized)
	}
	if err.Message != "missing token" {
		t.Errorf("Message = %q, want %q", err.Message, "missing token")
	}
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
	if err.Details != nil {
		t.Errorf("Details = %v, want nil", err.Details)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("db down")
	err := Wrap(cause, CodeInternal, "query failed")

	if err.Code != CodeInternal {
		t.Errorf("Code = %q, want %q", err.Code, CodeInternal)
	}
	if err.Message != "query failed" {
		t.Errorf("Message = %q, want %q", err.Message, "query failed")
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("cause")

	if got := Wrap(cause, CodeInternal, "msg").Unwrap(); !errors.Is(got, cause) {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}
	if got := New(CodeInternal, "msg").Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

func TestWithDetails(t *testing.T) {
	err := New(CodeInvalidInput, "bad request")
	details := map[string]string{"email": "required"}

	got := err.WithDetails(details)

	if got != err {
		t.Errorf("WithDetails() returned a different error, want the receiver for chaining")
	}
	if err.Details["email"] != "required" {
		t.Errorf("Details = %v, want %v", err.Details, details)
	}

	// Later calls replace the previous map rather than merging into it.
	err.WithDetails(map[string]string{"name": "min"})
	if _, ok := err.Details["email"]; ok {
		t.Errorf("Details = %v, want the earlier entries replaced", err.Details)
	}
}

func TestFromNil(t *testing.T) {
	if got := From(nil); got != nil {
		t.Errorf("From(nil) = %v, want nil", got)
	}
}

func TestFromAppError(t *testing.T) {
	appErr := New(CodeConflict, "already exists")

	if got := From(appErr); got != appErr {
		t.Errorf("From() = %v, want the same *AppError back", got)
	}
}

func TestFromWrappedAppError(t *testing.T) {
	appErr := New(CodeForbidden, "no access")
	wrapped := fmt.Errorf("service layer: %w", appErr)

	if got := From(wrapped); got != appErr {
		t.Errorf("From() = %v, want the nested *AppError %v", got, appErr)
	}
}

func TestFromValidationErrors(t *testing.T) {
	type payload struct {
		Email string `validate:"required"`
		Age   int    `validate:"gte=18"`
	}

	verr := validator.New().Struct(payload{Age: 10})
	if verr == nil {
		t.Fatal("expected validation to fail")
	}

	got := From(verr)

	if got.Code != CodeInvalidInput {
		t.Errorf("Code = %q, want %q", got.Code, CodeInvalidInput)
	}
	if got.Message != "validation failed" {
		t.Errorf("Message = %q, want %q", got.Message, "validation failed")
	}
	want := map[string]string{"Email": "required", "Age": "gte"}
	if len(got.Details) != len(want) {
		t.Fatalf("Details = %v, want %v", got.Details, want)
	}
	for field, tag := range want {
		if got.Details[field] != tag {
			t.Errorf("Details[%q] = %q, want %q", field, got.Details[field], tag)
		}
	}
	// ValidationErrors is a slice, so it is unusable with errors.Is.
	var unwrapped validator.ValidationErrors
	if !errors.As(got, &unwrapped) {
		t.Errorf("From() dropped the original validation error")
	}
}

func TestFromWrappedValidationErrors(t *testing.T) {
	type payload struct {
		Name string `validate:"required"`
	}

	verr := validator.New().Struct(payload{})
	if verr == nil {
		t.Fatal("expected validation to fail")
	}

	got := From(fmt.Errorf("bind body: %w", verr))

	if got.Code != CodeInvalidInput {
		t.Errorf("Code = %q, want %q", got.Code, CodeInvalidInput)
	}
	if got.Details["Name"] != "required" {
		t.Errorf("Details = %v, want Name=required", got.Details)
	}
}

func TestFromStandardErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    Code
		message string
	}{
		{
			name:    "canceled",
			err:     context.Canceled,
			code:    CodeCanceled,
			message: "request canceled",
		},
		{
			name:    "wrapped canceled",
			err:     fmt.Errorf("fetch: %w", context.Canceled),
			code:    CodeCanceled,
			message: "request canceled",
		},
		{
			name:    "deadline exceeded",
			err:     context.DeadlineExceeded,
			code:    CodeTimeout,
			message: "request timed out",
		},
		{
			name:    "wrapped deadline exceeded",
			err:     fmt.Errorf("fetch: %w", context.DeadlineExceeded),
			code:    CodeTimeout,
			message: "request timed out",
		},
		{
			name:    "unknown",
			err:     errors.New("something broke"),
			code:    CodeInternal,
			message: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := From(tt.err)

			if got.Code != tt.code {
				t.Errorf("Code = %q, want %q", got.Code, tt.code)
			}
			if got.Message != tt.message {
				t.Errorf("Message = %q, want %q", got.Message, tt.message)
			}
			if !errors.Is(got, tt.err) {
				t.Errorf("From() dropped the original error %v", tt.err)
			}
		})
	}
}

// A canceled context reaches From as ctx.Err(), not as the sentinel itself.
func TestFromContextErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := From(ctx.Err()); got.Code != CodeCanceled {
		t.Errorf("Code = %q, want %q", got.Code, CodeCanceled)
	}
}
