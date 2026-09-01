package response

import (
	"github.com/aarondever/go-gin-template/internal/util"
	"github.com/gin-gonic/gin"
)

type response struct {
	Data    any    `json:"data"`
	TraceID string `json:"trace_id,omitempty"`
}

func JSON(c *gin.Context, code int, data any) {
	c.JSON(code, response{
		Data:    data,
		TraceID: util.TraceID(c.Request.Context()),
	})
}
