package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func New(cfg Config) (*Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		TranslateError: true,
		Logger: gormLogger.NewSlogLogger(slog.Default(), gormLogger.Config{
			LogLevel: gormLogger.Warn,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	if d.db == nil {
		return nil
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// Close Database connection
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database connection: %w", err)
	}

	d.db = nil
	return nil
}

func (d *Database) DB() *gorm.DB {
	return d.db
}
