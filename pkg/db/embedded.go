package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

type DBManager struct {
	postgres *embeddedpostgres.EmbeddedPostgres
	db       *sql.DB
	connStr  string
}

func NewDBManager() *DBManager {
	return &DBManager{}
}

func (m *DBManager) Init() error {
	customURL := os.Getenv("ZURI_DB_URL")
	if customURL != "" {
		db, err := sql.Open("postgres", customURL)
		if err != nil {
			return fmt.Errorf("failed to connect to custom database URL: %w", err)
		}
		if err := db.Ping(); err != nil {
			db.Close()
			return fmt.Errorf("failed to ping custom database: %w", err)
		}
		m.db = db
		m.connStr = customURL
		return nil
	}

	port := uint32(5433)
	if portEnv := os.Getenv("ZURI_DB_PORT"); portEnv != "" {
		if p, err := strconv.ParseUint(portEnv, 10, 32); err == nil {
			port = uint32(p)
		}
	}

	dbPath := os.Getenv("ZURI_DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dbPath = filepath.Join(".", "data", "zuri_db")
		} else {
			dbPath = filepath.Join(homeDir, ".zuri", "db")
		}
	}

	cfg := embeddedpostgres.DefaultConfig().
		Port(port).
		DataPath(filepath.Join(dbPath, "data")).
		RuntimePath(filepath.Join(dbPath, "runtime")).
		BinariesPath(filepath.Join(dbPath, "binaries")).
		Database("zuri_db").
		Username("zuri").
		Password("zuri_pass")

	m.postgres = embeddedpostgres.NewDatabase(cfg)

	if err := m.postgres.Start(); err != nil {
		return fmt.Errorf("failed to start embedded Postgres: %w", err)
	}

	m.connStr = fmt.Sprintf("host=localhost port=%d user=zuri password=zuri_pass dbname=zuri_db sslmode=disable", port)
	db, err := sql.Open("postgres", m.connStr)
	if err != nil {
		m.postgres.Stop()
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		m.postgres.Stop()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.db = db
	return nil
}

func (m *DBManager) GetDB() *sql.DB {
	return m.db
}

func (m *DBManager) Close() error {
	var errs []error
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close sql.DB: %w", err))
		}
		m.db = nil
	}
	if m.postgres != nil {
		if err := m.postgres.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop embedded postgres: %w", err))
		}
		m.postgres = nil
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
