package middleware

import (
	"log/slog"
	"time"

	"github.com/aarondever/go-gin-template/internal/logger"
	"github.com/gin-gonic/gin"
)

func Logger(skipPaths ...string) gin.HandlerFunc {
	// Create a set of paths to skip logging for
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// Skip logging for the specified paths
		if _, ok := skip[path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		query := c.Request.URL.RawQuery

		c.Next()

		logger.InfoContext(c.Request.Context(), "HTTP Request",
			slog.Int("status", c.Writer.Status()),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", time.Since(start)),
			slog.String("user_agent", c.Request.UserAgent()),
		)
	}
}
