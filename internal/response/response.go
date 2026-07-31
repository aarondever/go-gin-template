package response

import (
	"github.com/gin-gonic/gin"
)

type response struct {
	Data any `json:"data,omitempty"`
}

func JSON(c *gin.Context, code int, data any) {
	c.JSON(code, response{
		Data: data,
	})
}
