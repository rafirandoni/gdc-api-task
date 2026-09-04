package main

import (
	"context"
	"fmt"
	"os"

	"github.com/uptrace/bun/migrate"

	"api-task/internal/config"
	"api-task/internal/platform/database"
	"api-task/internal/platform/logger"
	"api-task/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("config: %+v", err)
		os.Exit(1)
	}

	log := logger.NewLogger(cfg)

	db, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer db.Close()

	if len(os.Args) < 2 {
		fmt.Printf("usage expected one of: up, down, status")
		os.Exit(1)
	}

	migrator := migrate.NewMigrator(db, migrations.Migrations)
	ctx := context.Background()

	switch os.Args[1] {
	case "up":
		if err := migrator.Init(ctx); err != nil {
			log.Fatal().Err(err).Msg("init migrations table")
		}
		group, err := migrator.Migrate(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("migrate up")
		}
		log.Info().Int("applied", len(group.Migrations)).Msg("migrations applied")
	case "down":
		group, err := migrator.Rollback(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("migrate down")
		}
		if len(group.Migrations) == 0 {
			log.Info().Msg("no migrations to roll back")
			return
		}
		log.Info().Int("rolled_back", len(group.Migrations)).Msg("migrations rolled back")
	case "status":
		ms, err := migrator.MigrationsWithStatus(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("migration status")
		}
		for _, m := range ms {
			state := "unapplied"
			if m.IsApplied() {
				state = "applied"
			}
			fmt.Printf("%-14d %-40s %s\n", m.ID, m.Name, state)
		}
	default:
		fmt.Printf("unknown subcommand usage, expected one of: up, down, status")
		os.Exit(1)
	}
}
