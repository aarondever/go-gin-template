package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aarondever/go-gin-template/config"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/migrations"
	"github.com/aarondever/go-gin-template/pkg/logger"

	"github.com/pressly/goose/v3"
)

const (
	dialect          = "postgres"
	migrationsDir    = "migrations"
	embeddedFSPath   = "."
	defaultMigration = "sql"
)

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := goose.SetDialect(dialect); err != nil {
		log.Fatalf("failed to set dialect: %v", err)
	}

	command := args[0]
	cmdArgs := args[1:]

	// Commands that operate on files only — no DB connection required.
	switch command {
	case "create":
		runCreate(cmdArgs)
		return
	case "fix":
		if err := goose.Fix(migrationsDir); err != nil {
			log.Fatalf("goose fix: %v", err)
		}
		return
	}

	runWithDB(command, cmdArgs)
}

func runCreate(args []string) {
	if len(args) < 1 {
		log.Fatal("create: migration name required (usage: create <name> [sql|go])")
	}

	name := args[0]
	migrationType := defaultMigration
	if len(args) >= 2 {
		migrationType = args[1]
	}

	if err := goose.Create(nil, migrationsDir, name, migrationType); err != nil {
		log.Fatalf("goose create: %v", err)
	}
}

func runWithDB(command string, args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.Log.Level, cfg.Log.Format)

	db, err := database.New(cfg, logger.Logger())
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	sqlDB, err := db.DB().DB()
	if err != nil {
		logger.Fatal("Failed to get underlying sql.DB", "error", err)
	}

	goose.SetBaseFS(migrations.MigrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.RunContext(context.Background(), command, sqlDB, embeddedFSPath, args...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: migrate <command> [args]

Commands:
  up                      Migrate the DB to the most recent version available
  up-by-one               Migrate the DB up by 1
  up-to VERSION           Migrate the DB to a specific VERSION
  down                    Roll back the version by 1
  down-to VERSION         Roll back to a specific VERSION
  redo                    Re-run the latest migration
  reset                   Roll back all migrations
  status                  Dump the migration status for the current DB
  version                 Print the current version of the database
  create NAME [sql|go]    Create a new migration file in ./migrations
  fix                     Apply sequential ordering to migrations
`)
}
