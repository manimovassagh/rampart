package database

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestRunMigrationsInvalidDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	err := RunMigrations("postgres://invalid:invalid@localhost:9999/nonexistent?sslmode=disable", "../../migrations", logger)
	if err == nil {
		t.Fatal("expected error running migrations against invalid DB")
	}
}

func TestRunMigrationsMissingDirectory(t *testing.T) {
	dbURL := os.Getenv("RAMPART_DB_URL")
	if dbURL == "" {
		t.Skip("RAMPART_DB_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	err := RunMigrations(dbURL, "/nonexistent/path/migrations", logger)
	if err == nil {
		t.Fatal("expected error for nonexistent migrations directory")
	}
}

func TestRunMigrationsSuccess(t *testing.T) {
	dbURL := os.Getenv("RAMPART_DB_URL")
	if dbURL == "" {
		t.Skip("RAMPART_DB_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	err := RunMigrations(dbURL, "../../migrations", logger)
	if err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// Running again should be idempotent (no change)
	err = RunMigrations(dbURL, "../../migrations", logger)
	if err != nil {
		t.Fatalf("RunMigrations (idempotent): %v", err)
	}
}

func TestMigrationFilesExist(t *testing.T) {
	migrationsDir := "../../migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Skipf("migrations directory not accessible: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("migrations directory is empty")
	}

	// Verify each .up.sql has a corresponding .down.sql
	ups := make(map[string]bool)
	downs := make(map[string]bool)
	for _, e := range entries {
		name := e.Name()
		ext := filepath.Ext(name)
		if ext != ".sql" {
			continue
		}
		base := name[:len(name)-len(ext)]
		if filepath.Ext(base) == ".up" {
			prefix := base[:len(base)-3]
			ups[prefix] = true
		} else if filepath.Ext(base) == ".down" {
			prefix := base[:len(base)-5]
			downs[prefix] = true
		}
	}

	for prefix := range ups {
		if !downs[prefix] {
			t.Errorf("migration %s has .up.sql but no .down.sql", prefix)
		}
	}
	for prefix := range downs {
		if !ups[prefix] {
			t.Errorf("migration %s has .down.sql but no .up.sql", prefix)
		}
	}
}
