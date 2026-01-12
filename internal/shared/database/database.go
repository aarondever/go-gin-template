package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aarondever/go-gin-template/internal/shared/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	*gorm.DB
	Redis *redis.Client
}

func New(dbCfg config.DatabaseConfig, redisCfg config.RedisConfig, s *slog.Logger) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.Username,
		dbCfg.Password,
		dbCfg.Database,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.NewSlogLogger(s, logger.Config{
			LogLevel: logger.Info,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(dbCfg.ConnMaxLifetime) * time.Second)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Database{
		DB:    db,
		Redis: rdb,
	}, nil
}

func (d *Database) Close() error {
	var errs []error

	// Close PostgreSQL connection
	sqlDB, err := d.DB.DB()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get underlying sql.DB: %w", err))
	} else {
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	// Close Redis connection
	if d.Redis != nil {
		if err := d.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}
