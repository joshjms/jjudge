package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jjudge-oj/apiserver/types"
)

// ProblemRepository handles persistence for problems.
type ProblemRepository struct {
	db *sql.DB
}

func NewProblemRepository(db *sql.DB) *ProblemRepository {
	return &ProblemRepository{db: db}
}

func (r *ProblemRepository) List(ctx context.Context, offset, limit int) ([]types.Problem, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}

	const countQuery = `SELECT COUNT(1) FROM problems`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	const listQuery = `
		SELECT id, title, description, difficulty, time_limit, memory_limit, tags, testcase_bundle, created_at, updated_at
		FROM problems
		ORDER BY id
		OFFSET $1 LIMIT $2`
	rows, err := r.db.QueryContext(ctx, listQuery, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	problems := make([]types.Problem, 0, limit)
	for rows.Next() {
		var problem types.Problem
		var tagsJSON, bundleJSON []byte
		if err := rows.Scan(
			&problem.ID,
			&problem.Title,
			&problem.Description,
			&problem.Difficulty,
			&problem.TimeLimit,
			&problem.MemoryLimit,
			&tagsJSON,
			&bundleJSON,
			&problem.CreatedAt,
			&problem.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		_ = json.Unmarshal(tagsJSON, &problem.Tags)
		_ = json.Unmarshal(bundleJSON, &problem.TestcaseBundle)
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

func (r *ProblemRepository) Get(ctx context.Context, id int) (types.Problem, error) {
	const query = `
		SELECT id, title, description, difficulty, time_limit, memory_limit, tags, testcase_bundle, created_at, updated_at
		FROM problems
		WHERE id = $1`
	var problem types.Problem
	var tagsJSON, bundleJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&problem.ID,
		&problem.Title,
		&problem.Description,
		&problem.Difficulty,
		&problem.TimeLimit,
		&problem.MemoryLimit,
		&tagsJSON,
		&bundleJSON,
		&problem.CreatedAt,
		&problem.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Problem{}, ErrNotFound
		}
		return types.Problem{}, err
	}

	_ = json.Unmarshal(tagsJSON, &problem.Tags)
	_ = json.Unmarshal(bundleJSON, &problem.TestcaseBundle)
	return problem, nil
}

func (r *ProblemRepository) Create(ctx context.Context, problem types.Problem) (types.Problem, error) {
	now := time.Now()
	problem.CreatedAt = now
	problem.UpdatedAt = now

	tagsJSON, err := json.Marshal(problem.Tags)
	if err != nil {
		return types.Problem{}, err
	}
	bundleJSON, err := json.Marshal(problem.TestcaseBundle)
	if err != nil {
		return types.Problem{}, err
	}

	const query = `
		INSERT INTO problems (title, description, difficulty, time_limit, memory_limit, tags, testcase_bundle, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`
	if err := r.db.QueryRowContext(
		ctx,
		query,
		problem.Title,
		problem.Description,
		problem.Difficulty,
		problem.TimeLimit,
		problem.MemoryLimit,
		tagsJSON,
		bundleJSON,
		problem.CreatedAt,
		problem.UpdatedAt,
	).Scan(&problem.ID); err != nil {
		return types.Problem{}, err
	}

	return problem, nil
}

func (r *ProblemRepository) Update(ctx context.Context, problem types.Problem) (types.Problem, error) {
	problem.UpdatedAt = time.Now()

	tagsJSON, err := json.Marshal(problem.Tags)
	if err != nil {
		return types.Problem{}, err
	}
	bundleJSON, err := json.Marshal(problem.TestcaseBundle)
	if err != nil {
		return types.Problem{}, err
	}

	const query = `
		UPDATE problems
		SET title = $1,
			description = $2,
			difficulty = $3,
			time_limit = $4,
			memory_limit = $5,
			tags = $6,
			testcase_bundle = $7,
			updated_at = $8
		WHERE id = $9`
	result, err := r.db.ExecContext(
		ctx,
		query,
		problem.Title,
		problem.Description,
		problem.Difficulty,
		problem.TimeLimit,
		problem.MemoryLimit,
		tagsJSON,
		bundleJSON,
		problem.UpdatedAt,
		problem.ID,
	)
	if err != nil {
		return types.Problem{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.Problem{}, err
	}
	if affected == 0 {
		return types.Problem{}, ErrNotFound
	}

	return problem, nil
}

func (r *ProblemRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM problems WHERE id = $1`
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
