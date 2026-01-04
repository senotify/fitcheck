package services

import (
	"context"
	"testing"
	"time"

	"virtual-fitcheck/internal/models"
	"virtual-fitcheck/internal/repository"
)

// mockJobRepository is a simple in-memory mock for testing validation logic
type mockJobRepository struct {
	jobs map[string]*models.ProcessingJob
}

func newMockJobRepository() *mockJobRepository {
	return &mockJobRepository{
		jobs: make(map[string]*models.ProcessingJob),
	}
}

func (m *mockJobRepository) CreateJob(ctx context.Context, job *models.ProcessingJob) error {
	m.jobs[job.JobID] = job
	return nil
}

func (m *mockJobRepository) GetJob(ctx context.Context, jobID string) (*models.ProcessingJob, error) {
	job, exists := m.jobs[jobID]
	if !exists {
		return nil, repository.ErrJobNotFound
	}
	// Return a copy to simulate database behavior
	jobCopy := *job
	return &jobCopy, nil
}

func (m *mockJobRepository) GetJobsBySession(ctx context.Context, sessionID string) ([]*models.ProcessingJob, error) {
	var jobs []*models.ProcessingJob
	for _, job := range m.jobs {
		if job.SessionID == sessionID {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (m *mockJobRepository) GetJobsWithFilter(ctx context.Context, filter repository.JobFilter) (*repository.JobListResult, error) {
	var jobs []*models.ProcessingJob
	for _, job := range m.jobs {
		if job.SessionID == filter.SessionID {
			// Apply status filter if provided
			if filter.Status != nil && string(job.Status) != *filter.Status {
				continue
			}
			jobs = append(jobs, job)
		}
	}
	
	// Simple sorting (newest first by default)
	// For mock purposes, we'll just return the jobs as-is
	
	// Apply pagination
	start := filter.Offset
	end := filter.Offset + filter.Limit
	if start > len(jobs) {
		start = len(jobs)
	}
	if end > len(jobs) {
		end = len(jobs)
	}
	
	paginatedJobs := jobs[start:end]
	
	return &repository.JobListResult{
		Jobs:   paginatedJobs,
		Total:  len(jobs),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (m *mockJobRepository) UpdateJob(ctx context.Context, job *models.ProcessingJob) error {
	if _, exists := m.jobs[job.JobID]; !exists {
		return repository.ErrJobNotFound
	}
	m.jobs[job.JobID] = job
	return nil
}

func (m *mockJobRepository) DeleteJob(ctx context.Context, jobID string) error {
	if _, exists := m.jobs[jobID]; !exists {
		return repository.ErrJobNotFound
	}
	delete(m.jobs, jobID)
	return nil
}

func (m *mockJobRepository) GetJobByToken(ctx context.Context, token string) (*models.ProcessingJob, error) {
	for _, job := range m.jobs {
		if job.ResultToken != nil && *job.ResultToken == token {
			return job, nil
		}
	}
	return nil, repository.ErrJobNotFound
}

func (m *mockJobRepository) BeginTx(ctx context.Context) (repository.JobRepository, error) {
	return m, nil
}

func (m *mockJobRepository) Commit() error {
	return nil
}

func (m *mockJobRepository) Rollback() error {
	return nil
}

// TestJobValidationUnit tests validation logic without requiring a database
func TestJobValidationUnit(t *testing.T) {
	repo := newMockJobRepository()
	js := NewJobService(repo)

	t.Run("SetJobResult rejects empty result ID", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to set empty result
		err = js.SetJobResult(job.JobID, "")
		if err == nil {
			t.Error("expected error when setting empty result ID")
		}
		if err != nil && err.Error() != "result ID cannot be empty" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("SetJobResult accepts non-empty result ID", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set valid result
		err = js.SetJobResult(job.JobID, "result-789")
		if err != nil {
			t.Errorf("unexpected error when setting valid result ID: %v", err)
		}

		// Verify result was set
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.ResultID == nil || *updated.ResultID != "result-789" {
			t.Error("result ID was not set correctly")
		}
	})

	t.Run("SetJobError rejects empty error message", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to set empty error
		err = js.SetJobError(job.JobID, "")
		if err == nil {
			t.Error("expected error when setting empty error message")
		}
		if err != nil && err.Error() != "error message cannot be empty" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("SetJobError accepts non-empty error message and sets status to failed", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set valid error
		err = js.SetJobError(job.JobID, "Something went wrong")
		if err != nil {
			t.Errorf("unexpected error when setting valid error message: %v", err)
		}

		// Verify error was set and status is failed
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Error == nil || *updated.Error != "Something went wrong" {
			t.Error("error message was not set correctly")
		}
		if updated.Status != models.JobStatusFailed {
			t.Errorf("expected status to be failed, got %s", updated.Status)
		}
		if updated.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("UpdateJobStatus rejects completed status without result", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to mark as completed without setting result
		err = js.UpdateJobStatus(job.JobID, models.JobStatusCompleted, "Done")
		if err == nil {
			t.Error("expected error when marking job as completed without result")
		}
		if err != nil && err.Error() != "cannot mark job as completed: result URL is not set" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("UpdateJobStatus accepts completed status with result", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set result first
		err = js.SetJobResult(job.JobID, "result-789")
		if err != nil {
			t.Fatalf("failed to set result: %v", err)
		}

		// Now mark as completed
		err = js.UpdateJobStatus(job.JobID, models.JobStatusCompleted, "Done")
		if err != nil {
			t.Errorf("unexpected error when marking job as completed with result: %v", err)
		}

		// Verify status and completion time
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Status != models.JobStatusCompleted {
			t.Errorf("expected status to be completed, got %s", updated.Status)
		}
		if updated.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
		if updated.StatusMessage != "Done" {
			t.Errorf("expected status message to be 'Done', got %s", updated.StatusMessage)
		}
	})

	t.Run("UpdateJobStatus rejects failed status without error message", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to mark as failed without setting error
		err = js.UpdateJobStatus(job.JobID, models.JobStatusFailed, "Failed")
		if err == nil {
			t.Error("expected error when marking job as failed without error message")
		}
		if err != nil && err.Error() != "cannot mark job as failed: error message is not set" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("UpdateJobStatus accepts failed status with error message", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set error first using SetJobError (which also sets status to failed)
		err = js.SetJobError(job.JobID, "Something went wrong")
		if err != nil {
			t.Fatalf("failed to set error: %v", err)
		}

		// Verify status is already set to failed
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Status != models.JobStatusFailed {
			t.Errorf("expected status to be failed, got %s", updated.Status)
		}
		if updated.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("UpdateJobStatus allows pending and processing without validation", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Update to processing (should work without result or error)
		err = js.UpdateJobStatus(job.JobID, models.JobStatusProcessing, "Processing started")
		if err != nil {
			t.Errorf("unexpected error when updating to processing: %v", err)
		}

		// Verify status
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Status != models.JobStatusProcessing {
			t.Errorf("expected status to be processing, got %s", updated.Status)
		}
		if updated.CompletedAt != nil {
			t.Error("expected CompletedAt to be nil for processing job")
		}
	})

	t.Run("Completed job has completion timestamp", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set result and mark as completed
		err = js.SetJobResult(job.JobID, "result-789")
		if err != nil {
			t.Fatalf("failed to set result: %v", err)
		}

		beforeComplete := time.Now()
		err = js.UpdateJobStatus(job.JobID, models.JobStatusCompleted, "Done")
		afterComplete := time.Now()
		if err != nil {
			t.Fatalf("failed to mark as completed: %v", err)
		}

		// Verify completion timestamp is set and reasonable
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}
		if updated.CompletedAt.Before(beforeComplete) || updated.CompletedAt.After(afterComplete) {
			t.Error("CompletedAt timestamp is not within expected range")
		}
	})

	t.Run("Failed job has completion timestamp", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		beforeFail := time.Now()
		err = js.SetJobError(job.JobID, "Something went wrong")
		afterFail := time.Now()
		if err != nil {
			t.Fatalf("failed to set error: %v", err)
		}

		// Verify completion timestamp is set and reasonable
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}
		if updated.CompletedAt.Before(beforeFail) || updated.CompletedAt.After(afterFail) {
			t.Error("CompletedAt timestamp is not within expected range")
		}
	})
}
