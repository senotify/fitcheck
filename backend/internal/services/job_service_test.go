package services

import (
	"testing"

	"virtual-fitcheck/internal/models"
)

func TestJobService_CreateJob(t *testing.T) {
	js := NewJobService()

	t.Run("creates job with valid inputs", func(t *testing.T) {
		job, err := js.CreateJob("user-photo-123", "shirt-456")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if job.JobID == "" {
			t.Error("expected job ID to be set")
		}
		if job.UserPhotoID != "user-photo-123" {
			t.Errorf("expected UserPhotoID to be 'user-photo-123', got %s", job.UserPhotoID)
		}
		if job.ShirtImageID != "shirt-456" {
			t.Errorf("expected ShirtImageID to be 'shirt-456', got %s", job.ShirtImageID)
		}
		if job.Status != models.JobStatusPending {
			t.Errorf("expected status to be pending, got %s", job.Status)
		}
		if job.Progress != 0 {
			t.Errorf("expected progress to be 0, got %d", job.Progress)
		}
	})

	t.Run("returns error for empty userPhotoID", func(t *testing.T) {
		_, err := js.CreateJob("", "shirt-456")
		if err == nil {
			t.Error("expected error for empty userPhotoID")
		}
	})

	t.Run("returns error for empty shirtImageID", func(t *testing.T) {
		_, err := js.CreateJob("user-photo-123", "")
		if err == nil {
			t.Error("expected error for empty shirtImageID")
		}
	})
}

func TestJobService_GetJob(t *testing.T) {
	js := NewJobService()

	t.Run("retrieves existing job", func(t *testing.T) {
		created, _ := js.CreateJob("user-photo-123", "shirt-456")

		retrieved, err := js.GetJob(created.JobID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if retrieved.JobID != created.JobID {
			t.Errorf("expected JobID %s, got %s", created.JobID, retrieved.JobID)
		}
	})

	t.Run("returns error for non-existent job", func(t *testing.T) {
		_, err := js.GetJob("non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent job")
		}
	})
} 