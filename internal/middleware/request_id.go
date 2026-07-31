package middleware

import (
	"github.com/aarondever/go-gin-template/internal/util"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get id from request
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = util.NewID()
		}

		c.Header(RequestIDHeader, reqID)
		c.Request = c.Request.WithContext(
			util.InjectRequestID(c.Request.Context(), reqID),
		)

		c.Next()
	}
}
