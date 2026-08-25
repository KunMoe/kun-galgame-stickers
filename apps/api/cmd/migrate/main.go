package main

import (
	"database/sql"
	"flag"
	"log/slog"
	"os"

	"kun-galgame-sticker-api/internal/infrastructure/database"
	"kun-galgame-sticker-api/pkg/config"
	"kun-galgame-sticker-api/pkg/logger"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	direction := flag.String("dir", "up", "up or down")
	step := flag.Int("step", 0, "number of steps (0 = all)")
	path := flag.String("path", defaultMigrationsPath(), "path to migration files")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db, err := sql.Open("postgres", database.SanitizeDSN(cfg.Database.URL))
	if err != nil {
		slog.Error("open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("migrate driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+*path, "postgres", driver)
	if err != nil {
		slog.Error("migrate", "error", err)
		os.Exit(1)
	}

	var runErr error
	switch *direction {
	case "up":
		if *step > 0 {
			runErr = m.Steps(*step)
		} else {
			runErr = m.Up()
		}
	case "down":
		if *step > 0 {
			runErr = m.Steps(-*step)
		} else {
			runErr = m.Down()
		}
	default:
		slog.Error("unknown direction", "dir", *direction)
		os.Exit(1)
	}

	if runErr != nil && runErr != migrate.ErrNoChange {
		slog.Error("migrate run", "error", runErr)
		os.Exit(1)
	}
	slog.Info("migrate ok", "dir", *direction)
}

// The image copies SQL to /migrations and pins WORKDIR /. Local `go run`
// still uses ./migrations. Prefer the absolute path when it exists so a
// compose file does not have to hard-code -path (forum's pattern: default
// `up` with no command, so `run --rm migrate -dir down` still works).
func defaultMigrationsPath() string {
	if st, err := os.Stat("/migrations"); err == nil && st.IsDir() {
		return "/migrations"
	}
	return "migrations"
}
