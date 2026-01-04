-- Add constraints to ensure completed jobs have result_id and failed jobs have error message

-- Add check constraint: completed jobs must have result_id
ALTER TABLE jobs ADD CONSTRAINT check_completed_has_result 
    CHECK (status != 'completed' OR (result_id IS NOT NULL AND result_id != ''));

-- Add check constraint: failed jobs must have error message
ALTER TABLE jobs ADD CONSTRAINT check_failed_has_error 
    CHECK (status != 'failed' OR (error IS NOT NULL AND error != ''));
