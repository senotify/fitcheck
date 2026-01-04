package repository

import (
	"context"
	"errors"
	"virtual-fitcheck/internal/models"
)

// Common repository errors
var (
	ErrJobNotFound = errors.New("job not found")
)

// JobFilter represents filtering and pagination options for job queries
type JobFilter struct {
	SessionID string
	Status    *string
	SortOrder string // "newest" or "oldest"
	Limit     int
	Offset    int
}

// JobListResult represents a paginated list of jobs with metadata
type JobListResult struct {
	Jobs   []*models.ProcessingJob
	Total  int
	Limit  int
	Offset int
}

// JobRepository defines the interface for job data access
type JobRepository interface {
	// CreateJob creates a new job in the database
	CreateJob(ctx context.Context, job *models.ProcessingJob) error

	// GetJob retrieves a job by its ID
	GetJob(ctx context.Context, jobID string) (*models.ProcessingJob, error)

	// GetJobsBySession retrieves all jobs for a given session ID
	GetJobsBySession(ctx context.Context, sessionID string) ([]*models.ProcessingJob, error)

	// GetJobsWithFilter retrieves jobs with filtering, sorting, and pagination
	GetJobsWithFilter(ctx context.Context, filter JobFilter) (*JobListResult, error)

	// UpdateJob updates an existing job in the database
	UpdateJob(ctx context.Context, job *models.ProcessingJob) error

	// DeleteJob removes a job from the database
	DeleteJob(ctx context.Context, jobID string) error

	// GetJobByToken retrieves a job by its result token
	GetJobByToken(ctx context.Context, token string) (*models.ProcessingJob, error)

	// BeginTx starts a new transaction and returns a repository bound to that transaction
	BeginTx(ctx context.Context) (JobRepository, error)

	// Commit commits the current transaction
	Commit() error

	// Rollback rolls back the current transaction
	Rollback() error
}
