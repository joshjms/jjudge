-- Add hash and storage keys to testcases table
ALTER TABLE testcases ADD COLUMN in_key TEXT NOT NULL DEFAULT '';
ALTER TABLE testcases ADD COLUMN out_key TEXT NOT NULL DEFAULT '';
ALTER TABLE testcases ADD COLUMN hash TEXT NOT NULL DEFAULT '';

-- Update testcase_groups to reference problem_id instead of bundle_id
-- First add the new column
ALTER TABLE testcase_groups ADD COLUMN problem_id INTEGER;

-- Copy problem_id from testcase_bundles (for existing data)
UPDATE testcase_groups tg
SET problem_id = tb.problem_id
FROM testcase_bundles tb
WHERE tg.bundle_id = tb.id;

-- Make problem_id NOT NULL and add foreign key
ALTER TABLE testcase_groups ALTER COLUMN problem_id SET NOT NULL;
ALTER TABLE testcase_groups ADD CONSTRAINT testcase_groups_problem_id_fkey
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;

-- Drop the old bundle_id column and index
DROP INDEX IF EXISTS testcase_groups_bundle_id_idx;
ALTER TABLE testcase_groups DROP COLUMN bundle_id;

-- Create index on new problem_id column
CREATE INDEX testcase_groups_problem_id_idx ON testcase_groups(problem_id);

-- Drop testcase_bundles table as it's no longer needed
DROP TABLE IF EXISTS testcase_bundles CASCADE;

-- Remove testcase_bundle column from problems table
ALTER TABLE problems DROP COLUMN IF EXISTS testcase_bundle;
