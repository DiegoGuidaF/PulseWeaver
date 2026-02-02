package database

import (
	"context"
	"fmt"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	db *sqlx.DB
}

func NewSQLite(dbConf *config.ConfDB) (*SQLite, error) {
	// Open with WAL mode for better concurrency
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL", dbConf.File)

	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify connection
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Connection pool settings (SQLite handles 1 writer at a time)
	db.SetMaxOpenConns(1) // SQLite limitation
	db.SetMaxIdleConns(1)

	return &SQLite{db: db}, nil
}

func (s *SQLite) DB() *sqlx.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}
