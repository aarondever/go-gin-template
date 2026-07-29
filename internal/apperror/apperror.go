package apperror

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
)

type Code string

const (
	CodeInvalidInput Code = "INVALID_INPUT"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeRateLimited  Code = "RATE_LIMITED"
	CodeCanceled     Code = "CANCELED"
	CodeTimeout      Code = "TIMEOUT"
	CodeInternal     Code = "INTERNAL"
)

type AppError struct {
	Code    Code
	Message string
	Details map[string]string // optional, e.g. field-level validation
	Err     error             // internal cause, never serialized
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code Code, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Wrap(err error, code Code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}

func (e *AppError) WithDetails(d map[string]string) *AppError {
	e.Details = d
	return e
}

// From normalizes any error into an *AppError. Returns nil for nil.
func From(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := make(map[string]string, len(ve))
		for _, fe := range ve {
			details[fe.Field()] = fe.Tag()
		}
		return Wrap(err, CodeInvalidInput, "validation failed").
			WithDetails(details)
	}

	switch {
	case errors.Is(err, context.Canceled):
		return Wrap(err, CodeCanceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return Wrap(err, CodeTimeout, "request timed out")
	}

	return Wrap(err, CodeInternal, "internal server error")
}
