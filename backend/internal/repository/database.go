package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	URL         string
	MaxConns    int
	MaxIdle     int
	MaxLifetime time.Duration
}

// NewDatabase creates a new database connection with connection pooling
func NewDatabase(config DatabaseConfig) (*sql.DB, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := sql.Open("postgres", config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	if config.MaxConns > 0 {
		db.SetMaxOpenConns(config.MaxConns)
	} else {
		db.SetMaxOpenConns(25) // Default
	}

	if config.MaxIdle > 0 {
		db.SetMaxIdleConns(config.MaxIdle)
	} else {
		db.SetMaxIdleConns(5) // Default
	}

	if config.MaxLifetime > 0 {
		db.SetConnMaxLifetime(config.MaxLifetime)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute) // Default
	}

	// Verify connection with retry logic
	if err := pingWithRetry(db, 3, 2*time.Second); err != nil {
		return nil, fmt.Errorf("failed to ping database after retries: %w", err)
	}

	return db, nil
}

// pingWithRetry attempts to ping the database with exponential backoff
func pingWithRetry(db *sql.DB, maxAttempts int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := db.Ping()
		if err == nil {
			if attempt > 1 {
				log.Printf("Database connection established after %d attempts", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("Database ping attempt %d/%d failed: %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			log.Printf("Retrying in %v...", delay)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("all connection attempts failed: %w", lastErr)
}

// CheckDatabaseHealth checks if the database connection is healthy
func CheckDatabaseHealth(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
