package services

import (
	"context"
	"testing"

	"virtual-fitcheck/internal/models"
	"virtual-fitcheck/internal/repository"
)

// TestJobValidation tests the validation logic for completed and failed jobs
func TestJobValidation(t *testing.T) {
	// Create a database connection for testing
	// Use environment variable or default test database
	dbURL := "postgresql://postgres:postgres@localhost:5432/virtualfitcheck_test?sslmode=disable"
	db, err := repository.NewDatabase(repository.DatabaseConfig{
		URL: dbURL,
	})
	
	// If database is not available, skip the test
	if err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}
	defer db.Close()

	repo := repository.NewPostgresJobRepository(db)
	js := NewJobService(repo)

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

		// Clean up
		js.DeleteJob(job.JobID)
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

		// Clean up
		js.DeleteJob(job.JobID)
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

		// Verify status
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Status != models.JobStatusCompleted {
			t.Errorf("expected status to be completed, got %s", updated.Status)
		}

		// Clean up
		js.DeleteJob(job.JobID)
	})

	t.Run("UpdateJobStatus accepts failed status with error message", func(t *testing.T) {
		// Create a job
		job, err := js.CreateJob("session-test", "photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set error first
		err = js.SetJobError(job.JobID, "Something went wrong")
		if err != nil {
			t.Fatalf("failed to set error: %v", err)
		}

		// Verify status is already set to failed by SetJobError
		updated, err := js.GetJob(job.JobID)
		if err != nil {
			t.Fatalf("failed to get job: %v", err)
		}
		if updated.Status != models.JobStatusFailed {
			t.Errorf("expected status to be failed, got %s", updated.Status)
		}

		// Clean up
		js.DeleteJob(job.JobID)
	})

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

		// Clean up
		js.DeleteJob(job.JobID)
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

		// Clean up
		js.DeleteJob(job.JobID)
	})

	t.Run("Database constraint prevents completed job without result", func(t *testing.T) {
		// Create a job directly in the database
		job := &models.ProcessingJob{
			JobID:        "test-job-1",
			SessionID:    "session-test",
			UserPhotoID:  "photo-123",
			ShirtImageID: "shirt-456",
			Status:       models.JobStatusPending,
			Progress:     0,
		}

		ctx := context.Background()
		err := repo.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to update to completed without result (bypassing service validation)
		job.Status = models.JobStatusCompleted
		err = repo.UpdateJob(ctx, job)
		if err == nil {
			t.Error("expected database constraint to prevent completed job without result")
		}

		// Clean up
		repo.DeleteJob(ctx, job.JobID)
	})

	t.Run("Database constraint prevents failed job without error", func(t *testing.T) {
		// Create a job directly in the database
		job := &models.ProcessingJob{
			JobID:        "test-job-2",
			SessionID:    "session-test",
			UserPhotoID:  "photo-123",
			ShirtImageID: "shirt-456",
			Status:       models.JobStatusPending,
			Progress:     0,
		}

		ctx := context.Background()
		err := repo.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Try to update to failed without error (bypassing service validation)
		job.Status = models.JobStatusFailed
		err = repo.UpdateJob(ctx, job)
		if err == nil {
			t.Error("expected database constraint to prevent failed job without error")
		}

		// Clean up
		repo.DeleteJob(ctx, job.JobID)
	})
}
