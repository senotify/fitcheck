package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"virtual-fitcheck/internal/models"

	"github.com/google/uuid"
)

// PostgresJobRepository implements JobRepository using PostgreSQL
type PostgresJobRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPostgresJobRepository creates a new PostgreSQL job repository
func NewPostgresJobRepository(db *sql.DB) *PostgresJobRepository {
	return &PostgresJobRepository{
		db: db,
	}
}

// getExecutor returns the appropriate executor (transaction or database)
func (r *PostgresJobRepository) getExecutor() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// CreateJob creates a new job in the database
func (r *PostgresJobRepository) CreateJob(ctx context.Context, job *models.ProcessingJob) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	// Generate UUID if not provided
	if job.JobID == "" {
		job.JobID = uuid.New().String()
	}

	// Set timestamps if not provided
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}

	query := `
		INSERT INTO jobs (
			id, session_id, user_photo_id, shirt_image_id, status, 
			progress, status_message, result_id, result_token, error, 
			email, created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	executor := r.getExecutor()
	_, err := executor.ExecContext(
		ctx,
		query,
		job.JobID,
		job.SessionID,
		job.UserPhotoID,
		job.ShirtImageID,
		job.Status,
		job.Progress,
		job.StatusMessage,
		job.ResultID,
		job.ResultToken,
		job.Error,
		job.Email,
		job.CreatedAt,
		job.UpdatedAt,
		job.CompletedAt,
	)

	if err != nil {
		log.Printf("ERROR: Failed to create job %s in database: %v", job.JobID, err)
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by its ID
func (r *PostgresJobRepository) GetJob(ctx context.Context, jobID string) (*models.ProcessingJob, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	query := `
		SELECT 
			id, session_id, user_photo_id, shirt_image_id, status, 
			progress, status_message, result_id, result_token, error, 
			email, created_at, updated_at, completed_at
		FROM jobs
		WHERE id = $1
	`

	executor := r.getExecutor()
	row := executor.QueryRowContext(ctx, query, jobID)

	job := &models.ProcessingJob{}
	err := row.Scan(
		&job.JobID,
		&job.SessionID,
		&job.UserPhotoID,
		&job.ShirtImageID,
		&job.Status,
		&job.Progress,
		&job.StatusMessage,
		&job.ResultID,
		&job.ResultToken,
		&job.Error,
		&job.Email,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		log.Printf("ERROR: Failed to get job %s from database: %v", jobID, err)
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// GetJobsBySession retrieves all jobs for a given session ID
func (r *PostgresJobRepository) GetJobsBySession(ctx context.Context, sessionID string) ([]*models.ProcessingJob, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	query := `
		SELECT 
			id, session_id, user_photo_id, shirt_image_id, status, 
			progress, status_message, result_id, result_token, error, 
			email, created_at, updated_at, completed_at
		FROM jobs
		WHERE session_id = $1
		ORDER BY created_at DESC
	`

	executor := r.getExecutor()
	rows, err := executor.QueryContext(ctx, query, sessionID)
	if err != nil {
		log.Printf("ERROR: Failed to query jobs for session %s: %v", sessionID, err)
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.ProcessingJob
	for rows.Next() {
		job := &models.ProcessingJob{}
		err := rows.Scan(
			&job.JobID,
			&job.SessionID,
			&job.UserPhotoID,
			&job.ShirtImageID,
			&job.Status,
			&job.Progress,
			&job.StatusMessage,
			&job.ResultID,
			&job.ResultToken,
			&job.Error,
			&job.Email,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating jobs: %w", err)
	}

	return jobs, nil
}

// GetJobsWithFilter retrieves jobs with filtering, sorting, and pagination
// This method is optimized to use database indexes efficiently
func (r *PostgresJobRepository) GetJobsWithFilter(ctx context.Context, filter JobFilter) (*JobListResult, error) {
	if filter.SessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	// Set default values
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "newest"
	}

	// Build the WHERE clause
	whereClause := "WHERE session_id = $1"
	args := []interface{}{filter.SessionID}
	argCount := 1

	// Add status filter if provided
	if filter.Status != nil && *filter.Status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	// Build the ORDER BY clause
	// Use the indexed created_at column for efficient sorting
	orderClause := "ORDER BY created_at DESC"
	if filter.SortOrder == "oldest" {
		orderClause = "ORDER BY created_at ASC"
	}

	// Build the main query with LIMIT and OFFSET for pagination
	// This query will use the idx_jobs_session_id and idx_jobs_created_at indexes
	query := fmt.Sprintf(`
		SELECT 
			id, session_id, user_photo_id, shirt_image_id, status, 
			progress, status_message, result_id, result_token, error, 
			email, created_at, updated_at, completed_at
		FROM jobs
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argCount+1, argCount+2)

	args = append(args, filter.Limit, filter.Offset)

	// Execute the query
	executor := r.getExecutor()
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ERROR: Failed to query jobs with filter: %v", err)
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	// Scan results
	var jobs []*models.ProcessingJob
	for rows.Next() {
		job := &models.ProcessingJob{}
		err := rows.Scan(
			&job.JobID,
			&job.SessionID,
			&job.UserPhotoID,
			&job.ShirtImageID,
			&job.Status,
			&job.Progress,
			&job.StatusMessage,
			&job.ResultID,
			&job.ResultToken,
			&job.Error,
			&job.Email,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating jobs: %w", err)
	}

	// Get total count for pagination metadata
	// This query will use the idx_jobs_session_id index
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs %s", whereClause)
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET args

	var total int
	err = executor.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		log.Printf("ERROR: Failed to count jobs: %v", err)
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	return &JobListResult{
		Jobs:   jobs,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// UpdateJob updates an existing job in the database
func (r *PostgresJobRepository) UpdateJob(ctx context.Context, job *models.ProcessingJob) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}
	if job.JobID == "" {
		return fmt.Errorf("jobID cannot be empty")
	}

	query := `
		UPDATE jobs SET
			session_id = $2,
			user_photo_id = $3,
			shirt_image_id = $4,
			status = $5,
			progress = $6,
			status_message = $7,
			result_id = $8,
			result_token = $9,
			error = $10,
			email = $11,
			completed_at = $12
		WHERE id = $1
	`

	executor := r.getExecutor()
	result, err := executor.ExecContext(
		ctx,
		query,
		job.JobID,
		job.SessionID,
		job.UserPhotoID,
		job.ShirtImageID,
		job.Status,
		job.Progress,
		job.StatusMessage,
		job.ResultID,
		job.ResultToken,
		job.Error,
		job.Email,
		job.CompletedAt,
	)

	if err != nil {
		log.Printf("ERROR: Failed to update job %s in database: %v", job.JobID, err)
		return fmt.Errorf("failed to update job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", job.JobID)
	}

	return nil
}

// DeleteJob removes a job from the database
func (r *PostgresJobRepository) DeleteJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("jobID cannot be empty")
	}

	query := `DELETE FROM jobs WHERE id = $1`

	executor := r.getExecutor()
	result, err := executor.ExecContext(ctx, query, jobID)
	if err != nil {
		log.Printf("ERROR: Failed to delete job %s from database: %v", jobID, err)
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return nil
}

// GetJobByToken retrieves a job by its result token
func (r *PostgresJobRepository) GetJobByToken(ctx context.Context, token string) (*models.ProcessingJob, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	query := `
		SELECT 
			id, session_id, user_photo_id, shirt_image_id, status, 
			progress, status_message, result_id, result_token, error, 
			email, created_at, updated_at, completed_at
		FROM jobs
		WHERE result_token = $1
	`

	executor := r.getExecutor()
	row := executor.QueryRowContext(ctx, query, token)

	job := &models.ProcessingJob{}
	err := row.Scan(
		&job.JobID,
		&job.SessionID,
		&job.UserPhotoID,
		&job.ShirtImageID,
		&job.Status,
		&job.Progress,
		&job.StatusMessage,
		&job.ResultID,
		&job.ResultToken,
		&job.Error,
		&job.Email,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found for token")
	}
	if err != nil {
		log.Printf("ERROR: Failed to get job by token from database: %v", err)
		return nil, fmt.Errorf("failed to get job by token: %w", err)
	}

	return job, nil
}

// BeginTx starts a new transaction and returns a repository bound to that transaction
func (r *PostgresJobRepository) BeginTx(ctx context.Context) (JobRepository, error) {
	if r.tx != nil {
		return nil, fmt.Errorf("transaction already in progress")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("ERROR: Failed to begin database transaction: %v", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgresJobRepository{
		db: r.db,
		tx: tx,
	}, nil
}

// Commit commits the current transaction
func (r *PostgresJobRepository) Commit() error {
	if r.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	err := r.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.tx = nil
	return nil
}

// Rollback rolls back the current transaction
func (r *PostgresJobRepository) Rollback() error {
	if r.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	err := r.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	r.tx = nil
	return nil
}
