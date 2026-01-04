-- Script to test query performance and verify index usage
-- Run this with: psql <database_url> -f scripts/test_query_performance.sql

\echo 'Testing Query Performance and Index Usage'
\echo '=========================================='
\echo ''

-- Enable timing
\timing on

-- Test 1: Session-based query
\echo '=== Test 1: Get jobs by session ==='
EXPLAIN ANALYZE
SELECT 
    id, session_id, user_photo_id, shirt_image_id, status, 
    progress, status_message, result_id, result_token, error, 
    email, created_at, updated_at, completed_at
FROM jobs
WHERE session_id = (SELECT session_id FROM jobs LIMIT 1)
ORDER BY created_at DESC;

\echo ''
\echo '=== Test 2: Filtered query with status ==='
EXPLAIN ANALYZE
SELECT 
    id, session_id, user_photo_id, shirt_image_id, status, 
    progress, status_message, result_id, result_token, error, 
    email, created_at, updated_at, completed_at
FROM jobs
WHERE session_id = (SELECT session_id FROM jobs LIMIT 1)
  AND status = 'completed'
ORDER BY created_at DESC
LIMIT 10 OFFSET 0;

\echo ''
\echo '=== Test 3: Paginated query ==='
EXPLAIN ANALYZE
SELECT 
    id, session_id, user_photo_id, shirt_image_id, status, 
    progress, status_message, result_id, result_token, error, 
    email, created_at, updated_at, completed_at
FROM jobs
WHERE session_id = (SELECT session_id FROM jobs LIMIT 1)
ORDER BY created_at DESC
LIMIT 50 OFFSET 0;

\echo ''
\echo '=== Test 4: Count query for pagination ==='
EXPLAIN ANALYZE
SELECT COUNT(*) 
FROM jobs
WHERE session_id = (SELECT session_id FROM jobs LIMIT 1);

\echo ''
\echo '=== Test 5: Token lookup ==='
EXPLAIN ANALYZE
SELECT 
    id, session_id, user_photo_id, shirt_image_id, status, 
    progress, status_message, result_id, result_token, error, 
    email, created_at, updated_at, completed_at
FROM jobs
WHERE result_token = (SELECT result_token FROM jobs WHERE result_token IS NOT NULL LIMIT 1);

\echo ''
\echo '=== Performance Summary ==='
\echo 'Check the query plans above for:'
\echo '1. "Index Scan" or "Index Only Scan" (good) vs "Seq Scan" (bad)'
\echo '2. Execution time should be < 10ms for typical queries'
\echo '3. "Rows Removed by Filter" should be 0 or low'
\echo ''

\timing off
