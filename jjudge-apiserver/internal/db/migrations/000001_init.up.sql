-- Initial schema for jjudge apiserver.

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS problems (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    difficulty INTEGER NOT NULL DEFAULT 0,
    time_limit BIGINT NOT NULL DEFAULT 0,
    memory_limit BIGINT NOT NULL DEFAULT 0,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    testcase_bundle JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS submissions (
    id BIGSERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    language TEXT NOT NULL,
    verdict INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    cpu_time BIGINT NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    tests_passed INTEGER NOT NULL DEFAULT 0,
    tests_total INTEGER NOT NULL DEFAULT 0,
    testcase_results JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS submissions_problem_id_idx ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS submissions_user_id_idx ON submissions(user_id);

CREATE TABLE IF NOT EXISTS testcase_bundles (
    id BIGSERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS testcase_bundles_problem_id_idx ON testcase_bundles(problem_id);

CREATE TABLE IF NOT EXISTS testcase_groups (
    id BIGSERIAL PRIMARY KEY,
    bundle_id BIGINT NOT NULL REFERENCES testcase_bundles(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,
    points INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS testcase_groups_bundle_id_idx ON testcase_groups(bundle_id);

CREATE TABLE IF NOT EXISTS testcases (
    id BIGSERIAL PRIMARY KEY,
    testcase_group_id BIGINT NOT NULL REFERENCES testcase_groups(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL DEFAULT 0,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS testcases_group_id_idx ON testcases(testcase_group_id);

CREATE TABLE IF NOT EXISTS testcase_results (
    submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    testcase_id BIGINT NOT NULL REFERENCES testcases(id) ON DELETE CASCADE,
    verdict INTEGER NOT NULL DEFAULT 0,
    cpu_time BIGINT NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    input TEXT NOT NULL DEFAULT '',
    expected_output TEXT NOT NULL DEFAULT '',
    actual_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (submission_id, testcase_id)
);

CREATE INDEX IF NOT EXISTS testcase_results_submission_id_idx ON testcase_results(submission_id);
CREATE INDEX IF NOT EXISTS testcase_results_testcase_id_idx ON testcase_results(testcase_id);
