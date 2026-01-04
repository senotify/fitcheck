package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"virtual-fitcheck/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// Note: This requires a running PostgreSQL instance for integration tests
func setupTestDB(t *testing.T) *sql.DB {
	// Use docker-compose database credentials
	dbURL := "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable"
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not reachable: %v", err)
		return nil
	}

	// Clean up test data
	_, _ = db.Exec("DELETE FROM jobs WHERE session_id LIKE 'test-%'")

	return db
}

func TestPostgresJobRepository_CreateJob(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	job := &models.ProcessingJob{
		SessionID:     "test-session-1",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)
	assert.NotEmpty(t, job.JobID)
	assert.False(t, job.CreatedAt.IsZero())
	assert.False(t, job.UpdatedAt.IsZero())

	// Clean up
	_ = repo.DeleteJob(ctx, job.JobID)
}

func TestPostgresJobRepository_GetJob(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create a job first
	job := &models.ProcessingJob{
		SessionID:     "test-session-2",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Retrieve the job
	retrieved, err := repo.GetJob(ctx, job.JobID)
	require.NoError(t, err)
	assert.Equal(t, job.JobID, retrieved.JobID)
	assert.Equal(t, job.SessionID, retrieved.SessionID)
	assert.Equal(t, job.UserPhotoID, retrieved.UserPhotoID)
	assert.Equal(t, job.ShirtImageID, retrieved.ShirtImageID)
	assert.Equal(t, job.Status, retrieved.Status)

	// Clean up
	_ = repo.DeleteJob(ctx, job.JobID)
}

func TestPostgresJobRepository_GetJobsBySession(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	sessionID := "test-session-3"

	// Create multiple jobs
	job1 := &models.ProcessingJob{
		SessionID:     sessionID,
		UserPhotoID:   "photo-1",
		ShirtImageID:  "shirt-1",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job 1",
	}

	job2 := &models.ProcessingJob{
		SessionID:     sessionID,
		UserPhotoID:   "photo-2",
		ShirtImageID:  "shirt-2",
		Status:        models.JobStatusProcessing,
		Progress:      50,
		StatusMessage: "Job 2",
	}

	err := repo.CreateJob(ctx, job1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = repo.CreateJob(ctx, job2)
	require.NoError(t, err)

	// Retrieve jobs by session
	jobs, err := repo.GetJobsBySession(ctx, sessionID)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Should be ordered by created_at DESC (newest first)
	assert.Equal(t, job2.JobID, jobs[0].JobID)
	assert.Equal(t, job1.JobID, jobs[1].JobID)

	// Clean up
	_ = repo.DeleteJob(ctx, job1.JobID)
	_ = repo.DeleteJob(ctx, job2.JobID)
}

func TestPostgresJobRepository_UpdateJob(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create a job
	job := &models.ProcessingJob{
		SessionID:     "test-session-4",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Update the job
	job.Status = models.JobStatusCompleted
	job.Progress = 100
	job.StatusMessage = "Job completed"
	resultID := "result-789"
	job.ResultID = &resultID
	now := time.Now()
	job.CompletedAt = &now

	err = repo.UpdateJob(ctx, job)
	require.NoError(t, err)

	// Retrieve and verify
	updated, err := repo.GetJob(ctx, job.JobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, updated.Status)
	assert.Equal(t, 100, updated.Progress)
	assert.Equal(t, "Job completed", updated.StatusMessage)
	assert.NotNil(t, updated.ResultID)
	assert.Equal(t, "result-789", *updated.ResultID)
	assert.NotNil(t, updated.CompletedAt)

	// Clean up
	_ = repo.DeleteJob(ctx, job.JobID)
}

func TestPostgresJobRepository_DeleteJob(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create a job
	job := &models.ProcessingJob{
		SessionID:     "test-session-5",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Delete the job
	err = repo.DeleteJob(ctx, job.JobID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetJob(ctx, job.JobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestPostgresJobRepository_GetJobByToken(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create a job with a token
	token := "test-token-123"
	resultID := "result-789"
	job := &models.ProcessingJob{
		SessionID:     "test-session-6",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusCompleted,
		Progress:      100,
		StatusMessage: "Job completed",
		ResultID:      &resultID,
		ResultToken:   &token,
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Retrieve by token
	retrieved, err := repo.GetJobByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, job.JobID, retrieved.JobID)
	assert.NotNil(t, retrieved.ResultToken)
	assert.Equal(t, token, *retrieved.ResultToken)

	// Clean up
	_ = repo.DeleteJob(ctx, job.JobID)
}

func TestPostgresJobRepository_Transaction(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Begin transaction
	txRepo, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	// Create a job within transaction
	job := &models.ProcessingJob{
		SessionID:     "test-session-7",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err = txRepo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Job should be visible within transaction
	retrieved, err := txRepo.GetJob(ctx, job.JobID)
	require.NoError(t, err)
	assert.Equal(t, job.JobID, retrieved.JobID)

	// Rollback transaction
	err = txRepo.Rollback()
	require.NoError(t, err)

	// Job should not exist after rollback
	_, err = repo.GetJob(ctx, job.JobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestPostgresJobRepository_TransactionCommit(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Begin transaction
	txRepo, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	// Create a job within transaction
	job := &models.ProcessingJob{
		SessionID:     "test-session-8",
		UserPhotoID:   "photo-123",
		ShirtImageID:  "shirt-456",
		Status:        models.JobStatusPending,
		Progress:      0,
		StatusMessage: "Job created",
	}

	err = txRepo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Commit transaction
	err = txRepo.Commit()
	require.NoError(t, err)

	// Job should exist after commit
	retrieved, err := repo.GetJob(ctx, job.JobID)
	require.NoError(t, err)
	assert.Equal(t, job.JobID, retrieved.JobID)

	// Clean up
	_ = repo.DeleteJob(ctx, job.JobID)
}

// Unit tests that don't require database connection

func TestPostgresJobRepository_CreateJob_Validation(t *testing.T) {
	// Create a mock DB (won't be used for validation tests)
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("nil job", func(t *testing.T) {
		err := repo.CreateJob(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job cannot be nil")
	})
}

func TestPostgresJobRepository_GetJob_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("empty jobID", func(t *testing.T) {
		_, err := repo.GetJob(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "jobID cannot be empty")
	})
}

func TestPostgresJobRepository_GetJobsBySession_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("empty sessionID", func(t *testing.T) {
		_, err := repo.GetJobsBySession(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sessionID cannot be empty")
	})
}

func TestPostgresJobRepository_UpdateJob_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("nil job", func(t *testing.T) {
		err := repo.UpdateJob(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job cannot be nil")
	})

	t.Run("empty jobID", func(t *testing.T) {
		job := &models.ProcessingJob{}
		err := repo.UpdateJob(ctx, job)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "jobID cannot be empty")
	})
}

func TestPostgresJobRepository_DeleteJob_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("empty jobID", func(t *testing.T) {
		err := repo.DeleteJob(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "jobID cannot be empty")
	})
}

func TestPostgresJobRepository_GetJobByToken_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("empty token", func(t *testing.T) {
		_, err := repo.GetJobByToken(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cannot be empty")
	})
}

func TestNewPostgresJobRepository(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.Nil(t, repo.tx)
}
