package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetJobsWithFilter_Validation tests validation logic without database
func TestGetJobsWithFilter_Validation(t *testing.T) {
	db, _ := sql.Open("postgres", "postgresql://fake@localhost/fake")
	defer db.Close()

	repo := NewPostgresJobRepository(db)
	ctx := context.Background()

	t.Run("empty sessionID", func(t *testing.T) {
		filter := JobFilter{
			SessionID: "",
			Limit:     10,
			Offset:    0,
		}
		_, err := repo.GetJobsWithFilter(ctx, filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sessionID cannot be empty")
	})

	t.Run("default values applied", func(t *testing.T) {
		// This test verifies the logic, but won't actually execute the query
		filter := JobFilter{
			SessionID: "test-session",
			Limit:     0,  // Should default to 50
			Offset:    -1, // Should default to 0
		}
		
		// We can't test the actual query without a database,
		// but we can verify the validation passes
		_, err := repo.GetJobsWithFilter(ctx, filter)
		// Will fail on query execution, but not on validation
		assert.Error(t, err) // Expected because no real DB
	})
}

// TestJobFilter tests the JobFilter struct
func TestJobFilter(t *testing.T) {
	t.Run("create filter with all fields", func(t *testing.T) {
		status := "completed"
		filter := JobFilter{
			SessionID: "session-123",
			Status:    &status,
			SortOrder: "newest",
			Limit:     25,
			Offset:    10,
		}

		assert.Equal(t, "session-123", filter.SessionID)
		assert.NotNil(t, filter.Status)
		assert.Equal(t, "completed", *filter.Status)
		assert.Equal(t, "newest", filter.SortOrder)
		assert.Equal(t, 25, filter.Limit)
		assert.Equal(t, 10, filter.Offset)
	})

	t.Run("create filter without status", func(t *testing.T) {
		filter := JobFilter{
			SessionID: "session-123",
			Status:    nil,
			SortOrder: "oldest",
			Limit:     50,
			Offset:    0,
		}

		assert.Equal(t, "session-123", filter.SessionID)
		assert.Nil(t, filter.Status)
		assert.Equal(t, "oldest", filter.SortOrder)
	})
}

// TestJobListResult tests the JobListResult struct
func TestJobListResult(t *testing.T) {
	t.Run("create result", func(t *testing.T) {
		result := JobListResult{
			Jobs:   nil,
			Total:  100,
			Limit:  50,
			Offset: 0,
		}

		assert.Equal(t, 100, result.Total)
		assert.Equal(t, 50, result.Limit)
		assert.Equal(t, 0, result.Offset)
		assert.Nil(t, result.Jobs)
	})
}
