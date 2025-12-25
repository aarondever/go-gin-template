package response

import (
	"github.com/aarondever/go-gin-template/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		logger.Error("Request error", "error", err, "path", c.Request.URL.Path)
	}

	c.JSON(code, Response{
		Success: false,
		Message: message,
		Error:   errMsg,
	})
}
