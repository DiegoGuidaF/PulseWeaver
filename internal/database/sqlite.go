package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"
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

		params := url.Values{}
		params.Add("_time_format", "sqlite")
		params.Add("_texttotime", "1")
		params.Add("_timezone", "UTC")

		// SQLite Pragmas
		params.Add("_pragma", "foreign_keys(1)")
		params.Add("_pragma", "journal_mode(WAL)")
		params.Add("_pragma", "synchronous(NORMAL)")
		params.Add("_pragma", "busy_timeout(5000)")
		params.Add("_pragma", "cache_size(-16000)")

		dsn = "file:" + dbPath + "?" + params.Encode()
	}

	pool, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Increase connection pool to allow concurrent readers.
	// We set MaxOpenConns to allow concurrency, but cap it so we don't overwhelm the OS.
	// MaxIdleConns keeps connections warm, reducing overhead.
	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(0) // Reuse connections indefinitely

	if err := pool.PingContext(context.Background()); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
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
