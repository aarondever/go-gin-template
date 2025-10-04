package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aarondever/go-gin-template/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Database struct {
	Queries Querier
	Pool    *pgxpool.Pool
	Redis   *redis.Client
}

// NewDatabase establishes database connection pool and validates connectivity
func NewDatabase(cfg *config.Config) (*Database, error) {
	slog.Info("Initializing database connection")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Configure connection pool settings
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		slog.Error("Database configuration parsing failed", "error", err)
		return nil, err
	}

	// Configure pool settings
	poolConfig.MaxConns = 20 // Maximum number of connections
	poolConfig.MinConns = 5  // Minimum number of connections to maintain
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("Database connection pool creation failed",
			"error", err,
			"max_conns", poolConfig.MaxConns,
			"min_conns", poolConfig.MinConns,
		)
		return nil, err
	}

	// Test connection
	if err = pool.Ping(ctx); err != nil {
		slog.Error("Database ping failed", "error", err)
		pool.Close()
		return nil, err
	}

	database := &Database{
		Queries: New(pool),
		Pool:    pool,
	}

	slog.Info("Database connection pool established successfully",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
		"max_conn_lifetime", poolConfig.MaxConnLifetime,
		"max_conn_idle_time", poolConfig.MaxConnIdleTime,
	)

	// Connect to Redis
	if err = database.connectToRedis(cfg); err != nil {
		pool.Close()
		return nil, err
	}

	return database, nil
}

func (database *Database) connectToRedis(config *config.Config) error {
	var redisURL string
	// Format: redis://[:password@]host:port/db
	if config.Redis.Password != "" {
		redisURL = fmt.Sprintf("redis://:%s@%s:%d/%d",
			config.Redis.Password,
			config.Redis.Host,
			config.Redis.Port,
			config.Redis.DB,
		)
	} else {
		redisURL = fmt.Sprintf("redis://%s:%d/%d",
			config.Redis.Host,
			config.Redis.Port,
			config.Redis.DB,
		)
	}

	// Initialize Redis client
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Error("Failed to parse Redis URL", "error", err)
		return err
	}

	database.Redis = redis.NewClient(opts)

	// Verify Redis connectivity
	ctxPing, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err = database.Redis.Ping(ctxPing).Err(); err != nil {
		slog.Error("Failed to ping Redis", "error", err, "address", opts.Addr, "db", opts.DB)
		return err
	}

	slog.Info("Connected to Redis successfully", "address", opts.Addr, "db", opts.DB)

	return nil
}

// ExecuteTransaction runs multiple database operations within a single transaction
func (queries *Queries) ExecuteTransaction(
	ctx context.Context,
	pool *pgxpool.Pool,
	operations ...func(*Queries) error,
) error {
	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		slog.Error("Failed to begin database transaction", "error", err)
		return err
	}

	// Ensure transaction rollback
	defer tx.Rollback(ctx)

	qtx := queries.WithTx(tx)

	// Execute all operations within the transaction scope
	for _, op := range operations {
		if op == nil {
			slog.Warn("Transaction operation is nil")
			continue
		}

		if err = op(qtx); err != nil {
			slog.Error("Transaction operation failed", "error", err)
			return err
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit database transaction", "error", err)
		return err
	}

	return nil
}
