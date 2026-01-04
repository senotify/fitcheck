package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✓ Database connection successful")

	// Test migration down (cleanup first)
	fmt.Println("\nTesting migration DOWN...")
	if err := runMigrationDown(db); err != nil {
		log.Printf("Warning during migration down: %v", err)
	}
	fmt.Println("✓ Migration DOWN completed")

	// Test migration up
	fmt.Println("\nTesting migration UP...")
	if err := runMigrationUp(db); err != nil {
		log.Fatalf("Migration UP failed: %v", err)
	}
	fmt.Println("✓ Migration UP completed")

	// Verify table exists
	fmt.Println("\nVerifying table structure...")
	if err := verifyTable(db); err != nil {
		log.Fatalf("Table verification failed: %v", err)
	}
	fmt.Println("✓ Table structure verified")

	// Verify indexes exist
	fmt.Println("\nVerifying indexes...")
	if err := verifyIndexes(db); err != nil {
		log.Fatalf("Index verification failed: %v", err)
	}
	fmt.Println("✓ Indexes verified")

	// Verify trigger exists
	fmt.Println("\nVerifying trigger...")
	if err := verifyTrigger(db); err != nil {
		log.Fatalf("Trigger verification failed: %v", err)
	}
	fmt.Println("✓ Trigger verified")

	// Test trigger functionality
	fmt.Println("\nTesting trigger functionality...")
	if err := testTrigger(db); err != nil {
		log.Fatalf("Trigger test failed: %v", err)
	}
	fmt.Println("✓ Trigger works correctly")

	// Test migration down again
	fmt.Println("\nTesting migration DOWN (cleanup)...")
	if err := runMigrationDown(db); err != nil {
		log.Fatalf("Migration DOWN failed: %v", err)
	}
	fmt.Println("✓ Migration DOWN completed")

	// Verify table is dropped
	fmt.Println("\nVerifying table is dropped...")
	if err := verifyTableDropped(db); err != nil {
		log.Fatalf("Table should be dropped: %v", err)
	}
	fmt.Println("✓ Table successfully dropped")

	fmt.Println("\n✅ All migration tests passed!")
}

func runMigrationUp(db *sql.DB) error {
	// Read migration file
	upSQL, err := os.ReadFile("../../migrations/000001_create_jobs_table.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	_, err = db.Exec(string(upSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func runMigrationDown(db *sql.DB) error {
	// Read migration file
	downSQL, err := os.ReadFile("../../migrations/000001_create_jobs_table.down.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	_, err = db.Exec(string(downSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func verifyTable(db *sql.DB) error {
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'jobs'
		ORDER BY ordinal_position;
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query table structure: %w", err)
	}
	defer rows.Close()

	expectedColumns := map[string]bool{
		"id": false, "session_id": false, "user_photo_id": false,
		"shirt_image_id": false, "status": false, "progress": false,
		"status_message": false, "result_id": false, "result_token": false,
		"error": false, "email": false, "created_at": false,
		"updated_at": false, "completed_at": false,
	}

	foundColumns := 0
	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			return fmt.Errorf("failed to scan column: %w", err)
		}
		if _, exists := expectedColumns[columnName]; exists {
			foundColumns++
			fmt.Printf("  - %s (%s, nullable: %s)\n", columnName, dataType, isNullable)
		}
	}

	if foundColumns != len(expectedColumns) {
		return fmt.Errorf("expected %d columns, found %d", len(expectedColumns), foundColumns)
	}

	return nil
}

func verifyIndexes(db *sql.DB) error {
	query := `
		SELECT indexname
		FROM pg_indexes
		WHERE tablename = 'jobs'
		AND indexname LIKE 'idx_%';
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	expectedIndexes := map[string]bool{
		"idx_jobs_session_id": false,
		"idx_jobs_status":     false,
		"idx_jobs_created_at": false,
	}

	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			return fmt.Errorf("failed to scan index: %w", err)
		}
		if _, exists := expectedIndexes[indexName]; exists {
			expectedIndexes[indexName] = true
			fmt.Printf("  - %s\n", indexName)
		}
	}

	for indexName, found := range expectedIndexes {
		if !found {
			return fmt.Errorf("index %s not found", indexName)
		}
	}

	return nil
}

func verifyTrigger(db *sql.DB) error {
	query := `
		SELECT trigger_name
		FROM information_schema.triggers
		WHERE event_object_table = 'jobs'
		AND trigger_name = 'update_jobs_updated_at';
	`

	var triggerName string
	err := db.QueryRow(query).Scan(&triggerName)
	if err != nil {
		return fmt.Errorf("trigger not found: %w", err)
	}

	fmt.Printf("  - %s\n", triggerName)
	return nil
}

func testTrigger(db *sql.DB) error {
	// Insert a test job
	insertQuery := `
		INSERT INTO jobs (session_id, user_photo_id, shirt_image_id, status)
		VALUES ('test-session', 'photo-123', 'shirt-456', 'pending')
		RETURNING id, created_at, updated_at;
	`

	var jobID string
	var createdAt, updatedAt string
	err := db.QueryRow(insertQuery).Scan(&jobID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert test job: %w", err)
	}

	fmt.Printf("  - Created job %s at %s\n", jobID, createdAt)

	// Wait a moment to ensure timestamp difference
	// (In real scenario, there would be a time difference)
	
	// Update the job
	updateQuery := `
		UPDATE jobs
		SET status = 'completed'
		WHERE id = $1
		RETURNING updated_at;
	`

	var newUpdatedAt string
	err = db.QueryRow(updateQuery, jobID).Scan(&newUpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update test job: %w", err)
	}

	fmt.Printf("  - Updated job, new updated_at: %s\n", newUpdatedAt)

	// Verify updated_at changed
	if updatedAt == newUpdatedAt {
		return fmt.Errorf("updated_at should have changed after update")
	}

	// Clean up test job
	_, err = db.Exec("DELETE FROM jobs WHERE id = $1", jobID)
	if err != nil {
		return fmt.Errorf("failed to delete test job: %w", err)
	}

	return nil
}

func verifyTableDropped(db *sql.DB) error {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'jobs'
		);
	`

	var exists bool
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		return fmt.Errorf("table 'jobs' still exists after migration down")
	}

	return nil
}
