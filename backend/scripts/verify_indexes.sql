-- Script to verify database indexes are properly configured
-- Run this with: psql <database_url> -f scripts/verify_indexes.sql

\echo 'Checking database indexes for jobs table...'
\echo ''

-- List all indexes on the jobs table
\echo '=== Current Indexes on jobs table ==='
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'jobs'
ORDER BY indexname;

\echo ''
\echo '=== Expected Indexes ==='
\echo '1. idx_jobs_session_id - for session-based queries'
\echo '2. idx_jobs_status - for status filtering'
\echo '3. idx_jobs_created_at - for date sorting'
\echo '4. idx_jobs_result_token - for token lookups'
\echo ''

-- Check if all expected indexes exist
\echo '=== Index Verification ==='
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'jobs' AND indexname = 'idx_jobs_session_id') 
        THEN '✓ idx_jobs_session_id exists'
        ELSE '✗ idx_jobs_session_id MISSING'
    END AS session_index,
    CASE 
        WHEN EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'jobs' AND indexname = 'idx_jobs_status') 
        THEN '✓ idx_jobs_status exists'
        ELSE '✗ idx_jobs_status MISSING'
    END AS status_index,
    CASE 
        WHEN EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'jobs' AND indexname = 'idx_jobs_created_at') 
        THEN '✓ idx_jobs_created_at exists'
        ELSE '✗ idx_jobs_created_at MISSING'
    END AS created_at_index,
    CASE 
        WHEN EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'jobs' AND indexname = 'idx_jobs_result_token') 
        THEN '✓ idx_jobs_result_token exists'
        ELSE '✗ idx_jobs_result_token MISSING'
    END AS token_index;

\echo ''
\echo '=== Table Statistics ==='
SELECT 
    schemaname,
    tablename,
    n_live_tup as row_count,
    n_dead_tup as dead_rows,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE tablename = 'jobs';

\echo ''
\echo '=== Index Usage Statistics ==='
SELECT 
    indexrelname as index_name,
    idx_scan as times_used,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE schemaname = 'public' AND relname = 'jobs'
ORDER BY idx_scan DESC;
