package middleware

import (
	"github.com/aarondever/go-gin-template/pkg/util"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get id from request
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Header(RequestIDHeader, reqID)
		c.Request = c.Request.WithContext(
			util.InjectRequestID(c.Request.Context(), reqID),
		)

		c.Next()
	}
}
