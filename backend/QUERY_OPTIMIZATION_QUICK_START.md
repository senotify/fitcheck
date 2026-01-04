# Query Optimization - Quick Start Guide

## TL;DR

Database queries are now optimized with indexes and efficient pagination. Use the new `GetJobsWithFilter()` method for filtered, sorted, and paginated queries.

## Quick Example

```go
// Get jobs with filtering, sorting, and pagination
status := "completed"
filter := repository.JobFilter{
    SessionID: sessionID,
    Status:    &status,      // Optional: filter by status
    SortOrder: "newest",     // "newest" or "oldest"
    Limit:     50,           // Results per page
    Offset:    0,            // Page offset
}

result, err := repo.GetJobsWithFilter(ctx, filter)
if err != nil {
    return err
}

// Access results
for _, job := range result.Jobs {
    fmt.Printf("Job %s: %s\n", job.JobID, job.Status)
}

// Pagination info
fmt.Printf("Showing %d of %d total jobs\n", len(result.Jobs), result.Total)
```

## What's Optimized

✅ Session-based queries use `idx_jobs_session_id` index  
✅ Status filtering uses `idx_jobs_status` index  
✅ Date sorting uses `idx_jobs_created_at` index  
✅ Token lookups use `idx_jobs_result_token` index  
✅ Pagination with LIMIT/OFFSET  
✅ Total count for pagination metadata

## Performance

| Query Type      | Expected Time |
| --------------- | ------------- |
| Get by session  | < 5ms         |
| Filtered query  | < 15ms        |
| Paginated query | < 20ms        |
| Token lookup    | < 5ms         |

## Verify Indexes

```bash
# Check indexes exist
psql "$DATABASE_URL" -f scripts/verify_indexes.sql

# Test performance
psql "$DATABASE_URL" -f scripts/test_query_performance.sql
```

## Migration Required

Apply migration 000003 to add the result_token index:

```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

Or manually:

```sql
CREATE INDEX IF NOT EXISTS idx_jobs_result_token
ON jobs(result_token)
WHERE result_token IS NOT NULL;
```

## API Integration Example

```go
// In your API handler
func (s *Server) handleGetJobs(c *gin.Context) {
    sessionID := GetSessionID(c)

    // Parse query parameters
    status := c.Query("status")
    sortOrder := c.DefaultQuery("sort", "newest")
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

    // Build filter
    filter := repository.JobFilter{
        SessionID: sessionID,
        SortOrder: sortOrder,
        Limit:     limit,
        Offset:    offset,
    }

    if status != "" {
        filter.Status = &status
    }

    // Execute query
    result, err := s.jobRepo.GetJobsWithFilter(c.Request.Context(), filter)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Return response
    c.JSON(200, gin.H{
        "success": true,
        "jobs":    result.Jobs,
        "total":   result.Total,
        "limit":   result.Limit,
        "offset":  result.Offset,
    })
}
```

## Monitoring

Use QueryAnalyzer to debug slow queries:

```go
analyzer := repository.NewQueryAnalyzer(db)

// Log query performance
analyzer.LogQueryPerformance(ctx, "MyQuery", query, args...)

// Verify index usage
usesIndex, plan, err := analyzer.VerifyIndexUsage(
    ctx, query, []string{"idx_jobs_session_id"}, args...,
)
```

## Common Patterns

### Get all jobs for a session (newest first)

```go
filter := repository.JobFilter{
    SessionID: sessionID,
    SortOrder: "newest",
    Limit:     50,
    Offset:    0,
}
```

### Get only completed jobs

```go
status := "completed"
filter := repository.JobFilter{
    SessionID: sessionID,
    Status:    &status,
    SortOrder: "newest",
    Limit:     50,
    Offset:    0,
}
```

### Get second page of results

```go
filter := repository.JobFilter{
    SessionID: sessionID,
    SortOrder: "newest",
    Limit:     50,
    Offset:    50,  // Skip first 50
}
```

### Get oldest jobs first

```go
filter := repository.JobFilter{
    SessionID: sessionID,
    SortOrder: "oldest",
    Limit:     50,
    Offset:    0,
}
```

## Testing

```bash
# Run validation tests (no DB required)
go test ./internal/repository -run Validation -v

# Run performance tests (requires DB)
go test ./internal/repository -run TestQueryIndexUsage -v
go test ./internal/repository -run TestPaginationPerformance -v
```

## Documentation

- **QUERY_OPTIMIZATION.md** - Complete guide
- **OPTIMIZATION_SUMMARY.md** - Implementation details
- **TASK_16_COMPLETION.md** - Completion report

## Need Help?

Check the EXPLAIN output to see if indexes are being used:

```sql
EXPLAIN ANALYZE
SELECT * FROM jobs
WHERE session_id = 'your-session-id'
ORDER BY created_at DESC;
```

Look for "Index Scan" (good) vs "Seq Scan" (bad).
