package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jjudge-oj/apiserver/types"
)

// SubmissionRepository handles persistence for submissions.
type SubmissionRepository struct {
	db *sql.DB
}

func NewSubmissionRepository(db *sql.DB) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

func (r *SubmissionRepository) Get(ctx context.Context, id int64) (types.Submission, error) {
	const query = `
		SELECT id, problem_id, user_id, code, language, verdict, score,
		       cpu_time, memory, message, tests_passed, tests_total,
		       created_at, updated_at, testcase_results
		FROM submissions
		WHERE id = $1`
	var submission types.Submission
	var resultsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&submission.ID,
		&submission.ProblemID,
		&submission.UserID,
		&submission.Code,
		&submission.Language,
		&submission.Verdict,
		&submission.Score,
		&submission.CPUTime,
		&submission.Memory,
		&submission.Message,
		&submission.TestsPassed,
		&submission.TestsTotal,
		&submission.CreatedAt,
		&submission.UpdatedAt,
		&resultsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Submission{}, ErrNotFound
		}
		return types.Submission{}, err
	}

	_ = json.Unmarshal(resultsJSON, &submission.TestcaseResults)
	return submission, nil
}

func (r *SubmissionRepository) Create(ctx context.Context, submission types.Submission) (types.Submission, error) {
	now := time.Now()
	submission.CreatedAt = now
	submission.UpdatedAt = now

	resultsJSON, err := json.Marshal(submission.TestcaseResults)
	if err != nil {
		return types.Submission{}, err
	}

	const query = `
		INSERT INTO submissions (
			problem_id, user_id, code, language, verdict, score,
			cpu_time, memory, message, tests_passed, tests_total,
			created_at, updated_at, testcase_results
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`
	if err := r.db.QueryRowContext(
		ctx,
		query,
		submission.ProblemID,
		submission.UserID,
		submission.Code,
		submission.Language,
		submission.Verdict,
		submission.Score,
		submission.CPUTime,
		submission.Memory,
		submission.Message,
		submission.TestsPassed,
		submission.TestsTotal,
		submission.CreatedAt,
		submission.UpdatedAt,
		resultsJSON,
	).Scan(&submission.ID); err != nil {
		return types.Submission{}, err
	}

	return submission, nil
}

func (r *SubmissionRepository) Update(ctx context.Context, submission types.Submission) (types.Submission, error) {
	submission.UpdatedAt = time.Now()

	resultsJSON, err := json.Marshal(submission.TestcaseResults)
	if err != nil {
		return types.Submission{}, err
	}

	const query = `
		UPDATE submissions
		SET verdict = $1,
			score = $2,
			cpu_time = $3,
			memory = $4,
			message = $5,
			tests_passed = $6,
			tests_total = $7,
			updated_at = $8,
			testcase_results = $9
		WHERE id = $10`
	result, err := r.db.ExecContext(
		ctx,
		query,
		submission.Verdict,
		submission.Score,
		submission.CPUTime,
		submission.Memory,
		submission.Message,
		submission.TestsPassed,
		submission.TestsTotal,
		submission.UpdatedAt,
		resultsJSON,
		submission.ID,
	)
	if err != nil {
		return types.Submission{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.Submission{}, err
	}
	if affected == 0 {
		return types.Submission{}, ErrNotFound
	}
	return submission, nil
}

func (r *SubmissionRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM submissions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
