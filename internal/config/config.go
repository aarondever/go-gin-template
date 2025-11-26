package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port         string
	Mode         string // "debug", "release", "test"
	ReadTimeout  int
	WriteTimeout int
}

type DatabaseConfig struct {
	Host            string
	Port            string
	Username        string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type LogConfig struct {
	Level  string
	Format string // "json" or "text"
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getStringEnv("SERVER_PORT", "8080"),
			Mode:         getStringEnv("SERVER_MODE", "release"),
			ReadTimeout:  getIntEnv("SERVER_READ_TIMEOUT", 15),
			WriteTimeout: getIntEnv("SERVER_WRITE_TIMEOUT", 15),
		},
		Database: DatabaseConfig{
			Host:            getStringEnv("DB_HOST", "localhost"),
			Port:            getStringEnv("DB_PORT", "5432"),
			Username:        getStringEnv("DB_USERNAME", "postgres"),
			Password:        getStringEnv("DB_PASSWORD", "postgres"),
			Database:        getStringEnv("DB_DATABASE", "myapp"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getIntEnv("DB_CONN_MAX_LIFETIME", 300),
		},
		Log: LogConfig{
			Level:  getStringEnv("LOG_LEVEL", "info"),
			Format: getStringEnv("LOG_FORMAT", "json"),
		},
	}

	return cfg, nil
}

func getStringEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
