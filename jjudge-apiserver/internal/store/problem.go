package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jjudge-oj/api/types"
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
		SELECT id, title, description, difficulty, time_limit, memory_limit, tags, created_at, updated_at
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
		var tagsJSON []byte
		if err := rows.Scan(
			&problem.ID,
			&problem.Title,
			&problem.Description,
			&problem.Difficulty,
			&problem.TimeLimit,
			&problem.MemoryLimit,
			&tagsJSON,
			&problem.CreatedAt,
			&problem.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		_ = json.Unmarshal(tagsJSON, &problem.Tags)
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

func (r *ProblemRepository) Get(ctx context.Context, id int) (types.Problem, error) {
	const query = `
		SELECT id, title, description, difficulty, time_limit, memory_limit, tags, created_at, updated_at
		FROM problems
		WHERE id = $1`
	var problem types.Problem
	var tagsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&problem.ID,
		&problem.Title,
		&problem.Description,
		&problem.Difficulty,
		&problem.TimeLimit,
		&problem.MemoryLimit,
		&tagsJSON,
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
	return problem, nil
}

func (r *ProblemRepository) GetWithTestcases(ctx context.Context, id int) (types.Problem, error) {
	problem, err := r.Get(ctx, id)
	if err != nil {
		return types.Problem{}, err
	}

	// Query testcase groups and testcases directly
	const query = `
		SELECT g.id, g.ordinal, g.name, g.points,
			   t.id, t.ordinal, t.input, t.output, t.in_key, t.out_key, t.hash, t.is_hidden
		FROM testcase_groups g
		LEFT JOIN testcases t ON t.testcase_group_id = g.id
		WHERE g.problem_id = $1
		ORDER BY g.ordinal, t.ordinal`
	rows, err := r.db.QueryContext(ctx, query, problem.ID)
	if err != nil {
		return types.Problem{}, err
	}
	defer rows.Close()

	groupsByID := make(map[int]int)
	groups := make([]types.TestcaseGroup, 0)

	for rows.Next() {
		var (
			groupID      int
			groupOrdinal int
			groupName    string
			groupPoints  int
			testcaseID   sql.NullInt64
			testOrdinal  sql.NullInt64
			input        sql.NullString
			output       sql.NullString
			inKey        sql.NullString
			outKey       sql.NullString
			hash         sql.NullString
			isHidden     sql.NullBool
		)
		if err := rows.Scan(
			&groupID,
			&groupOrdinal,
			&groupName,
			&groupPoints,
			&testcaseID,
			&testOrdinal,
			&input,
			&output,
			&inKey,
			&outKey,
			&hash,
			&isHidden,
		); err != nil {
			return types.Problem{}, err
		}

		groupIndex, exists := groupsByID[groupID]
		if !exists {
			groupIndex = len(groups)
			groupsByID[groupID] = groupIndex
			groups = append(groups, types.TestcaseGroup{
				ID:        groupID,
				Ordinal:   groupOrdinal,
				ProblemID: problem.ID,
				Name:      groupName,
				Points:    groupPoints,
			})
		}

		if testcaseID.Valid {
			groups[groupIndex].Testcases = append(groups[groupIndex].Testcases, types.Testcase{
				ID:              int(testcaseID.Int64),
				Ordinal:         int(testOrdinal.Int64),
				TestcaseGroupID: groupID,
				Input:           input.String,
				Output:          output.String,
				InKey:           inKey.String,
				OutKey:          outKey.String,
				Hash:            hash.String,
				IsHidden:        isHidden.Bool,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return types.Problem{}, err
	}

	problem.TestcaseGroups = groups
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

// SaveTestcaseGroups saves testcase groups and their testcases for a problem.
// It replaces all existing testcase groups for the problem.
func (r *ProblemRepository) SaveTestcaseGroups(ctx context.Context, problemID int, groups []types.TestcaseGroup) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete existing testcase groups (cascades to testcases)
	if _, err = tx.ExecContext(ctx, `DELETE FROM testcase_groups WHERE problem_id = $1`, problemID); err != nil {
		return err
	}

	// Insert new testcase groups and testcases
	for _, group := range groups {
		var groupID int
		if err = tx.QueryRowContext(
			ctx,
			`INSERT INTO testcase_groups (problem_id, ordinal, name, points) VALUES ($1, $2, $3, $4) RETURNING id`,
			problemID,
			group.Ordinal,
			group.Name,
			group.Points,
		).Scan(&groupID); err != nil {
			return err
		}

		// Insert testcases for this group
		for _, testcase := range group.Testcases {
			if _, err = tx.ExecContext(
				ctx,
				`INSERT INTO testcases (testcase_group_id, ordinal, input, output, in_key, out_key, hash, is_hidden)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				groupID,
				testcase.Ordinal,
				testcase.Input,
				testcase.Output,
				testcase.InKey,
				testcase.OutKey,
				testcase.Hash,
				testcase.IsHidden,
			); err != nil {
				return err
			}
		}
	}

	// Update problem's updated_at timestamp
	if _, err = tx.ExecContext(ctx, `UPDATE problems SET updated_at = $1 WHERE id = $2`, time.Now(), problemID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
