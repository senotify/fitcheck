-- Add index for result_token lookups
-- This improves performance for email link access
CREATE INDEX IF NOT EXISTS idx_jobs_result_token ON jobs(result_token) WHERE result_token IS NOT NULL;
