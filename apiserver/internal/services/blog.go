package services

import (
	"context"

	"github.com/jjudge-oj/api/types"
)

// BlogRepository defines persistence operations for blog posts.
type BlogRepository interface {
	List(ctx context.Context, offset, limit int, publishedOnly bool) ([]types.BlogPost, int, error)
	Get(ctx context.Context, slug string) (types.BlogPost, error)
	SlugExists(ctx context.Context, slug string, excludeID int) (bool, error)
	Create(ctx context.Context, post types.BlogPost) (types.BlogPost, error)
	Update(ctx context.Context, post types.BlogPost) (types.BlogPost, error)
	Delete(ctx context.Context, slug string) error
}

// BlogService encapsulates blog use-cases.
type BlogService struct {
	repo BlogRepository
}

func NewBlogService(repo BlogRepository) *BlogService {
	return &BlogService{repo: repo}
}

func (s *BlogService) List(ctx context.Context, offset, limit int, publishedOnly bool) ([]types.BlogPost, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, offset, limit, publishedOnly)
}

func (s *BlogService) Get(ctx context.Context, slug string) (types.BlogPost, error) {
	return s.repo.Get(ctx, slug)
}

func (s *BlogService) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	return s.repo.SlugExists(ctx, slug, excludeID)
}

func (s *BlogService) Create(ctx context.Context, post types.BlogPost) (types.BlogPost, error) {
	return s.repo.Create(ctx, post)
}

func (s *BlogService) Update(ctx context.Context, post types.BlogPost) (types.BlogPost, error) {
	return s.repo.Update(ctx, post)
}

func (s *BlogService) Delete(ctx context.Context, slug string) error {
	return s.repo.Delete(ctx, slug)
}
