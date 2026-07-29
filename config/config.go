package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
	Log    LogConfig
}

type ServerConfig struct {
	Port         int           `env:"SERVER_PORT" envDefault:"8080"`
	Mode         string        `env:"SERVER_MODE" envDefault:"release"` // "debug", "test", "release"
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
}

type DBConfig struct {
	Host            string        `env:"DB_HOST,required"`
	Port            int           `env:"DB_PORT" envDefault:"5432"`
	User            string        `env:"DB_USER,required"`
	Password        string        `env:"DB_PASSWORD,required,unset"` // unset clears it from env after read
	DBName          string        `env:"DB_NAME,required"`
	SSLMode         string        `env:"DB_SSLMODE" envDefault:"disable"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"json"` // "json" or "text"
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load .env: %w", err)
		}
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	return &cfg, nil
}
