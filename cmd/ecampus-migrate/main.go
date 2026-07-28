package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const defaultMigrationPath = "db/migrations"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("ecampus-migrate", flag.ContinueOnError)
	configDir := flags.String("config-dir", "configs/ecampus", "configuration directory")
	migrationPath := flags.String("path", defaultMigrationPath, "migration directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: ecampus-migrate [-config-dir path] [-path path] <up|down|version>")
	}

	cfg := config.Load(*configDir)
	dsn, err := migrationDSN(cfg.MySQL.DSN)
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}

	driver, err := migratemysql.WithInstance(db, &migratemysql.Config{})
	if err != nil {
		return fmt.Errorf("create mysql migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+*migrationPath, "mysql", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	switch flags.Arg(0) {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "version":
		version, dirty, versionErr := m.Version()
		if versionErr != nil {
			return fmt.Errorf("read migration version: %w", versionErr)
		}
		fmt.Printf("version=%d dirty=%t\n", version, dirty)
		return nil
	default:
		return fmt.Errorf("unsupported migration action %q", flags.Arg(0))
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %s: %w", flags.Arg(0), err)
	}
	return nil
}

func migrationDSN(raw string) (string, error) {
	cfg, err := mysqldriver.ParseDSN(raw)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.MultiStatements = true
	return cfg.FormatDSN(), nil
}
