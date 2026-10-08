package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &Store{db: db}
	if err := store.Ping(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS services (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE,
            description TEXT NOT NULL,
            owner_team TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'Operational',
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS incidents (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            description TEXT NOT NULL,
            severity TEXT NOT NULL,
            status TEXT NOT NULL,
            reporter TEXT NOT NULL,
            assigned_owner TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            resolved_at TEXT
        )`,
		`CREATE TABLE IF NOT EXISTS incident_services (
            incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
            service_id INTEGER NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
            PRIMARY KEY (incident_id, service_id)
        )`,
		`CREATE TABLE IF NOT EXISTS timeline_updates (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
            message TEXT NOT NULL,
            status TEXT NOT NULL,
            created_at TEXT NOT NULL
        )`,
		`CREATE INDEX IF NOT EXISTS idx_services_status ON services(status)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_created ON incidents(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_timeline_incident ON timeline_updates(incident_id, created_at DESC)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply schema migration: %w", err)
		}
	}
	return nil
}

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record conflicts with existing data")
