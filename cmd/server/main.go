package main

import (
	"context"
	"errors"
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
	if err := run(); err != nil {
		log.Fatalln(err)
	}
	logger.Info("server exited")
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger.Init(cfg.Log, logger.WithTrace())

	// Initialize telemetry
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTEL)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = errors.Join(err, otelShutdown(shutdownCtx))
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
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()

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
		return fmt.Errorf("failed to start server: %w", err)
	case <-ctx.Done():
		stop()
	}

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
