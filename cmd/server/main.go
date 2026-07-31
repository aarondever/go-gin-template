package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aarondever/go-gin-template/config"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/handler"
	"github.com/aarondever/go-gin-template/internal/logger"
	"github.com/aarondever/go-gin-template/internal/middleware"
	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/internal/service"
	"github.com/aarondever/go-gin-template/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	}, logger.WithRequestID)

	// Initialize database
	db, err := database.New(database.Config{
		Host:            cfg.DB.Host,
		Port:            cfg.DB.Port,
		User:            cfg.DB.User,
		Password:        cfg.DB.Password,
		DBName:          cfg.DB.DBName,
		SSLMode:         cfg.DB.SSLMode,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
	})
	if err != nil {
		logger.Fatal("failed to connect to database", logger.Err(err))
	}
	defer db.Close()

	// Initialize repository
	repo := repository.New(db.DB())

	// Initialize service
	service := service.New(repo)

	// Setup router
	gin.SetMode(cfg.Server.Mode)

	// Make gin's binding validator report fields by their json/form name.
	validation.UseFieldNames(binding.Validator.Engine())

	r := gin.New()

	// Global middleware
	r.Use(
		middleware.CORS(),
		middleware.RequestID(),
		middleware.Logger("/health"),
		middleware.ErrorHandler(),
		gin.Recovery(),
	)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")

	// Register routes
	handler.New(service).RegisterRoutes(v1)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting server", slog.Int("port", cfg.Server.Port), slog.String("mode", cfg.Server.Mode))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", logger.Err(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", logger.Err(err))
	}

	logger.Info("server exited")
}
