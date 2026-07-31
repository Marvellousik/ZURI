package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

func RunMigrations(db *sql.DB) (int, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	appliedRows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version ASC;")
	if err != nil {
		return 0, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer appliedRows.Close()

	applied := make(map[int]bool)
	for appliedRows.Next() {
		var v int
		if err := appliedRows.Scan(&v); err != nil {
			return 0, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[v] = true
	}
	if err := appliedRows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating migration rows: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue;
		}

		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue;
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue;
		}

		content, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return 0, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	appliedCount := 0
	for _, m := range migrations {
		if applied[m.Version] {
			continue;
		}

		tx, err := db.Begin()
		if err != nil {
			return appliedCount, fmt.Errorf("failed to start transaction for migration %s: %w", m.Name, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return appliedCount, fmt.Errorf("failed to execute migration %s: %w", m.Name, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2);", m.Version, m.Name); err != nil {
			tx.Rollback()
			return appliedCount, fmt.Errorf("failed to record migration %s: %w", m.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return appliedCount, fmt.Errorf("failed to commit migration %s: %w", m.Name, err)
		}

		appliedCount++
	}

	if err := ValidateVectorExtension(db); err != nil {
		return appliedCount, err
	}

	return appliedCount, nil
}

func ValidateVectorExtension(db *sql.DB) error {
	if os.Getenv("ZURI_SKIP_VECTOR_CHECK") == "1" {
		return nil
	}
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector');").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to query pg_extension table: %w", err)
	}
	if !exists {
		return fmt.Errorf("pgvector extension is required for Zuri daemon operation, but is not installed in this PostgreSQL instance")
	}

	var distance float64
	err = db.QueryRow("SELECT ('[1.0, 0.0]'::vector <=> '[1.0, 0.0]'::vector);").Scan(&distance)
	if err != nil {
		return fmt.Errorf("pgvector similarity query test failed: %w", err)
	}

	return nil
}
