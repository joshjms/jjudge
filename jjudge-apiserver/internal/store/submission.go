package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jjudge-oj/api/types"
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
		SELECT s.id, s.problem_id, s.user_id, u.username, s.code, s.language, s.verdict, s.score,
		       s.cpu_time, s.memory, s.message, s.tests_passed, s.tests_total,
		       s.created_at, s.updated_at, s.testcase_results
		FROM submissions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.id = $1`
	var submission types.Submission
	var resultsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&submission.ID,
		&submission.ProblemID,
		&submission.UserID,
		&submission.Username,
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

func (r *SubmissionRepository) List(ctx context.Context, problemID, userID int) ([]types.Submission, error) {
	query := `SELECT s.id, s.problem_id, s.user_id, u.username, s.code, s.language, s.verdict, s.score,
	                 s.cpu_time, s.memory, s.message, s.tests_passed, s.tests_total,
	                 s.created_at, s.updated_at
	          FROM submissions s
	          LEFT JOIN users u ON u.id = s.user_id
	          WHERE 1=1`
	args := []any{}
	argIdx := 1

	if problemID > 0 {
		query += fmt.Sprintf(" AND s.problem_id = $%d", argIdx)
		args = append(args, problemID)
		argIdx++
	}
	if userID > 0 {
		query += fmt.Sprintf(" AND s.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}

	query += " ORDER BY s.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []types.Submission
	for rows.Next() {
		var s types.Submission
		if err := rows.Scan(
			&s.ID, &s.ProblemID, &s.UserID, &s.Username, &s.Code, &s.Language,
			&s.Verdict, &s.Score, &s.CPUTime, &s.Memory, &s.Message,
			&s.TestsPassed, &s.TestsTotal, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}
	return submissions, rows.Err()
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
