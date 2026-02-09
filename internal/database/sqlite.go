package database

import (
	"embed"
	"errors"
	"fmt"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type SQLite struct {
	db *sqlx.DB
}

func NewSQLite(dbConf config.ConfDB) (*SQLite, error) {
	var dsn string

	// This allows to easily override dsn for tests
	if dbConf.Dsn != "" {
		dsn = dbConf.Dsn
	} else {
		dsn = fmt.Sprintf("file:%s?cache=shared&mode=rwc&_loc=auto&parseTime=true", dbConf.File)
	}

	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Connection pool settings (SQLite handles 1 writer at a time)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Reuse connections indefinitely (SQLite is file-based)

	// SQLite-specific pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA cache_size = -64000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %q: %w", pragma, err)
		}
	}

	return &SQLite{db: db}, nil
}

func (s *SQLite) DB() *sqlx.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) Migrate() error {
	// Create sqlite3 driver instance
	driver, err := sqlite3.WithInstance(s.db.DB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("create driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	// Run migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
