DROP INDEX IF EXISTS testcase_bundles_problem_id_idx;
CREATE UNIQUE INDEX IF NOT EXISTS testcase_bundles_problem_version_idx ON testcase_bundles(problem_id, version);
