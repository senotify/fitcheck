# Implementation Plan

- [x] 1. Set up PostgreSQL database and migrations

  - Install PostgreSQL locally or use Docker
  - Add database driver dependency (`github.com/lib/pq` or `github.com/jackc/pgx`)
  - Add migration tool (`golang-migrate/migrate`)
  - Create database connection configuration
  - _Requirements: 2.1, 2.3, 3.1_

- [x] 1.1 Create database schema migration

  - Create migration file for jobs table
  - Add indexes for session_id, status, and created_at
  - Add trigger for updated_at timestamp
  - Test migration up and down
  - _Requirements: 2.1, 3.3_

- [x] 2. Implement database job repository

  - Create `JobRepository` interface with CRUD methods
  - Implement PostgreSQL repository with connection pooling
  - Add methods: CreateJob, GetJob, GetJobsBySession, UpdateJob, DeleteJob
  - Implement transaction support for consistency
  - _Requirements: 2.1, 2.4, 3.2_

- [ ]\* 2.1 Write property test for database persistence

  - **Property 6: Database persistence**
  - **Validates: Requirements 2.1**

- [ ]\* 2.2 Write property test for transaction consistency

  - **Property 8: Transaction consistency**
  - **Validates: Requirements 3.2**

- [x] 3. Add session management

  - Create session middleware for generating/reading session IDs
  - Use secure HTTP-only cookies for session storage
  - Generate UUID v4 for session identifiers
  - Set 30-day cookie expiration
  - _Requirements: 1.1, 2.2_

- [ ]\* 3.1 Write property test for session-based retrieval

  - **Property 1: Session-based job retrieval**
  - **Validates: Requirements 1.1, 2.2**

- [x] 4. Update job service to use database

  - Replace in-memory storage with database repository
  - Update CreateJob to store in database with session ID
  - Update GetJob to query from database
  - Update job status/progress methods to persist changes
  - Ensure backward compatibility with existing code
  - _Requirements: 2.1, 2.3, 2.4_

- [ ]\* 4.1 Write property test for job completion updates

  - **Property 7: Job completion updates database**
  - **Validates: Requirements 2.4**

- [x] 5. Implement job list API endpoint

  - Create GET /api/jobs endpoint
  - Add query parameters for filtering (status) and sorting (date)
  - Add pagination support (limit, offset)
  - Return job list with total count
  - Include thumbnail URLs in response
  - _Requirements: 1.1, 1.2, 4.1, 4.2_

- [ ]\* 5.1 Write property test for job data completeness

  - **Property 2: Job data completeness**
  - **Validates: Requirements 1.2**

- [ ]\* 5.2 Write property test for status filtering

  - **Property 9: Status filtering**
  - **Validates: Requirements 4.1**

- [ ]\* 5.3 Write property test for date sorting

  - **Property 10: Date sorting**
  - **Validates: Requirements 4.2**

- [ ]\* 5.4 Write property test for filter reset

  - **Property 11: Filter reset**
  - **Validates: Requirements 4.4**

- [x] 6. Implement job deletion API endpoint

  - Create DELETE /api/jobs/:jobId endpoint
  - Verify job belongs to current session
  - Delete job from database
  - Delete associated files (user photo, shirt image, result)
  - Return success response
  - _Requirements: 5.2, 5.4_

- [ ]\* 6.1 Write property test for job deletion completeness

  - **Property 12: Job deletion completeness**
  - **Validates: Requirements 5.2**

- [ ]\* 6.2 Write property test for list update after deletion

  - **Property 13: List update after deletion**
  - **Validates: Requirements 5.4**

- [x] 7. Add validation for completed and failed jobs

  - Update job completion logic to ensure result URL is set
  - Update job failure logic to ensure error message is set
  - Add database constraints if needed
  - _Requirements: 1.4, 1.5_

- [ ]\* 7.1 Write property test for completed jobs

  - **Property 4: Completed jobs have results**
  - **Validates: Requirements 1.4**

- [ ]\* 7.2 Write property test for failed jobs

  - **Property 5: Failed jobs have error messages**
  - **Validates: Requirements 1.5**

- [x] 8. Implement JobHistoryView frontend component

  - Create JobHistoryView component with job list display
  - Add filter controls (all, pending, processing, completed, failed)
  - Add sort controls (newest first, oldest first)
  - Implement polling for real-time updates (every 5 seconds)
  - Display loading states
  - _Requirements: 1.1, 1.3, 4.1, 4.2, 4.3_

- [ ]\* 8.1 Write property test for progress updates

  - **Property 3: Progress update consistency**
  - **Validates: Requirements 1.3**

- [x] 9. Implement JobCard component

  - Create JobCard component to display individual job
  - Show job status with visual indicator
  - Show creation time and progress
  - Show thumbnail preview of user photo
  - Add "View Result" button for completed jobs
  - Add "Delete" button with confirmation dialog
  - _Requirements: 1.2, 1.4, 5.1, 5.3_

- [x] 10. Add navigation to job history page

  - Add "My Jobs" or "History" link to main navigation
  - Create route for job history page (/jobs or /history)
  - Update App component with new route
  - _Requirements: 1.1_

- [x] 11. Update configuration and environment variables

  - Add DATABASE_URL to .env and .env.example
  - Add database connection pool settings
  - Update config.go to load database configuration
  - Document database setup in README
  - _Requirements: 3.1_

- [x] 12. Create database setup documentation

  - Document PostgreSQL installation (local and Docker)
  - Document migration commands
  - Add troubleshooting section
  - Update README with database setup instructions
  - _Requirements: 3.1_

- [x] 13. Checkpoint - Test database integration

  - Ensure all tests pass, ask the user if questions arise
  - Test job creation and retrieval from database
  - Test filtering and sorting
  - Test job deletion
  - Verify session isolation

- [x] 14. Handle database migration on startup

  - Add automatic migration on server startup
  - Handle migration errors gracefully
  - Log migration status
  - _Requirements: 3.1, 3.4_

- [x] 15. Add error handling for database unavailability

  - Return 503 Service Unavailable when database is down
  - Log database errors
  - Implement connection retry logic
  - Display user-friendly error messages
  - _Requirements: 3.4_

- [x] 16. Optimize database queries

  - Verify indexes are being used (EXPLAIN queries)
  - Add query result caching if needed
  - Optimize pagination queries
  - _Requirements: 3.3_

- [x] 17. Final checkpoint - Test complete job history flow

  - Ensure all tests pass, ask the user if questions arise
  - Test complete flow: create jobs → view list → filter → sort → delete
  - Test persistence across server restart
  - Test with multiple sessions
