ALTER TABLE problems DROP COLUMN IF EXISTS creator_id, DROP COLUMN IF EXISTS approval_status;
ALTER TABLE contests DROP COLUMN IF EXISTS approval_status;
