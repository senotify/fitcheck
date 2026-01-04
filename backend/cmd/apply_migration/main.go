package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Apply the new migration
	migration := `
-- Add index for result_token lookups
-- This improves performance for email link access
CREATE INDEX IF NOT EXISTS idx_jobs_result_token ON jobs(result_token) WHERE result_token IS NOT NULL;
`

	_, err = db.Exec(migration)
	if err != nil {
		log.Fatalf("Failed to apply migration: %v", err)
	}

	fmt.Println("Migration applied successfully!")

	// Verify the index was created
	var indexExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE tablename = 'jobs' 
			AND indexname = 'idx_jobs_result_token'
		)
	`).Scan(&indexExists)
	if err != nil {
		log.Fatalf("Failed to verify index: %v", err)
	}

	if indexExists {
		fmt.Println("Index idx_jobs_result_token verified!")
	} else {
		fmt.Println("Warning: Index was not created")
	}
}
