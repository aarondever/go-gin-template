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
	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/internal/router"
	"github.com/aarondever/go-gin-template/internal/service"
	"github.com/aarondever/go-gin-template/internal/telemetry"
)

func main() {
	// Handle SIGINT (CTRL+C) gracefully.
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.Log, logger.WithTrace())

	// Initialize telemetry
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTEL)
	if err != nil {
		logger.Fatal("failed to initialize telemetry", logger.Err(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelShutdown(ctx); err != nil {
			logger.Error("failed to shutdown telemetry", logger.Err(err))
		}
	}()

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
	svc := service.New(repo)

	// Initialize handler
	h := handler.New(svc)

	// Setup router
	r := router.SetupRouter(cfg, h)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	srvErr := make(chan error, 1)
	go func() {
		logger.Info("starting server", slog.Int("port", cfg.Server.Port), slog.String("mode", cfg.Server.Mode))
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		logger.Error("server failed to start", logger.Err(err))
	case <-signalCtx.Done():
		logger.Info("shutting down server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", logger.Err(err))
	}

	logger.Info("server exited")
}
