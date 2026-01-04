# Database Error Handling

This document describes the error handling mechanisms implemented for database unavailability.

## Features

### 1. Connection Retry Logic

The database connection now includes automatic retry logic with exponential backoff:

- **Maximum Attempts**: 3 attempts
- **Initial Delay**: 2 seconds
- **Backoff Strategy**: Exponential (2x multiplier)
- **Location**: `backend/internal/repository/database.go`

When the application starts, it will attempt to connect to the database up to 3 times before failing. This helps handle temporary network issues or database startup delays.

### 2. Database Health Check Middleware

A middleware function checks database connectivity before processing API requests:

- **Endpoint**: Applied to all `/api/*` routes
- **Response Code**: 503 Service Unavailable
- **Error Code**: `DATABASE_UNAVAILABLE`
- **Location**: `backend/internal/api/server.go`

The middleware performs a quick health check (2-second timeout) on each request to ensure the database is available. If the database is down, it returns a user-friendly error message.

### 3. Enhanced Error Logging

All database operations now include detailed error logging:

- **Log Level**: ERROR
- **Context**: Includes operation type and relevant IDs
- **Format**: `ERROR: Failed to [operation] [entity] [id]: [error details]`
- **Location**: `backend/internal/repository/postgres_job_repository.go`

Examples:

```
ERROR: Failed to create job 85febe27-5f41-4240-b0bb-9ee034d47ff4 in database: pq: relation "jobs" does not exist
ERROR: Failed to get job non-existent-job-id from database: pq: relation "jobs" does not exist
ERROR: Failed to update job abc123 in database: connection refused
```

### 4. User-Friendly Error Messages

When the database is unavailable, users receive clear, actionable error messages:

```json
{
  "success": false,
  "error": {
    "code": "DATABASE_UNAVAILABLE",
    "message": "The database service is currently unavailable. Please try again later.",
    "details": "Unable to connect to database"
  }
}
```

## Testing

### Manual Testing

To test database unavailability handling:

1. Stop the PostgreSQL database
2. Start the backend server
3. Make any API request
4. Verify you receive a 503 response with appropriate error message

### Automated Testing

Run the database health check test:

```bash
cd backend
go test ./internal/api/database_health_test.go ./internal/api/server.go ./internal/api/session.go -v
```

## Configuration

No additional configuration is required. The error handling is automatically enabled when the database is configured via the `DATABASE_URL` environment variable.

## Monitoring

Monitor the application logs for database-related errors:

```bash
# Look for ERROR level logs
grep "ERROR:" application.log

# Specifically for database errors
grep "ERROR: Failed to" application.log
grep "Database health check failed" application.log
```

## Recovery

When the database becomes available again:

1. The connection pool will automatically reconnect on the next request
2. No application restart is required
3. Users can retry their requests immediately

## Requirements Satisfied

This implementation satisfies the following requirements from the job-history spec:

- **Requirement 3.4**: Return 503 Service Unavailable when database is down ✓
- **Requirement 3.4**: Implement connection retry logic ✓
- **Requirement 3.4**: Log database errors ✓
- **Requirement 3.4**: Display user-friendly error messages ✓
