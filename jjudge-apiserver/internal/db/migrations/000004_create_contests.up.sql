-- Contests feature: adds contests, contest_problems, contest_registrations, contest_submissions.

CREATE TABLE IF NOT EXISTS contests (
    id           SERIAL      PRIMARY KEY,
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    scoring_type TEXT        NOT NULL DEFAULT 'icpc' CHECK (scoring_type IN ('icpc', 'ioi')),
    visibility   TEXT        NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    owner_id     INTEGER     NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS contests_owner_id_idx   ON contests(owner_id);
CREATE INDEX IF NOT EXISTS contests_start_time_idx ON contests(start_time);

CREATE TABLE IF NOT EXISTS contest_problems (
    contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE RESTRICT,
    ordinal    INTEGER NOT NULL DEFAULT 0,
    max_points INTEGER NOT NULL DEFAULT 100,
    PRIMARY KEY (contest_id, problem_id)
);

CREATE INDEX IF NOT EXISTS contest_problems_contest_id_idx ON contest_problems(contest_id);

CREATE TABLE IF NOT EXISTS contest_registrations (
    contest_id    INTEGER     NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id       INTEGER     NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    registered_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (contest_id, user_id)
);

CREATE INDEX IF NOT EXISTS contest_registrations_contest_id_idx ON contest_registrations(contest_id);
CREATE INDEX IF NOT EXISTS contest_registrations_user_id_idx    ON contest_registrations(user_id);

CREATE TABLE IF NOT EXISTS contest_submissions (
    id               BIGSERIAL   PRIMARY KEY,
    contest_id       INTEGER     NOT NULL REFERENCES contests(id)  ON DELETE CASCADE,
    problem_id       INTEGER     NOT NULL REFERENCES problems(id)  ON DELETE RESTRICT,
    user_id          INTEGER     NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    code             TEXT        NOT NULL,
    language         TEXT        NOT NULL,
    verdict          INTEGER     NOT NULL DEFAULT 0,
    score            INTEGER     NOT NULL DEFAULT 0,
    cpu_time         BIGINT      NOT NULL DEFAULT 0,
    memory           BIGINT      NOT NULL DEFAULT 0,
    message          TEXT        NOT NULL DEFAULT '',
    tests_passed     INTEGER     NOT NULL DEFAULT 0,
    tests_total      INTEGER     NOT NULL DEFAULT 0,
    testcase_results JSONB       NOT NULL DEFAULT '[]'::jsonb,
    submitted_at     TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cs_contest_id_idx           ON contest_submissions(contest_id);
CREATE INDEX IF NOT EXISTS cs_user_id_idx              ON contest_submissions(user_id);
CREATE INDEX IF NOT EXISTS cs_contest_problem_user_idx ON contest_submissions(contest_id, problem_id, user_id);
