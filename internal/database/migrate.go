package database

import (
	"database/sql"
	"fmt"

	"github.com/aarondever/go-gin-template/migrations"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Embed all SQL files at compile time — no external files needed at runtime
//

func RunMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return runGooseMigrations(sqlDB)
}

func runGooseMigrations(sqlDB *sql.DB) error {
	goose.SetBaseFS(migrations.MigrationsFS)
	defer goose.SetBaseFS(nil) // reset after use

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// files are embedded at the FS root (embed.go uses //go:embed *.sql)
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("goose up failed: %w", err)
	}

	return nil
}
