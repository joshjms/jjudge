BEGIN;

CREATE TABLE problems (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    difficulty INTEGER NOT NULL,
    time_limit BIGINT NOT NULL,
    memory_limit BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE problem_tags (
    problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (problem_id, tag)
);

CREATE TABLE testcase_groups (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    name TEXT NOT NULL,
    points INTEGER NOT NULL,
    UNIQUE (problem_id, ordinal)
);

CREATE TABLE testcases (
    id BIGSERIAL PRIMARY KEY,
    testcase_group_id BIGINT NOT NULL REFERENCES testcase_groups(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (testcase_group_id, ordinal)
);

CREATE TABLE submissions (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE RESTRICT,
    user_id BIGINT NOT NULL,
    code TEXT NOT NULL,
    language TEXT NOT NULL,
    verdict SMALLINT NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    cpu_time BIGINT NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    tests_passed INTEGER NOT NULL DEFAULT 0,
    tests_total INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE testcase_results (
    submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    testcase_id BIGINT NOT NULL REFERENCES testcases(id) ON DELETE CASCADE,
    verdict SMALLINT NOT NULL,
    cpu_time BIGINT NOT NULL,
    memory BIGINT NOT NULL,
    input TEXT,
    expected_output TEXT,
    actual_output TEXT,
    error_message TEXT,
    PRIMARY KEY (submission_id, testcase_id)
);

CREATE INDEX idx_testcase_groups_problem_id ON testcase_groups(problem_id);
CREATE INDEX idx_testcases_group_id ON testcases(testcase_group_id);
CREATE INDEX idx_submissions_problem_id ON submissions(problem_id);
CREATE INDEX idx_submissions_user_id ON submissions(user_id);
CREATE INDEX idx_testcase_results_submission_id ON testcase_results(submission_id);
CREATE INDEX idx_testcase_results_testcase_id ON testcase_results(testcase_id);

COMMIT;
