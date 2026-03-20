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

// ContestRepository handles persistence for contests.
type ContestRepository struct {
	db *sql.DB
}

func NewContestRepository(db *sql.DB) *ContestRepository {
	return &ContestRepository{db: db}
}

// ---------- Contest CRUD ----------

func (r *ContestRepository) ListContests(ctx context.Context, offset, limit int, publicOnly bool) ([]types.Contest, int, error) {
	var countQuery string
	if publicOnly {
		countQuery = `SELECT COUNT(*) FROM contests WHERE approval_status = 'approved'`
	} else {
		countQuery = `SELECT COUNT(*) FROM contests`
	}
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	var query string
	if publicOnly {
		query = `
		SELECT id, title, description, start_time, end_time,
		       scoring_type, visibility, owner_id, created_at, updated_at, approval_status
		FROM contests
		WHERE approval_status = 'approved'
		ORDER BY start_time DESC
		LIMIT $1 OFFSET $2`
	} else {
		query = `
		SELECT id, title, description, start_time, end_time,
		       scoring_type, visibility, owner_id, created_at, updated_at, approval_status
		FROM contests
		ORDER BY start_time DESC
		LIMIT $1 OFFSET $2`
	}
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contests []types.Contest
	for rows.Next() {
		var c types.Contest
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Description, &c.StartTime, &c.EndTime,
			&c.ScoringType, &c.Visibility, &c.OwnerID, &c.CreatedAt, &c.UpdatedAt,
			&c.ApprovalStatus,
		); err != nil {
			return nil, 0, err
		}
		contests = append(contests, c)
	}
	return contests, total, rows.Err()
}

func (r *ContestRepository) ListPendingContests(ctx context.Context, offset, limit int) ([]types.Contest, int, error) {
	const countQuery = `SELECT COUNT(*) FROM contests WHERE approval_status = 'pending'`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT id, title, description, start_time, end_time,
		       scoring_type, visibility, owner_id, created_at, updated_at, approval_status
		FROM contests
		WHERE approval_status = 'pending'
		ORDER BY start_time DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contests []types.Contest
	for rows.Next() {
		var c types.Contest
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Description, &c.StartTime, &c.EndTime,
			&c.ScoringType, &c.Visibility, &c.OwnerID, &c.CreatedAt, &c.UpdatedAt,
			&c.ApprovalStatus,
		); err != nil {
			return nil, 0, err
		}
		contests = append(contests, c)
	}
	return contests, total, rows.Err()
}

func (r *ContestRepository) ListContestsByOwner(ctx context.Context, ownerID, offset, limit int) ([]types.Contest, int, error) {
	const countQuery = `SELECT COUNT(*) FROM contests WHERE owner_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, ownerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT id, title, description, start_time, end_time,
		       scoring_type, visibility, owner_id, created_at, updated_at, approval_status
		FROM contests
		WHERE owner_id = $1
		ORDER BY start_time DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contests []types.Contest
	for rows.Next() {
		var c types.Contest
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Description, &c.StartTime, &c.EndTime,
			&c.ScoringType, &c.Visibility, &c.OwnerID, &c.CreatedAt, &c.UpdatedAt,
			&c.ApprovalStatus,
		); err != nil {
			return nil, 0, err
		}
		contests = append(contests, c)
	}
	return contests, total, rows.Err()
}

func (r *ContestRepository) GetContest(ctx context.Context, id int) (types.Contest, error) {
	const query = `
		SELECT id, title, description, start_time, end_time,
		       scoring_type, visibility, owner_id, created_at, updated_at, approval_status
		FROM contests
		WHERE id = $1`
	var c types.Contest
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.Description, &c.StartTime, &c.EndTime,
		&c.ScoringType, &c.Visibility, &c.OwnerID, &c.CreatedAt, &c.UpdatedAt,
		&c.ApprovalStatus,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Contest{}, ErrNotFound
		}
		return types.Contest{}, err
	}
	return c, nil
}

func (r *ContestRepository) GetContestWithProblems(ctx context.Context, id int) (types.Contest, error) {
	contest, err := r.GetContest(ctx, id)
	if err != nil {
		return types.Contest{}, err
	}

	problems, err := r.ListContestProblems(ctx, id)
	if err != nil {
		return types.Contest{}, err
	}
	contest.Problems = problems
	return contest, nil
}

func (r *ContestRepository) CreateContest(ctx context.Context, c types.Contest) (types.Contest, error) {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	if c.ApprovalStatus == "" {
		c.ApprovalStatus = "approved"
	}

	const query = `
		INSERT INTO contests (title, description, start_time, end_time, scoring_type, visibility, owner_id, approval_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`
	if err := r.db.QueryRowContext(ctx, query,
		c.Title, c.Description, c.StartTime, c.EndTime,
		c.ScoringType, c.Visibility, c.OwnerID, c.ApprovalStatus, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID); err != nil {
		return types.Contest{}, err
	}
	return c, nil
}

func (r *ContestRepository) UpdateContest(ctx context.Context, c types.Contest) (types.Contest, error) {
	c.UpdatedAt = time.Now()

	if c.ApprovalStatus == "" {
		c.ApprovalStatus = "approved"
	}

	const query = `
		UPDATE contests
		SET title = $1, description = $2, start_time = $3, end_time = $4,
		    scoring_type = $5, visibility = $6, approval_status = $7, updated_at = $8
		WHERE id = $9`
	result, err := r.db.ExecContext(ctx, query,
		c.Title, c.Description, c.StartTime, c.EndTime,
		c.ScoringType, c.Visibility, c.ApprovalStatus, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return types.Contest{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.Contest{}, err
	}
	if affected == 0 {
		return types.Contest{}, ErrNotFound
	}
	return c, nil
}

func (r *ContestRepository) ApproveContest(ctx context.Context, id int) error {
	const query = `UPDATE contests SET approval_status = 'approved', updated_at = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
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

func (r *ContestRepository) RejectContest(ctx context.Context, id int) error {
	const query = `UPDATE contests SET approval_status = 'rejected', updated_at = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
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

func (r *ContestRepository) DeleteContest(ctx context.Context, id int) error {
	const query = `DELETE FROM contests WHERE id = $1`
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

// ---------- Contest Problems ----------

func (r *ContestRepository) ListContestProblems(ctx context.Context, contestID int) ([]types.ContestProblem, error) {
	const query = `
		SELECT cp.contest_id, cp.problem_id, cp.ordinal, cp.max_points,
		       p.id, p.title, p.description, p.difficulty, p.time_limit, p.memory_limit, p.tags, p.created_at, p.updated_at
		FROM contest_problems cp
		JOIN problems p ON p.id = cp.problem_id
		WHERE cp.contest_id = $1
		ORDER BY cp.ordinal`
	rows, err := r.db.QueryContext(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.ContestProblem
	for rows.Next() {
		var cp types.ContestProblem
		var p types.Problem
		var tagsJSON []byte
		if err := rows.Scan(
			&cp.ContestID, &cp.ProblemID, &cp.Ordinal, &cp.MaxPoints,
			&p.ID, &p.Title, &p.Description, &p.Difficulty, &p.TimeLimit, &p.MemoryLimit, &tagsJSON, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &p.Tags)
		cp.Problem = &p
		items = append(items, cp)
	}
	return items, rows.Err()
}

func (r *ContestRepository) AddContestProblem(ctx context.Context, cp types.ContestProblem) (types.ContestProblem, error) {
	const query = `
		INSERT INTO contest_problems (contest_id, problem_id, ordinal, max_points)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (contest_id, problem_id) DO UPDATE
		  SET ordinal = EXCLUDED.ordinal, max_points = EXCLUDED.max_points`
	_, err := r.db.ExecContext(ctx, query, cp.ContestID, cp.ProblemID, cp.Ordinal, cp.MaxPoints)
	return cp, err
}

func (r *ContestRepository) RemoveContestProblem(ctx context.Context, contestID, problemID int) error {
	const query = `DELETE FROM contest_problems WHERE contest_id = $1 AND problem_id = $2`
	result, err := r.db.ExecContext(ctx, query, contestID, problemID)
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

// ReorderContestProblems updates ordinals for multiple problems in a single transaction.
func (r *ContestRepository) ReorderContestProblems(ctx context.Context, contestID int, ordinals map[int]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE contest_problems SET ordinal = $1 WHERE contest_id = $2 AND problem_id = $3`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for problemID, ordinal := range ordinals {
		if _, err := stmt.ExecContext(ctx, ordinal, contestID, problemID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------- Registrations ----------

func (r *ContestRepository) Register(ctx context.Context, contestID, userID int) error {
	const query = `
		INSERT INTO contest_registrations (contest_id, user_id, registered_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, contestID, userID, time.Now())
	return err
}

func (r *ContestRepository) Unregister(ctx context.Context, contestID, userID int) error {
	const query = `DELETE FROM contest_registrations WHERE contest_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, contestID, userID)
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

func (r *ContestRepository) IsRegistered(ctx context.Context, contestID, userID int) (bool, error) {
	const query = `SELECT 1 FROM contest_registrations WHERE contest_id = $1 AND user_id = $2`
	var dummy int
	err := r.db.QueryRowContext(ctx, query, contestID, userID).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ContestRepository) ListRegistrations(ctx context.Context, contestID int) ([]types.ContestRegistration, error) {
	const query = `
		SELECT cr.contest_id, cr.user_id, u.username, cr.registered_at
		FROM contest_registrations cr
		JOIN users u ON u.id = cr.user_id
		WHERE cr.contest_id = $1
		ORDER BY cr.registered_at`
	rows, err := r.db.QueryContext(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []types.ContestRegistration
	for rows.Next() {
		var reg types.ContestRegistration
		if err := rows.Scan(&reg.ContestID, &reg.UserID, &reg.Username, &reg.RegisteredAt); err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	return regs, rows.Err()
}

// ---------- Contest Submissions ----------

func (r *ContestRepository) CreateContestSubmission(ctx context.Context, cs types.ContestSubmission) (types.ContestSubmission, error) {
	now := time.Now()
	cs.SubmittedAt = now
	cs.UpdatedAt = now

	resultsJSON, err := json.Marshal(cs.TestcaseResults)
	if err != nil {
		return types.ContestSubmission{}, err
	}

	const query = `
		INSERT INTO contest_submissions (
			contest_id, problem_id, user_id, code, language, verdict, score,
			cpu_time, memory, message, tests_passed, tests_total,
			testcase_results, submitted_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`
	if err := r.db.QueryRowContext(ctx, query,
		cs.ContestID, cs.ProblemID, cs.UserID, cs.Code, cs.Language,
		cs.Verdict, cs.Score, cs.CPUTime, cs.Memory, cs.Message,
		cs.TestsPassed, cs.TestsTotal, resultsJSON, cs.SubmittedAt, cs.UpdatedAt,
	).Scan(&cs.ID); err != nil {
		return types.ContestSubmission{}, err
	}
	return cs, nil
}

func (r *ContestRepository) GetContestSubmission(ctx context.Context, id int64) (types.ContestSubmission, error) {
	const query = `
		SELECT cs.id, cs.contest_id, cs.problem_id, cs.user_id, u.username,
		       cs.code, cs.language, cs.verdict, cs.score,
		       cs.cpu_time, cs.memory, cs.message, cs.tests_passed, cs.tests_total,
		       cs.testcase_results, cs.submitted_at, cs.updated_at
		FROM contest_submissions cs
		LEFT JOIN users u ON u.id = cs.user_id
		WHERE cs.id = $1`
	var cs types.ContestSubmission
	var resultsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cs.ID, &cs.ContestID, &cs.ProblemID, &cs.UserID, &cs.Username,
		&cs.Code, &cs.Language, &cs.Verdict, &cs.Score,
		&cs.CPUTime, &cs.Memory, &cs.Message, &cs.TestsPassed, &cs.TestsTotal,
		&resultsJSON, &cs.SubmittedAt, &cs.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ContestSubmission{}, ErrNotFound
		}
		return types.ContestSubmission{}, err
	}
	_ = json.Unmarshal(resultsJSON, &cs.TestcaseResults)
	return cs, nil
}

func (r *ContestRepository) UpdateContestSubmission(ctx context.Context, cs types.ContestSubmission) (types.ContestSubmission, error) {
	cs.UpdatedAt = time.Now()

	resultsJSON, err := json.Marshal(cs.TestcaseResults)
	if err != nil {
		return types.ContestSubmission{}, err
	}

	const query = `
		UPDATE contest_submissions
		SET verdict = $1, score = $2, cpu_time = $3, memory = $4, message = $5,
		    tests_passed = $6, tests_total = $7, updated_at = $8, testcase_results = $9
		WHERE id = $10`
	result, err := r.db.ExecContext(ctx, query,
		cs.Verdict, cs.Score, cs.CPUTime, cs.Memory, cs.Message,
		cs.TestsPassed, cs.TestsTotal, cs.UpdatedAt, resultsJSON, cs.ID,
	)
	if err != nil {
		return types.ContestSubmission{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.ContestSubmission{}, err
	}
	if affected == 0 {
		return types.ContestSubmission{}, ErrNotFound
	}
	return cs, nil
}

func (r *ContestRepository) ListContestSubmissions(ctx context.Context, contestID, problemID, userID int) ([]types.ContestSubmission, error) {
	query := `
		SELECT cs.id, cs.contest_id, cs.problem_id, cs.user_id, u.username,
		       cs.code, cs.language, cs.verdict, cs.score,
		       cs.cpu_time, cs.memory, cs.message, cs.tests_passed, cs.tests_total,
		       cs.submitted_at, cs.updated_at
		FROM contest_submissions cs
		LEFT JOIN users u ON u.id = cs.user_id
		WHERE cs.contest_id = $1`
	args := []any{contestID}
	argIdx := 2

	if problemID > 0 {
		query += fmt.Sprintf(" AND cs.problem_id = $%d", argIdx)
		args = append(args, problemID)
		argIdx++
	}
	if userID > 0 {
		query += fmt.Sprintf(" AND cs.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	query += " ORDER BY cs.submitted_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []types.ContestSubmission
	for rows.Next() {
		var cs types.ContestSubmission
		if err := rows.Scan(
			&cs.ID, &cs.ContestID, &cs.ProblemID, &cs.UserID, &cs.Username,
			&cs.Code, &cs.Language, &cs.Verdict, &cs.Score,
			&cs.CPUTime, &cs.Memory, &cs.Message, &cs.TestsPassed, &cs.TestsTotal,
			&cs.SubmittedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, err
		}
		submissions = append(submissions, cs)
	}
	return submissions, rows.Err()
}

func (r *ContestRepository) DeleteContestSubmission(ctx context.Context, id int64) error {
	const query = `DELETE FROM contest_submissions WHERE id = $1`
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

// ---------- Leaderboard ----------

// LeaderboardRow is a raw aggregate row returned by GetLeaderboardRows.
type LeaderboardRow struct {
	UserID        int
	Username      string
	ProblemID     int
	Attempts      int
	BestScore     int
	Accepted      bool
	AcceptSeconds *float64 // nil if never accepted
}

func (r *ContestRepository) GetLeaderboardRows(ctx context.Context, contestID int) ([]LeaderboardRow, error) {
	const query = `
		SELECT cs.user_id, u.username, cs.problem_id,
		       COUNT(*) AS attempts,
		       MAX(cs.score) AS best_score,
		       BOOL_OR(cs.verdict = 2) AS accepted,
		       MIN(CASE WHEN cs.verdict = 2
		               THEN EXTRACT(EPOCH FROM (cs.submitted_at - c.start_time)) END) AS accept_seconds
		FROM contest_submissions cs
		JOIN users u ON u.id = cs.user_id
		JOIN contests c ON c.id = cs.contest_id
		WHERE cs.contest_id = $1
		GROUP BY cs.user_id, u.username, cs.problem_id`

	rows, err := r.db.QueryContext(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LeaderboardRow
	for rows.Next() {
		var row LeaderboardRow
		if err := rows.Scan(
			&row.UserID, &row.Username, &row.ProblemID,
			&row.Attempts, &row.BestScore, &row.Accepted, &row.AcceptSeconds,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// ListSubmissionsForContestProblem returns all contest submissions for a given contest+problem.
func (r *ContestRepository) ListSubmissionsForContestProblem(ctx context.Context, contestID, problemID int) ([]types.ContestSubmission, error) {
	const query = `
		SELECT id, contest_id, problem_id, user_id, code, language,
		       verdict, score, cpu_time, memory, message, tests_passed, tests_total,
		       testcase_results, submitted_at, updated_at
		FROM contest_submissions
		WHERE contest_id = $1 AND problem_id = $2
		ORDER BY submitted_at`
	rows, err := r.db.QueryContext(ctx, query, contestID, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []types.ContestSubmission
	for rows.Next() {
		var cs types.ContestSubmission
		var resultsJSON []byte
		if err := rows.Scan(
			&cs.ID, &cs.ContestID, &cs.ProblemID, &cs.UserID, &cs.Code, &cs.Language,
			&cs.Verdict, &cs.Score, &cs.CPUTime, &cs.Memory, &cs.Message,
			&cs.TestsPassed, &cs.TestsTotal, &resultsJSON, &cs.SubmittedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(resultsJSON, &cs.TestcaseResults)
		submissions = append(submissions, cs)
	}
	return submissions, rows.Err()
}
