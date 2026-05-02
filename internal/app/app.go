package app

import (
	"github.com/aarondever/go-gin-template/config"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/internal/service"
	"github.com/aarondever/go-gin-template/pkg/logger"
)

type App struct {
	UserSvc service.UserService
}

func New(cfg *config.Config) *App {
	// Initialize logger
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	// Initialize database
	db, err := database.New(cfg, logger.Logger())
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	if cfg.Database.RunMigrations {
		// Run database migrations
		if err := database.RunMigrations(db.DB()); err != nil {
			logger.Fatal("Failed to run database migrations", "error", err)
		}
	}

	userRepo := repository.NewUserRepository(db.DB())

	return &App{
		UserSvc: service.NewUserService(userRepo),
	}
}
