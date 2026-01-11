package services

import (
	"context"

	"github.com/jjudge-oj/apiserver/types"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	GetByID(ctx context.Context, id int) (types.User, error)
	GetByUsername(ctx context.Context, username string) (types.User, error)
	Create(ctx context.Context, user types.User) (types.User, error)
	Update(ctx context.Context, user types.User) (types.User, error)
	Delete(ctx context.Context, id int) error
}

// UserService encapsulates user use-cases.
type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id int) (types.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) GetByUsername(ctx context.Context, username string) (types.User, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *UserService) Create(ctx context.Context, user types.User) (types.User, error) {
	return s.repo.Create(ctx, user)
}

func (s *UserService) Update(ctx context.Context, user types.User) (types.User, error) {
	return s.repo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
