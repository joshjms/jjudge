package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jjudge-oj/api/types"
)

// BlogRepository handles persistence for blog posts.
type BlogRepository struct {
	db *sql.DB
}

func NewBlogRepository(db *sql.DB) *BlogRepository {
	return &BlogRepository{db: db}
}

const blogSelectCols = `
	bp.id, bp.title, bp.slug, bp.content, bp.excerpt,
	bp.author_id, bp.published, bp.tags, bp.created_at, bp.updated_at,
	u.username, u.name`

func scanBlogPost(row interface{ Scan(...any) error }) (types.BlogPost, error) {
	var post types.BlogPost
	var tagsRaw []byte
	var authorUsername, authorName string

	if err := row.Scan(
		&post.ID,
		&post.Title,
		&post.Slug,
		&post.Content,
		&post.Excerpt,
		&post.AuthorID,
		&post.Published,
		&tagsRaw,
		&post.CreatedAt,
		&post.UpdatedAt,
		&authorUsername,
		&authorName,
	); err != nil {
		return types.BlogPost{}, err
	}

	if err := json.Unmarshal(tagsRaw, &post.Tags); err != nil {
		post.Tags = []string{}
	}
	if post.Tags == nil {
		post.Tags = []string{}
	}

	post.Author = &types.BlogAuthor{
		Username: authorUsername,
		Name:     authorName,
	}
	return post, nil
}

// List returns blog posts. If publishedOnly is true only published posts are returned.
func (r *BlogRepository) List(ctx context.Context, offset, limit int, publishedOnly bool) ([]types.BlogPost, int, error) {
	var countQuery string
	var listQuery string

	if publishedOnly {
		countQuery = `SELECT COUNT(*) FROM blog_posts WHERE published = true`
		listQuery = `
			SELECT ` + blogSelectCols + `
			FROM blog_posts bp
			JOIN users u ON u.id = bp.author_id
			WHERE bp.published = true
			ORDER BY bp.created_at DESC
			LIMIT $1 OFFSET $2`
	} else {
		countQuery = `SELECT COUNT(*) FROM blog_posts`
		listQuery = `
			SELECT ` + blogSelectCols + `
			FROM blog_posts bp
			JOIN users u ON u.id = bp.author_id
			ORDER BY bp.created_at DESC
			LIMIT $1 OFFSET $2`
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []types.BlogPost
	for rows.Next() {
		post, err := scanBlogPost(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}
	if posts == nil {
		posts = []types.BlogPost{}
	}
	return posts, total, rows.Err()
}

// Get returns a single blog post by slug.
func (r *BlogRepository) Get(ctx context.Context, slug string) (types.BlogPost, error) {
	const query = `
		SELECT ` + blogSelectCols + `
		FROM blog_posts bp
		JOIN users u ON u.id = bp.author_id
		WHERE bp.slug = $1`

	post, err := scanBlogPost(r.db.QueryRowContext(ctx, query, slug))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.BlogPost{}, ErrNotFound
		}
		return types.BlogPost{}, err
	}
	return post, nil
}

// SlugExists reports whether a slug is already taken (optionally excluding one post by id).
func (r *BlogRepository) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM blog_posts WHERE slug = $1 AND id != $2)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, slug, excludeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// Create inserts a new blog post and returns it with its assigned ID.
func (r *BlogRepository) Create(ctx context.Context, post types.BlogPost) (types.BlogPost, error) {
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	tagsRaw, err := json.Marshal(post.Tags)
	if err != nil {
		return types.BlogPost{}, err
	}

	const query = `
		INSERT INTO blog_posts (title, slug, content, excerpt, author_id, published, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	if err := r.db.QueryRowContext(ctx, query,
		post.Title,
		post.Slug,
		post.Content,
		post.Excerpt,
		post.AuthorID,
		post.Published,
		tagsRaw,
		post.CreatedAt,
		post.UpdatedAt,
	).Scan(&post.ID); err != nil {
		return types.BlogPost{}, err
	}
	return post, nil
}

// Update modifies an existing blog post.
func (r *BlogRepository) Update(ctx context.Context, post types.BlogPost) (types.BlogPost, error) {
	post.UpdatedAt = time.Now()

	tagsRaw, err := json.Marshal(post.Tags)
	if err != nil {
		return types.BlogPost{}, err
	}

	const query = `
		UPDATE blog_posts
		SET title      = $1,
		    slug       = $2,
		    content    = $3,
		    excerpt    = $4,
		    published  = $5,
		    tags       = $6,
		    updated_at = $7
		WHERE id = $8`

	result, err := r.db.ExecContext(ctx, query,
		post.Title,
		post.Slug,
		post.Content,
		post.Excerpt,
		post.Published,
		tagsRaw,
		post.UpdatedAt,
		post.ID,
	)
	if err != nil {
		return types.BlogPost{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return types.BlogPost{}, err
	}
	if affected == 0 {
		return types.BlogPost{}, ErrNotFound
	}
	return post, nil
}

// Delete removes a blog post by slug.
func (r *BlogRepository) Delete(ctx context.Context, slug string) error {
	const query = `DELETE FROM blog_posts WHERE slug = $1`
	result, err := r.db.ExecContext(ctx, query, slug)
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
