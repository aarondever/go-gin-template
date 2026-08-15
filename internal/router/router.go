package router

import (
	"fmt"
	"net/http"

	"github.com/aarondever/go-gin-template/config"
	"github.com/aarondever/go-gin-template/internal/handler"
	"github.com/aarondever/go-gin-template/internal/middleware"
	"github.com/aarondever/go-gin-template/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const healthPath = "/health"

func SetupRouter(
	cfg *config.Config,
	h *handler.Handler,
) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	// So bind errors and manual ValidateStruct errors name fields identically.
	validation.UseFieldNames(binding.Validator.Engine())

	r := gin.New()

	r.Use(
		otelgin.Middleware(cfg.OTEL.ServiceName, otelgin.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != healthPath
		})),
		middleware.Logger(healthPath),
		middleware.ErrorHandler(),
		// Skips gin's bare 500, so a panic unwinds into ErrorHandler and gets the
		// same envelope as every other failure.
		gin.CustomRecovery(func(c *gin.Context, err any) {
			c.Error(fmt.Errorf("panic: %v", err))
		}),
	)

	r.GET(healthPath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", h.Create)
			users.GET("/:userID", h.GetByID)
			users.GET("", h.GetList)
			users.PUT("/:userID", h.Update)
			users.DELETE("/:userID", h.Delete)
		}
	}

	return r
}
