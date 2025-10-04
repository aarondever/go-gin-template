//go:build wireinject
// +build wireinject

package internal

import (
	"github.com/aarondever/go-gin-template/internal/config"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/handlers"
	"github.com/aarondever/go-gin-template/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

type App struct {
	DB     *database.Database
	Router *gin.Engine
}

func NewApp(
	db *database.Database,
	userHandler *handlers.UserHandler,
	// Add all handlers as parameters
) *App {
	router := gin.Default()

	// Setup all routes
	userHandler.SetupRouters(router)

	return &App{
		DB:     db,
		Router: router,
	}
}

// InitializeApp uses Wire to initialize all dependencies
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(
		database.NewDatabase,
		services.ProviderSet,
		handlers.ProviderSet,
		NewApp,
	)

	return nil, nil
}
