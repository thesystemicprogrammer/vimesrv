package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewDatabaseMigration(t *testing.T) {
	db := setupTestDB(t)
	dm := NewDatabaseMigration(db)

	assert.NotNil(t, dm)
	assert.NotNil(t, dm.db)
	assert.NotNil(t, dm.migrationJobs)
	assert.Len(t, dm.migrationJobs, 11)
	assert.Equal(t, 1, dm.migrationJobs[0].version)
	assert.Equal(t, 2, dm.migrationJobs[1].version)
	assert.Equal(t, 3, dm.migrationJobs[2].version)
}

func TestMigrate_FreshDatabase(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	dm := NewDatabaseMigration(db)

	// Execute
	err := dm.Migrate()

	// Assert - migrations should succeed on fresh database
	// The first migration creates the schema_migrations table
	require.NoError(t, err)

	// Verify schema_migrations table was created with all migrations
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 11, count, "all migrations should be applied")

	// Verify media table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='media_files'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "media_files", tableName)
}

func TestMigrate_WithInitialSchemaTable(t *testing.T) {
	// Setup - manually create schema_migrations table first
	db := setupTestDB(t)

	// Create empty schema_migrations table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	dm := NewDatabaseMigration(db)

	// Execute
	err = dm.Migrate()
	require.NoError(t, err)

	// Verify schema_migrations table has correct records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 11, count)

	// Verify media table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='media_files'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "media_files", tableName)

	// Verify correct versions were recorded
	var maxVersion int
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&maxVersion)
	require.NoError(t, err)
	assert.Equal(t, 11, maxVersion)

	// Verify both migration names are recorded
	rows, err := db.Query("SELECT version, name FROM schema_migrations ORDER BY version")
	require.NoError(t, err)
	defer rows.Close()

	migrations := []struct {
		version int
		name    string
	}{}
	for rows.Next() {
		var version int
		var name string
		err := rows.Scan(&version, &name)
		require.NoError(t, err)
		migrations = append(migrations, struct {
			version int
			name    string
		}{version, name})
	}

	require.Len(t, migrations, 11)
	assert.Equal(t, 1, migrations[0].version)
	assert.Equal(t, "create_schema_migrations_table", migrations[0].name)
	assert.Equal(t, 2, migrations[1].version)
	assert.Equal(t, "create_media_files_table", migrations[1].name)
	assert.Equal(t, 3, migrations[2].version)
	assert.Equal(t, "create_transcoding_tables", migrations[2].name)
}

func TestMigrate_Idempotency(t *testing.T) {
	// Setup - create schema_migrations table first
	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	dm := NewDatabaseMigration(db)

	// Execute first migration
	err = dm.Migrate()
	require.NoError(t, err)

	// Execute second migration (should be idempotent)
	err = dm.Migrate()
	require.NoError(t, err)

	// Verify migrations weren't duplicated
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 11, count, "migrations should not be duplicated on second run")
}

func TestMigrate_PartiallyMigrated(t *testing.T) {
	// Setup
	db := setupTestDB(t)

	// Manually apply first migration only
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'create_schema_migrations_table')")
	require.NoError(t, err)

	// Execute migrations
	dm := NewDatabaseMigration(db)
	err = dm.Migrate()
	require.NoError(t, err)

	// Verify only the second migration was applied
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 11, count)

	// Verify media table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='media_files'").Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "media_files", tableName)
}

func TestMigrate_AlreadyUpToDate(t *testing.T) {
	// Setup - create schema_migrations table first
	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	dm := NewDatabaseMigration(db)

	// Run migrations to completion
	err = dm.Migrate()
	require.NoError(t, err)

	// Get count before second run
	var countBefore int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countBefore)
	require.NoError(t, err)

	// Run migrations again
	err = dm.Migrate()
	require.NoError(t, err)

	// Verify count is unchanged
	var countAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countAfter)
	require.NoError(t, err)
	assert.Equal(t, countBefore, countAfter)
}

func TestGetCurrentVersion_NoTable(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	dm := NewDatabaseMigration(db)

	// Execute - schema_migrations table doesn't exist
	version, err := dm.getCurrentVersion()

	// Assert - should handle gracefully and return 0 with no error
	// getCurrentVersion is designed to handle missing table by returning 0
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

func TestGetCurrentVersion_EmptyTable(t *testing.T) {
	// Setup
	db := setupTestDB(t)

	// Create empty schema_migrations table
	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	dm := NewDatabaseMigration(db)

	// Execute
	version, err := dm.getCurrentVersion()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

func TestGetCurrentVersion_WithMigrations(t *testing.T) {
	// Setup
	db := setupTestDB(t)

	// Create and populate schema_migrations table
	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'migration_1')")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (2, 'migration_2')")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (5, 'migration_5')")
	require.NoError(t, err)

	dm := NewDatabaseMigration(db)

	// Execute
	version, err := dm.getCurrentVersion()

	// Assert - should return maximum version
	require.NoError(t, err)
	assert.Equal(t, 5, version)
}

func TestMigrate_InvalidSQL(t *testing.T) {
	// Setup - create schema_migrations table first
	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Create a DatabaseMigration with invalid SQL
	dm := &DatabaseMigration{
		db: db,
		migrationJobs: []MigrationJob{
			{
				version: 1,
				name:    "invalid_migration",
				up:      "THIS IS NOT VALID SQL SYNTAX;;;",
				down:    "",
			},
		},
	}

	// Execute
	err = dm.Migrate()

	// Assert - should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "syntax error")
}

func TestMigrate_FailedTransactionRollback(t *testing.T) {
	// Setup
	db := setupTestDB(t)

	// First create the schema_migrations table
	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Create a migration that will fail during execution
	// The first statement succeeds, but the second fails
	dm := &DatabaseMigration{
		db: db,
		migrationJobs: []MigrationJob{
			{
				version: 1,
				name:    "failing_migration",
				up: `
					CREATE TABLE test_table (id INTEGER PRIMARY KEY);
					INSERT INTO nonexistent_table VALUES (1);
				`,
				down: "",
			},
		},
	}

	// Execute
	err = dm.Migrate()

	// Assert - should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such table")

	// Verify transaction was rolled back - test_table should not exist
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	assert.Error(t, err, "test_table should not exist after rollback")
	assert.Equal(t, sql.ErrNoRows, err)

	// Verify migration was not recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "failed migration should not be recorded")
}

func TestRecordMigration_DatabaseError(t *testing.T) {
	// Setup
	db := setupTestDB(t)

	// Create schema_migrations table with a constraint that will be violated
	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
	`)
	require.NoError(t, err)

	// Insert a migration
	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'test_migration')")
	require.NoError(t, err)

	// Create a migration with duplicate name to trigger constraint violation
	dm := &DatabaseMigration{
		db: db,
		migrationJobs: []MigrationJob{
			{
				version: 2,
				name:    "test_migration", // Duplicate name will violate UNIQUE constraint
				up:      "CREATE TABLE test (id INTEGER);",
				down:    "",
			},
		},
	}

	// Execute
	err = dm.Migrate()

	// Assert - should return an error about recording failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UNIQUE constraint failed")

	// Verify test table was not created (transaction rolled back)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "test table should not exist after rollback")
}
