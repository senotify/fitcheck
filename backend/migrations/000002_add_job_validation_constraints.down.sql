-- Remove validation constraints

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS check_completed_has_result;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS check_failed_has_error;
