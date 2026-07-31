package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// envKeys is every variable Load reads. Tests clear all of them so a developer's
// shell environment cannot leak into the results.
var envKeys = []string{
	"SERVER_PORT", "SERVER_MODE", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT",
	"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
	"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
	"LOG_LEVEL", "LOG_FORMAT",
}

// isolate moves the test into an empty directory (so no .env is picked up) and
// clears every config variable. t.Setenv records the original value, so the
// following Unsetenv is undone when the test finishes.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	for _, key := range envKeys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	return dir
}

// setRequired fills in the variables without defaults.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "appdb")
}

func TestLoadDefaults(t *testing.T) {
	isolate(t)
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Server: ServerConfig{
			Port:         8080,
			Mode:         "release",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		DB: DBConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "app",
			Password:        "secret",
			DBName:          "appdb",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Log: LogConfig{Level: "info", Format: "json"},
	}
	if *cfg != want {
		t.Errorf("Load() = %+v, want %+v", *cfg, want)
	}
}

func TestLoadFromEnv(t *testing.T) {
	isolate(t)
	setRequired(t)

	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_MODE", "debug")
	t.Setenv("SERVER_READ_TIMEOUT", "5s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "1m")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_MAX_OPEN_CONNS", "100")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("DB_CONN_MAX_LIFETIME", "90s")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Server: ServerConfig{
			Port:         9090,
			Mode:         "debug",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: time.Minute,
		},
		DB: DBConfig{
			Host:            "db.internal",
			Port:            6543,
			User:            "app",
			Password:        "secret",
			DBName:          "appdb",
			SSLMode:         "require",
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 90 * time.Second,
		},
		Log: LogConfig{Level: "debug", Format: "text"},
	}
	if *cfg != want {
		t.Errorf("Load() = %+v, want %+v", *cfg, want)
	}
}

// The password carries the "unset" option, which clears it from the process
// environment once it has been read into the struct.
func TestLoadUnsetsPassword(t *testing.T) {
	isolate(t)
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DB.Password != "secret" {
		t.Errorf("cfg.DB.Password = %q, want %q", cfg.DB.Password, "secret")
	}
	if got, ok := os.LookupEnv("DB_PASSWORD"); ok {
		t.Errorf("DB_PASSWORD still set to %q after Load()", got)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	required := []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	for _, missing := range required {
		t.Run(missing, func(t *testing.T) {
			isolate(t)
			setRequired(t)
			if err := os.Unsetenv(missing); err != nil {
				t.Fatalf("unset %s: %v", missing, err)
			}

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load() = %+v, want error when %s is unset", cfg, missing)
			}
			if cfg != nil {
				t.Errorf("Load() = %+v, want nil config alongside error", cfg)
			}
		})
	}
}

func TestLoadInvalidValue(t *testing.T) {
	isolate(t)
	setRequired(t)
	t.Setenv("SERVER_PORT", "not-a-number")

	if cfg, err := Load(); err == nil {
		t.Fatalf("Load() = %+v, want error for non-numeric SERVER_PORT", cfg)
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	dir := isolate(t)

	dotenv := "DB_HOST=file-host\nDB_USER=file-user\nDB_PASSWORD=file-pass\nDB_NAME=file-db\nSERVER_PORT=7070\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DB.Host != "file-host" {
		t.Errorf("cfg.DB.Host = %q, want %q", cfg.DB.Host, "file-host")
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("cfg.Server.Port = %d, want %d", cfg.Server.Port, 7070)
	}
}

// Real environment variables win over .env entries.
func TestLoadEnvOverridesDotEnvFile(t *testing.T) {
	dir := isolate(t)
	setRequired(t)

	dotenv := "DB_HOST=file-host\nSERVER_PORT=7070\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("cfg.DB.Host = %q, want the exported value %q", cfg.DB.Host, "localhost")
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("cfg.Server.Port = %d, want the .env value %d", cfg.Server.Port, 7070)
	}
}

// A missing .env is not an error; a malformed one is.
func TestLoadMalformedDotEnvFile(t *testing.T) {
	dir := isolate(t)
	setRequired(t)

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("this is not = valid = \x00\nDB_HOST\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if cfg, err := Load(); err == nil {
		t.Fatalf("Load() = %+v, want error for malformed .env", cfg)
	}
}
