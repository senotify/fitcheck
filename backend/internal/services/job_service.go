package services

import (
	"fmt"
	"sync"
	"time"

	"virtual-fitcheck/internal/models"

	"github.com/google/uuid"
)

// JobService manages processing jobs in memory
type JobService struct {
	mu   sync.RWMutex
	jobs map[string]*models.ProcessingJob
}

// NewJobService creates a new JobService instance
func NewJobService() *JobService {
	return &JobService{
		jobs: make(map[string]*models.ProcessingJob),
	}
}

// CreateJob creates a new processing job
func (js *JobService) CreateJob(userPhotoID, shirtImageID string) (*models.ProcessingJob, error) {
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
		UserPhotoID:   userPhotoID,
		ShirtImageID:  shirtImageID,
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created, waiting to start processing",
		CreatedAt:     now,
	}

	js.mu.Lock()
	js.jobs[jobID] = job
	js.mu.Unlock()

	return job, nil
}

// GetJob retrieves a job by its ID
func (js *JobService) GetJob(jobID string) (*models.ProcessingJob, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Return a copy to prevent external modifications
	jobCopy := *job
	return &jobCopy, nil
}

// UpdateJobStatus updates the status of a job
func (js *JobService) UpdateJobStatus(jobID string, status models.JobStatus, message string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Status = status
	job.StatusMessage = message

	// Set completion time if job is completed or failed
	if status == models.JobStatusCompleted || status == models.JobStatusFailed {
		now := time.Now()
		job.CompletedAt = &now
	}

	return nil
}

// UpdateJobProgress updates the progress of a job
func (js *JobService) UpdateJobProgress(jobID string, progress int, message string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Clamp progress between 0 and 100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	job.Progress = progress
	if message != "" {
		job.StatusMessage = message
	}

	return nil
}

// SetJobResult sets the result ID for a completed job
func (js *JobService) SetJobResult(jobID string, resultID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.ResultID = &resultID
	return nil
}

// SetJobError sets the error message for a failed job
func (js *JobService) SetJobError(jobID string, errorMsg string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Error = &errorMsg
	job.Status = models.JobStatusFailed
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// DeleteJob removes a job from the store
func (js *JobService) DeleteJob(jobID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if _, exists := js.jobs[jobID]; !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	delete(js.jobs, jobID)
	return nil
}

// SetJobEmail sets the email address for a job (for notification purposes)
func (js *JobService) SetJobEmail(jobID string, email string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Email = &email
	return nil
}

// SetJobResultToken sets the result token for secure result access
func (js *JobService) SetJobResultToken(jobID string, token string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.ResultToken = &token
	return nil
}

// ListJobs returns all jobs (useful for debugging/admin)
func (js *JobService) ListJobs() []*models.ProcessingJob {
	js.mu.RLock()
	defer js.mu.RUnlock()

	jobs := make([]*models.ProcessingJob, 0, len(js.jobs))
	for _, job := range js.jobs {
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}

	return jobs
}
