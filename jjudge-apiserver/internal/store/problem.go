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
		SELECT p.id,
			p.title,
			p.description,
			p.difficulty,
			p.time_limit,
			p.memory_limit,
			p.tags,
			p.testcase_bundle,
			p.created_at,
			p.updated_at,
			tb.object_key,
			tb.sha256,
			tb.version
		FROM problems p
		LEFT JOIN LATERAL (
			SELECT object_key, sha256, version
			FROM testcase_bundles
			WHERE problem_id = p.id
			ORDER BY version DESC
			LIMIT 1
		) tb ON true
		ORDER BY p.id
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
		var objectKey, sha256 sql.NullString
		var version sql.NullInt64
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
			&objectKey,
			&sha256,
			&version,
		); err != nil {
			return nil, 0, err
		}

		_ = json.Unmarshal(tagsJSON, &problem.Tags)
		if objectKey.Valid && sha256.Valid && version.Valid {
			problem.TestcaseBundle = types.TestcaseBundle{
				ObjectKey: objectKey.String,
				SHA256:    sha256.String,
				Version:   int(version.Int64),
			}
		} else {
			_ = json.Unmarshal(bundleJSON, &problem.TestcaseBundle)
		}
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

func (r *ProblemRepository) Get(ctx context.Context, id int) (types.Problem, error) {
	const query = `
		SELECT p.id,
			p.title,
			p.description,
			p.difficulty,
			p.time_limit,
			p.memory_limit,
			p.tags,
			p.testcase_bundle,
			p.created_at,
			p.updated_at,
			tb.object_key,
			tb.sha256,
			tb.version
		FROM problems p
		LEFT JOIN LATERAL (
			SELECT object_key, sha256, version
			FROM testcase_bundles
			WHERE problem_id = p.id
			ORDER BY version DESC
			LIMIT 1
		) tb ON true
		WHERE p.id = $1`
	var problem types.Problem
	var tagsJSON, bundleJSON []byte
	var objectKey, sha256 sql.NullString
	var version sql.NullInt64
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
		&objectKey,
		&sha256,
		&version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Problem{}, ErrNotFound
		}
		return types.Problem{}, err
	}

	_ = json.Unmarshal(tagsJSON, &problem.Tags)
	if objectKey.Valid && sha256.Valid && version.Valid {
		problem.TestcaseBundle = types.TestcaseBundle{
			ObjectKey: objectKey.String,
			SHA256:    sha256.String,
			Version:   int(version.Int64),
		}
	} else {
		_ = json.Unmarshal(bundleJSON, &problem.TestcaseBundle)
	}
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

	const query = `
		INSERT INTO problems (title, description, difficulty, time_limit, memory_limit, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return types.Problem{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = tx.QueryRowContext(
		ctx,
		query,
		problem.Title,
		problem.Description,
		problem.Difficulty,
		problem.TimeLimit,
		problem.MemoryLimit,
		tagsJSON,
		problem.CreatedAt,
		problem.UpdatedAt,
	).Scan(&problem.ID); err != nil {
		return types.Problem{}, err
	}

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO testcase_bundles (problem_id, object_key, sha256, version) VALUES ($1, $2, $3, $4)`,
		problem.ID,
		problem.TestcaseBundle.ObjectKey,
		problem.TestcaseBundle.SHA256,
		problem.TestcaseBundle.Version,
	); err != nil {
		return types.Problem{}, err
	}

	if err = tx.Commit(); err != nil {
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

	const query = `
		UPDATE problems
		SET title = $1,
			description = $2,
			difficulty = $3,
			time_limit = $4,
			memory_limit = $5,
			tags = $6,
			updated_at = $7
		WHERE id = $8`
	result, err := r.db.ExecContext(
		ctx,
		query,
		problem.Title,
		problem.Description,
		problem.Difficulty,
		problem.TimeLimit,
		problem.MemoryLimit,
		tagsJSON,
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

func (r *ProblemRepository) GetLatestTestcaseBundle(ctx context.Context, problemID int) (types.TestcaseBundle, error) {
	const query = `
		SELECT object_key, sha256, version
		FROM testcase_bundles
		WHERE problem_id = $1
		ORDER BY version DESC
		LIMIT 1`
	var bundle types.TestcaseBundle
	err := r.db.QueryRowContext(ctx, query, problemID).Scan(
		&bundle.ObjectKey,
		&bundle.SHA256,
		&bundle.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.TestcaseBundle{}, ErrNotFound
		}
		return types.TestcaseBundle{}, err
	}
	return bundle, nil
}

func (r *ProblemRepository) AddTestcaseBundleVersion(ctx context.Context, problemID int, bundle types.TestcaseBundle) error {
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO testcase_bundles (problem_id, object_key, sha256, version) VALUES ($1, $2, $3, $4)`,
		problemID,
		bundle.ObjectKey,
		bundle.SHA256,
		bundle.Version,
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE problems SET testcase_bundle = $1, updated_at = $2 WHERE id = $3`,
		bundleJSON,
		time.Now(),
		problemID,
	)
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

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
