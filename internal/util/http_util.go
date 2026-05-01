package util

import (
	"errors"
	"net/http"

	e "github.com/aarondever/go-gin-template/errors"
	"github.com/aarondever/go-gin-template/pkg/response"
	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error, fallbackMsg string) {
	var valErr *e.ValidationError
	switch {
	case errors.As(err, &valErr):
		response.Error(c, http.StatusBadRequest, "validation failed", valErr.Err)

	case errors.Is(err, e.ErrNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)

	case errors.Is(err, e.ErrConflict):
		response.Error(c, http.StatusConflict, "conflict", err)

	default:
		response.Error(c, http.StatusInternalServerError, fallbackMsg, err)
	}
}
