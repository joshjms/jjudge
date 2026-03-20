package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jjudge-oj/api/types"
)

// UserRepository handles persistence for users.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (types.User, error) {
	const query = `
		SELECT id, username, email, name, role, password_hash, created_at, updated_at,
		       bio, github, codeforces, atcoder, website, avatar_url
		FROM users
		WHERE id = $1`
	var user types.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Bio,
		&user.GitHub,
		&user.Codeforces,
		&user.AtCoder,
		&user.Website,
		&user.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.User{}, ErrNotFound
		}
		return types.User{}, err
	}
	return user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (types.User, error) {
	const query = `
		SELECT id, username, email, name, role, password_hash, created_at, updated_at,
		       bio, github, codeforces, atcoder, website, avatar_url
		FROM users
		WHERE LOWER(username) = LOWER($1)`
	var user types.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Bio,
		&user.GitHub,
		&user.Codeforces,
		&user.AtCoder,
		&user.Website,
		&user.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.User{}, ErrNotFound
		}
		return types.User{}, err
	}
	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user types.User) (types.User, error) {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	const query = `
		INSERT INTO users (username, email, name, role, password_hash, created_at, updated_at,
		                   bio, github, codeforces, atcoder, website, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`
	if err := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Name,
		user.Role,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
		user.Bio,
		user.GitHub,
		user.Codeforces,
		user.AtCoder,
		user.Website,
		user.AvatarURL,
	).Scan(&user.ID); err != nil {
		return types.User{}, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user types.User) (types.User, error) {
	user.UpdatedAt = time.Now()

	const query = `
		UPDATE users
		SET username = $1,
			email = $2,
			name = $3,
			role = $4,
			password_hash = $5,
			updated_at = $6,
			bio = $7,
			github = $8,
			codeforces = $9,
			atcoder = $10,
			website = $11,
			avatar_url = $12
		WHERE id = $13`
	result, err := r.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Name,
		user.Role,
		user.PasswordHash,
		user.UpdatedAt,
		user.Bio,
		user.GitHub,
		user.Codeforces,
		user.AtCoder,
		user.Website,
		user.AvatarURL,
		user.ID,
	)
	if err != nil {
		return types.User{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.User{}, err
	}
	if affected == 0 {
		return types.User{}, ErrNotFound
	}
	return user, nil
}

func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]types.User, int, error) {
	const countQuery = `SELECT COUNT(*) FROM users`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT id, username, email, name, role, created_at, updated_at
		FROM users
		ORDER BY id ASC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var u types.User
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.Name,
			&u.Role,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM users WHERE id = $1`
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
