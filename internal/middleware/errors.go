package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	e "github.com/aarondever/go-gin-template/internal/apperror"

	"github.com/aarondever/go-gin-template/internal/logger"
	"github.com/gin-gonic/gin"
)

var statusByCode = map[e.Code]int{
	e.CodeInvalidInput: http.StatusBadRequest,
	e.CodeUnauthorized: http.StatusUnauthorized,
	e.CodeForbidden:    http.StatusForbidden,
	e.CodeNotFound:     http.StatusNotFound,
	e.CodeConflict:     http.StatusConflict,
	e.CodeRateLimited:  http.StatusTooManyRequests,
	e.CodeCanceled:     499, // Client Closed Request
	e.CodeTimeout:      http.StatusGatewayTimeout,
	e.CodeInternal:     http.StatusInternalServerError,
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    e.Code            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		appErr := e.From(err)

		status, ok := statusByCode[appErr.Code]
		if !ok {
			status = http.StatusInternalServerError
		}

		// Log failed request
		attrs := []slog.Attr{
			slog.Any("code", appErr.Code),
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("query", c.Request.URL.Path),
			slog.String("error", appErr.Error()),
		}
		if status >= 500 {
			logger.ErrorContext(c.Request.Context(), "request failed", attrs...)
		} else {
			logger.WarnContext(c.Request.Context(), "request failed", attrs...)
		}

		// Client disconnected — nothing to write.
		if errors.Is(err, context.Canceled) {
			c.Abort()
			return
		}

		// A handler already committed a response; don't write twice.
		if c.Writer.Written() {
			return
		}

		body := errorResponse{Error: errorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		}}

		c.AbortWithStatusJSON(status, body)
	}
}
