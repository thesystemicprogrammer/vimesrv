package database

import (
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Create temp database
	tmpFile := "/tmp/vimesrv_test_db.sqlite"
	defer os.Remove(tmpFile)

	config := Config{
		Path:            tmpFile,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	db, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Test health check
	if err := db.Health(); err != nil {
		t.Fatalf("Database health check failed: %v", err)
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to check journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal_mode to be 'wal', got '%s'", journalMode)
	}

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("Failed to check foreign keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("Expected foreign_keys to be 1, got %d", foreignKeys)
	}
}

func TestStats(t *testing.T) {
	tmpFile := "/tmp/vimesrv_test_stats.sqlite"
	defer os.Remove(tmpFile)

	config := Config{
		Path:         tmpFile,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}

	db, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 10 {
		t.Errorf("Expected MaxOpenConnections to be 10, got %d", stats.MaxOpenConnections)
	}
}
