package services

import (
	"context"
	"fmt"
	"time"

	"virtual-fitcheck/internal/models"
	"virtual-fitcheck/internal/repository"

	"github.com/google/uuid"
)

// JobService manages processing jobs using database persistence
type JobService struct {
	repo repository.JobRepository
}

// NewJobService creates a new JobService instance
func NewJobService(repo repository.JobRepository) *JobService {
	return &JobService{
		repo: repo,
	}
}

// CreateJob creates a new processing job
func (js *JobService) CreateJob(sessionID, userPhotoID, shirtImageID string) (*models.ProcessingJob, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}
	if userPhotoID == "" {
		return nil, fmt.Errorf("userPhotoID is required")
	}
	if shirtImageID == "" {
		return nil, fmt.Errorf("shirtImageID is required")
	}

	jobID := uuid.New().String()
	now := time.Now()

	job := &models.ProcessingJob{
		JobID:         jobID,
		SessionID:     sessionID,
		UserPhotoID:   userPhotoID,
		ShirtImageID:  shirtImageID,
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created, waiting to start processing",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	ctx := context.Background()
	err := js.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetJob retrieves a job by its ID
func (js *JobService) GetJob(jobID string) (*models.ProcessingJob, error) {
	ctx := context.Background()
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// UpdateJobStatus updates the status of a job
func (js *JobService) UpdateJobStatus(jobID string, status models.JobStatus, message string) error {
	ctx := context.Background()
	
	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Validate completed jobs have result URL
	if status == models.JobStatusCompleted {
		if job.ResultID == nil || *job.ResultID == "" {
			return fmt.Errorf("cannot mark job as completed: result URL is not set")
		}
	}

	// Validate failed jobs have error message
	if status == models.JobStatusFailed {
		if job.Error == nil || *job.Error == "" {
			return fmt.Errorf("cannot mark job as failed: error message is not set")
		}
	}

	// Update fields
	job.Status = status
	job.StatusMessage = message

	// Set completion time if job is completed or failed
	if status == models.JobStatusCompleted || status == models.JobStatusFailed {
		now := time.Now()
		job.CompletedAt = &now
	}

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// UpdateJobProgress updates the progress of a job
func (js *JobService) UpdateJobProgress(jobID string, progress int, message string) error {
	ctx := context.Background()
	
	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Clamp progress between 0 and 100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	// Update fields
	job.Progress = progress
	if message != "" {
		job.StatusMessage = message
	}

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// SetJobResult sets the result ID for a completed job
func (js *JobService) SetJobResult(jobID string, resultID string) error {
	ctx := context.Background()
	
	// Validate result ID is not empty
	if resultID == "" {
		return fmt.Errorf("result ID cannot be empty")
	}

	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update result ID
	job.ResultID = &resultID

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to set job result: %w", err)
	}

	return nil
}

// SetJobError sets the error message for a failed job
func (js *JobService) SetJobError(jobID string, errorMsg string) error {
	ctx := context.Background()
	
	// Validate error message is not empty
	if errorMsg == "" {
		return fmt.Errorf("error message cannot be empty")
	}

	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update error and status
	job.Error = &errorMsg
	job.Status = models.JobStatusFailed
	now := time.Now()
	job.CompletedAt = &now

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to set job error: %w", err)
	}

	return nil
}

// DeleteJob removes a job from the store
func (js *JobService) DeleteJob(jobID string) error {
	ctx := context.Background()
	
	err := js.repo.DeleteJob(ctx, jobID)
	if err != nil {
		return err
	}

	return nil
}

// SetJobEmail sets the email address for a job (for notification purposes)
func (js *JobService) SetJobEmail(jobID string, email string) error {
	ctx := context.Background()
	
	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update email
	job.Email = &email

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to set job email: %w", err)
	}

	return nil
}

// SetJobResultToken sets the result token for secure result access
func (js *JobService) SetJobResultToken(jobID string, token string) error {
	ctx := context.Background()
	
	// Get the current job
	job, err := js.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update result token
	job.ResultToken = &token

	// Persist changes
	err = js.repo.UpdateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to set job result token: %w", err)
	}

	return nil
}

// GetJobByToken retrieves a job by its result token
func (js *JobService) GetJobByToken(token string) (*models.ProcessingJob, error) {
	ctx := context.Background()
	
	job, err := js.repo.GetJobByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// ListJobs returns all jobs (useful for debugging/admin)
func (js *JobService) ListJobs() []*models.ProcessingJob {
	// This method is deprecated and should not be used with database storage
	// Use GetJobsBySession instead
	return []*models.ProcessingJob{}
}

// GetJobsBySession retrieves all jobs for a given session ID
func (js *JobService) GetJobsBySession(sessionID string) ([]*models.ProcessingJob, error) {
	ctx := context.Background()
	
	jobs, err := js.repo.GetJobsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by session: %w", err)
	}

	return jobs, nil
}

// GetJobsWithFilter retrieves jobs with filtering, sorting, and pagination
func (js *JobService) GetJobsWithFilter(ctx context.Context, filter repository.JobFilter) (*repository.JobListResult, error) {
	result, err := js.repo.GetJobsWithFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with filter: %w", err)
	}

	return result, nil
}
