package database

import (
	"embed"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type SQLite struct {
	pool *sqlx.DB
	db   *DB
	tx   *Transactor
}

const dbFileName = "data.db"

func NewSQLite(dbConf config.ConfDB) (*SQLite, error) {
	var dsn string

	if dbConf.Dsn != "" {
		dsn = dbConf.Dsn
	} else {
		dbPath := filepath.Join(dbConf.DataDir, dbFileName)
		// _time_format=sqlite: Writes time.Time as YYYY-MM-DD HH:MM:SS[+-]HH:MM (SQLite format 4)
		// _texttotime=1: Makes the driver report time.Time for columns declared as DATE, DATETIME, TIME, or TIMESTAMP
		// _timezone=UTC: Interprets timezone-less strings as UTC on reads; converts to UTC before writing (v1.48+)
		dsn = "file:" + dbPath + "?_time_format=sqlite&_texttotime=1&_timezone=UTC"
	}

	pool, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Connection pool settings (SQLite handles 1 writer at a time)
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	pool.SetConnMaxLifetime(0) // Reuse connections indefinitely (SQLite is file-based)

	// SQLite-specific pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA cache_size = -64000",
	}

	//TODO: These pragmas are only set on this first connection. If Max connections is >1, a hook should be done instead
	for _, pragma := range pragmas {
		if _, err := pool.Exec(pragma); err != nil {
			_ = pool.Close()
			return nil, fmt.Errorf("exec %q: %w", pragma, err)
		}
	}

	return &SQLite{
		pool: pool,
		db:   newDB(pool),
		tx:   NewTransactor(pool),
	}, nil
}

func (s *SQLite) DB() *DB {
	return s.db
}

func (s *SQLite) Transactor() *Transactor { return s.tx }

func (s *SQLite) Close() error {
	return s.pool.Close()
}

func (s *SQLite) Migrate() error {
	// Create sqlite driver instance
	driver, err := sqlite.WithInstance(s.pool.DB, &sqlite.Config{NoTxWrap: true})
	if err != nil {
		return fmt.Errorf("create driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	// Run migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
