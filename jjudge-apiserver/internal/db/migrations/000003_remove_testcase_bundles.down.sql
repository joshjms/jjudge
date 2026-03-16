-- Recreate testcase_bundles table
CREATE TABLE IF NOT EXISTS testcase_bundles (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS testcase_bundles_problem_version_idx ON testcase_bundles(problem_id, version);

-- Add testcase_bundle column back to problems table
ALTER TABLE problems ADD COLUMN testcase_bundle JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Restore testcase_groups to reference bundle_id instead of problem_id
DROP INDEX IF EXISTS testcase_groups_problem_id_idx;
ALTER TABLE testcase_groups ADD COLUMN bundle_id BIGINT;
-- Note: bundle_id will be NULL for existing data since bundles were removed
ALTER TABLE testcase_groups DROP CONSTRAINT IF EXISTS testcase_groups_problem_id_fkey;
ALTER TABLE testcase_groups DROP COLUMN problem_id;

-- Recreate the bundle_id index
CREATE INDEX testcase_groups_bundle_id_idx ON testcase_groups(bundle_id);

-- Remove new columns from testcases table
ALTER TABLE testcases DROP COLUMN IF EXISTS hash;
ALTER TABLE testcases DROP COLUMN IF EXISTS out_key;
ALTER TABLE testcases DROP COLUMN IF EXISTS in_key;
