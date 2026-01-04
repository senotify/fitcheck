# Design Document

## Overview

The Job History feature adds persistent storage and list view capabilities to the Virtual FitCheck system. It replaces the in-memory job storage with PostgreSQL database, enabling users to view all their processing jobs, track progress, and access results even after server restarts. The feature includes session-based job tracking, filtering, sorting, and deletion capabilities.

## Architecture

### High-Level Architecture

```
┌─────────────────┐
│   Web Browser   │
│   (Frontend)    │
└────────┬────────┘
         │ HTTPS
         ▼
┌─────────────────┐
│   Backend API   │
│      (Go)       │
└────────┬────────┘
         │
         ├─────────────┬──────────────┐
         │             │              │
         ▼             ▼              ▼
┌──────────────┐  ┌──────────┐  ┌──────────────┐
│  PostgreSQL  │  │  File    │  │  AI Service  │
│   Database   │  │  Storage │  │              │
└──────────────┘  └──────────┘  └──────────────┘
```

### Technology Stack

**Backend:**

- PostgreSQL 14+ for persistent job storage
- `lib/pq` or `pgx` Go driver for database connectivity
- Database migrations using `golang-migrate` or similar
- Connection pooling for performance

**Frontend:**

- New JobHistory component to display job list
- Polling mechanism for real-time updates
- Filter and sort controls

## Components and Interfaces

### Database Schema

```sql
-- Jobs table
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) NOT NULL,
    user_photo_id VARCHAR(255) NOT NULL,
    shirt_image_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    status_message TEXT,
    result_id VARCHAR(255),
    result_token VARCHAR(255),
    error TEXT,
    email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,

    INDEX idx_session_id (session_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at DESC)
);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Backend API Endpoints

#### GET /api/jobs

List all jobs for the current session.

**Query Parameters:**

- `status` (optional): Filter by status (pending, processing, completed, failed)
- `sort` (optional): Sort order (newest, oldest)
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Pagination offset (default: 0)

**Response:**

```json
{
  "success": true,
  "jobs": [
    {
      "jobId": "uuid",
      "status": "completed",
      "progress": 100,
      "statusMessage": "Processing completed",
      "resultUrl": "/api/result/result-id",
      "thumbnailUrl": "/api/preview/user-photo-id",
      "createdAt": "2025-12-05T10:30:00Z",
      "completedAt": "2025-12-05T10:31:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

#### DELETE /api/jobs/:jobId

Delete a specific job and its associated files.

**Response:**

```json
{
  "success": true,
  "message": "Job deleted successfully"
}
```

### Frontend Components

#### JobHistoryView Component

Displays list of all jobs with filtering and sorting.

**Props:**

- `sessionId: string` - Current session identifier

**State:**

- `jobs: Job[]` - List of jobs
- `filter: 'all' | 'pending' | 'processing' | 'completed' | 'failed'`
- `sortOrder: 'newest' | 'oldest'`
- `loading: boolean`

**Methods:**

- `fetchJobs()` - Load jobs from API
- `applyFilter(status)` - Filter jobs by status
- `applySortOrder(order)` - Sort jobs by date
- `deleteJob(jobId)` - Delete a job
- `pollUpdates()` - Poll for job updates

#### JobCard Component

Displays individual job with status and actions.

**Props:**

- `job: Job` - Job data
- `onDelete: (jobId) => void` - Delete callback
- `onView: (jobId) => void` - View result callback

## Data Models

### Job (Updated)

```go
type ProcessingJob struct {
    JobID          string     `json:"jobId" db:"id"`
    SessionID      string     `json:"sessionId" db:"session_id"`
    UserPhotoID    string     `json:"userPhotoId" db:"user_photo_id"`
    ShirtImageID   string     `json:"shirtImageId" db:"shirt_image_id"`
    Status         JobStatus  `json:"status" db:"status"`
    Progress       int        `json:"progress" db:"progress"`
    StatusMessage  string     `json:"statusMessage" db:"status_message"`
    ResultID       *string    `json:"resultId,omitempty" db:"result_id"`
    ResultToken    *string    `json:"resultToken,omitempty" db:"result_token"`
    Error          *string    `json:"error,omitempty" db:"error"`
    Email          *string    `json:"email,omitempty" db:"email"`
    CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
    UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
    CompletedAt    *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}
```

### JobListResponse

```go
type JobListResponse struct {
    Success bool              `json:"success"`
    Jobs    []ProcessingJob   `json:"jobs"`
    Total   int               `json:"total"`
    Limit   int               `json:"limit"`
    Offset  int               `json:"offset"`
}
```

### JobFilter

```go
type JobFilter struct {
    SessionID string
    Status    *JobStatus
    SortOrder string // "newest" or "oldest"
    Limit     int
    Offset    int
}
```

## Correctness Properties

_A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees._

### Property 1: Session-based job retrieval

_For any_ session ID and set of jobs created with that session ID, querying jobs by that session ID should return exactly those jobs and no others.
**Validates: Requirements 1.1, 2.2**

### Property 2: Job data completeness

_For any_ job returned by the API, the response should include status, creation time, and thumbnail URL fields.
**Validates: Requirements 1.2**

### Property 3: Progress update consistency

_For any_ job with status "processing", polling the status endpoint should return progress values that are monotonically increasing (never decrease).
**Validates: Requirements 1.3**

### Property 4: Completed jobs have results

_For any_ job with status "completed", the job should have a non-null result URL.
**Validates: Requirements 1.4**

### Property 5: Failed jobs have error messages

_For any_ job with status "failed", the job should have a non-null error message.
**Validates: Requirements 1.5**

### Property 6: Database persistence

_For any_ job created through the API, querying the database directly should return a job with matching ID and data.
**Validates: Requirements 2.1**

### Property 7: Job completion updates database

_For any_ job that transitions to "completed" status, the database should be updated with the result ID and completion timestamp.
**Validates: Requirements 2.4**

### Property 8: Transaction consistency

_For any_ concurrent job creation operations, all jobs should be successfully stored without data corruption or lost updates.
**Validates: Requirements 3.2**

### Property 9: Status filtering

_For any_ status filter applied to job list query, all returned jobs should have that status, and no jobs with that status should be excluded.
**Validates: Requirements 4.1**

### Property 10: Date sorting

_For any_ job list sorted by date, the jobs should be ordered by creation timestamp in the specified direction (newest or oldest first).
**Validates: Requirements 4.2**

### Property 11: Filter reset

_For any_ job list query with filters cleared, the result should include all jobs for the session regardless of status.
**Validates: Requirements 4.4**

### Property 12: Job deletion completeness

_For any_ job deletion request, both the database record and associated files (user photo, shirt image, result) should be removed.
**Validates: Requirements 5.2**

### Property 13: List update after deletion

_For any_ job deletion, subsequent queries for the job list should not include the deleted job.
**Validates: Requirements 5.4**

## Error Handling

### Database Errors

**Connection Failures:**

- Retry connection with exponential backoff
- Return 503 Service Unavailable if database is down
- Log connection errors for monitoring

**Query Errors:**

- Return 500 Internal Server Error for unexpected query failures
- Return 404 Not Found for missing jobs
- Log query errors with context

**Transaction Errors:**

- Rollback transaction on any error
- Return appropriate error response
- Log transaction failures

### API Errors

**Invalid Session:**

- Return 401 Unauthorized if session is invalid
- Require session cookie or header

**Invalid Filters:**

- Return 400 Bad Request for invalid filter values
- Validate status values against enum

**Job Not Found:**

- Return 404 Not Found if job doesn't exist
- Verify job belongs to session before operations

## Testing Strategy

### Unit Testing

**Database Layer:**

- Test CRUD operations for jobs
- Test transaction handling
- Test connection pooling
- Test query builders and filters

**API Layer:**

- Test job list endpoint with various filters
- Test job deletion endpoint
- Test session validation
- Test error responses

### Property-Based Testing

Use `gopter` for Go backend property tests:

- Generate random jobs and verify persistence (Property 6)
- Generate random session IDs and verify isolation (Property 1)
- Generate random status values and verify filtering (Property 9)
- Generate random timestamps and verify sorting (Property 10)
- Test concurrent operations for consistency (Property 8)

### Integration Testing

**Database Integration:**

- Test with real PostgreSQL instance
- Test migrations up and down
- Test connection recovery
- Test transaction rollback

**End-to-End:**

- Create job → verify in list
- Filter jobs → verify correct results
- Delete job → verify removal
- Server restart → verify persistence

## Implementation Notes

### Database Connection

Use connection pooling with reasonable limits:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Session Management

Use secure HTTP-only cookies for session tracking:

- Generate session ID on first visit
- Store in cookie with 30-day expiration
- Use UUID v4 for session IDs

### Migration Strategy

1. Create migration files for schema
2. Run migrations on startup
3. Support rollback for failed migrations
4. Version migrations for tracking

### Performance Considerations

**Indexing:**

- Index on `session_id` for fast user queries
- Index on `status` for filtering
- Index on `created_at DESC` for sorting

**Caching:**

- Consider Redis cache for frequently accessed jobs
- Cache job lists for 5-10 seconds
- Invalidate cache on updates

**Pagination:**

- Default limit of 50 jobs per page
- Support offset-based pagination
- Consider cursor-based pagination for large datasets

### Security Considerations

**SQL Injection:**

- Use parameterized queries exclusively
- Never concatenate user input into SQL

**Session Security:**

- Use secure, HTTP-only cookies
- Regenerate session ID on sensitive operations
- Implement session timeout

**Authorization:**

- Verify job belongs to session before operations
- Don't expose other users' jobs
- Validate all input parameters

### Migration from In-Memory

1. Add PostgreSQL dependency
2. Create database schema
3. Implement database job service
4. Update API to use database service
5. Remove in-memory job service
6. Update tests to use database

## Database Setup

### Development

```bash
# Using Docker
docker run --name fitcheck-postgres \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=virtualfitcheck \
  -p 5432:5432 \
  -d postgres:14

# Run migrations
migrate -path ./migrations -database "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable" up
```

### Environment Variables

```env
DATABASE_URL=postgresql://user:password@localhost:5432/virtualfitcheck?sslmode=disable
DATABASE_MAX_CONNECTIONS=25
DATABASE_MAX_IDLE=5
```
