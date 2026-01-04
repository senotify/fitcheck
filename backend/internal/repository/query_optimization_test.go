package repository

import (
	"context"
	"testing"
	"time"
	"virtual-fitcheck/internal/models"

	"github.com/google/uuid"
)

// TestQueryIndexUsage verifies that our queries are using the appropriate indexes
func TestQueryIndexUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Setup test database
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	analyzer := NewQueryAnalyzer(db)
	ctx := context.Background()

	// Create test data
	sessionID := uuid.New().String()
	for i := 0; i < 10; i++ {
		job := &models.ProcessingJob{
			JobID:        uuid.New().String(),
			SessionID:    sessionID,
			UserPhotoID:  uuid.New().String(),
			ShirtImageID: uuid.New().String(),
			Status:       models.JobStatusPending,
			Progress:     0,
			CreatedAt:    time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt:    time.Now(),
		}
		if err := repo.CreateJob(ctx, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	t.Run("GetJobsBySession uses session_id index", func(t *testing.T) {
		query := `
			SELECT 
				id, session_id, user_photo_id, shirt_image_id, status, 
				progress, status_message, result_id, result_token, error, 
				email, created_at, updated_at, completed_at
			FROM jobs
			WHERE session_id = $1
			ORDER BY created_at DESC
		`

		expectedIndexes := []string{"idx_jobs_session_id", "idx_jobs_created_at"}
		usesIndex, plan, err := analyzer.VerifyIndexUsage(ctx, query, expectedIndexes, sessionID)
		if err != nil {
			t.Fatalf("Failed to analyze query: %v", err)
		}

		t.Logf("Query plan:\n%s", plan)

		if !usesIndex {
			t.Errorf("Query is not using expected indexes. Expected one of: %v", expectedIndexes)
		}
	})

	t.Run("GetJobsWithFilter uses indexes efficiently", func(t *testing.T) {
		status := string(models.JobStatusPending)
		filter := JobFilter{
			SessionID: sessionID,
			Status:    &status,
			SortOrder: "newest",
			Limit:     10,
			Offset:    0,
		}

		query := `
			SELECT 
				id, session_id, user_photo_id, shirt_image_id, status, 
				progress, status_message, result_id, result_token, error, 
				email, created_at, updated_at, completed_at
			FROM jobs
			WHERE session_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`

		expectedIndexes := []string{"idx_jobs_session_id", "idx_jobs_status", "idx_jobs_created_at"}
		usesIndex, plan, err := analyzer.VerifyIndexUsage(ctx, query, expectedIndexes, sessionID, status, filter.Limit, filter.Offset)
		if err != nil {
			t.Fatalf("Failed to analyze query: %v", err)
		}

		t.Logf("Query plan:\n%s", plan)

		if !usesIndex {
			t.Errorf("Query is not using expected indexes. Expected one of: %v", expectedIndexes)
		}
	})

	t.Run("GetJobByToken uses result_token index", func(t *testing.T) {
		// Create a job with a result token
		token := uuid.New().String()
		resultID := uuid.New().String()
		job := &models.ProcessingJob{
			JobID:        uuid.New().String(),
			SessionID:    sessionID,
			UserPhotoID:  uuid.New().String(),
			ShirtImageID: uuid.New().String(),
			Status:       models.JobStatusCompleted,
			Progress:     100,
			ResultToken:  &token,
			ResultID:     &resultID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := repo.CreateJob(ctx, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}

		query := `
			SELECT 
				id, session_id, user_photo_id, shirt_image_id, status, 
				progress, status_message, result_id, result_token, error, 
				email, created_at, updated_at, completed_at
			FROM jobs
			WHERE result_token = $1
		`

		expectedIndexes := []string{"idx_jobs_result_token"}
		usesIndex, plan, err := analyzer.VerifyIndexUsage(ctx, query, expectedIndexes, token)
		if err != nil {
			t.Fatalf("Failed to analyze query: %v", err)
		}

		t.Logf("Query plan:\n%s", plan)

		if !usesIndex {
			t.Errorf("Query is not using expected index: idx_jobs_result_token")
		}
	})
}

// TestPaginationPerformance tests that pagination queries perform efficiently
func TestPaginationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Setup test database
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create a larger dataset
	sessionID := uuid.New().String()
	for i := 0; i < 100; i++ {
		job := &models.ProcessingJob{
			JobID:        uuid.New().String(),
			SessionID:    sessionID,
			UserPhotoID:  uuid.New().String(),
			ShirtImageID: uuid.New().String(),
			Status:       models.JobStatusPending,
			Progress:     0,
			CreatedAt:    time.Now().Add(-time.Duration(i) * time.Minute),
			UpdatedAt:    time.Now(),
		}
		if err := repo.CreateJob(ctx, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	t.Run("Paginated query returns correct results", func(t *testing.T) {
		filter := JobFilter{
			SessionID: sessionID,
			SortOrder: "newest",
			Limit:     10,
			Offset:    0,
		}

		result, err := repo.GetJobsWithFilter(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to get jobs with filter: %v", err)
		}

		if len(result.Jobs) != 10 {
			t.Errorf("Expected 10 jobs, got %d", len(result.Jobs))
		}

		if result.Total != 100 {
			t.Errorf("Expected total of 100, got %d", result.Total)
		}

		if result.Limit != 10 {
			t.Errorf("Expected limit of 10, got %d", result.Limit)
		}

		if result.Offset != 0 {
			t.Errorf("Expected offset of 0, got %d", result.Offset)
		}
	})

	t.Run("Second page returns different results", func(t *testing.T) {
		// Get first page
		filter1 := JobFilter{
			SessionID: sessionID,
			SortOrder: "newest",
			Limit:     10,
			Offset:    0,
		}
		result1, err := repo.GetJobsWithFilter(ctx, filter1)
		if err != nil {
			t.Fatalf("Failed to get first page: %v", err)
		}

		// Get second page
		filter2 := JobFilter{
			SessionID: sessionID,
			SortOrder: "newest",
			Limit:     10,
			Offset:    10,
		}
		result2, err := repo.GetJobsWithFilter(ctx, filter2)
		if err != nil {
			t.Fatalf("Failed to get second page: %v", err)
		}

		// Verify pages are different
		if result1.Jobs[0].JobID == result2.Jobs[0].JobID {
			t.Error("First job on page 1 and page 2 should be different")
		}

		// Verify both pages have correct metadata
		if result1.Total != result2.Total {
			t.Errorf("Total count should be same for both pages: %d vs %d", result1.Total, result2.Total)
		}
	})

	t.Run("Status filter works correctly", func(t *testing.T) {
		// Update some jobs to completed status
		jobs, _ := repo.GetJobsBySession(ctx, sessionID)
		for i := 0; i < 20; i++ {
			jobs[i].Status = models.JobStatusCompleted
			resultID := uuid.New().String()
			jobs[i].ResultID = &resultID
			repo.UpdateJob(ctx, jobs[i])
		}

		// Query for completed jobs only
		status := string(models.JobStatusCompleted)
		filter := JobFilter{
			SessionID: sessionID,
			Status:    &status,
			SortOrder: "newest",
			Limit:     50,
			Offset:    0,
		}

		result, err := repo.GetJobsWithFilter(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to get filtered jobs: %v", err)
		}

		if result.Total != 20 {
			t.Errorf("Expected 20 completed jobs, got %d", result.Total)
		}

		// Verify all returned jobs have completed status
		for _, job := range result.Jobs {
			if job.Status != models.JobStatusCompleted {
				t.Errorf("Expected all jobs to be completed, got %s", job.Status)
			}
		}
	})
}

// TestSortingPerformance tests that sorting queries perform efficiently
func TestSortingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Setup test database
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	// Create test data with specific timestamps
	sessionID := uuid.New().String()
	var createdTimes []time.Time
	for i := 0; i < 10; i++ {
		createdAt := time.Now().Add(-time.Duration(i) * time.Hour)
		createdTimes = append(createdTimes, createdAt)
		job := &models.ProcessingJob{
			JobID:        uuid.New().String(),
			SessionID:    sessionID,
			UserPhotoID:  uuid.New().String(),
			ShirtImageID: uuid.New().String(),
			Status:       models.JobStatusPending,
			Progress:     0,
			CreatedAt:    createdAt,
			UpdatedAt:    time.Now(),
		}
		if err := repo.CreateJob(ctx, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	t.Run("Newest first sorting", func(t *testing.T) {
		filter := JobFilter{
			SessionID: sessionID,
			SortOrder: "newest",
			Limit:     10,
			Offset:    0,
		}

		result, err := repo.GetJobsWithFilter(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to get jobs: %v", err)
		}

		// Verify jobs are sorted newest first
		for i := 0; i < len(result.Jobs)-1; i++ {
			if result.Jobs[i].CreatedAt.Before(result.Jobs[i+1].CreatedAt) {
				t.Errorf("Jobs not sorted newest first at index %d", i)
			}
		}
	})

	t.Run("Oldest first sorting", func(t *testing.T) {
		filter := JobFilter{
			SessionID: sessionID,
			SortOrder: "oldest",
			Limit:     10,
			Offset:    0,
		}

		result, err := repo.GetJobsWithFilter(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to get jobs: %v", err)
		}

		// Verify jobs are sorted oldest first
		for i := 0; i < len(result.Jobs)-1; i++ {
			if result.Jobs[i].CreatedAt.After(result.Jobs[i+1].CreatedAt) {
				t.Errorf("Jobs not sorted oldest first at index %d", i)
			}
		}
	})
}
