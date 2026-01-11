package services

import (
	"context"

	"github.com/jjudge-oj/apiserver/types"
)

// ProblemRepository defines persistence operations for problems.
type ProblemRepository interface {
	List(ctx context.Context, offset, limit int) ([]types.Problem, int, error)
	Get(ctx context.Context, id int) (types.Problem, error)
	Create(ctx context.Context, problem types.Problem) (types.Problem, error)
	Update(ctx context.Context, problem types.Problem) (types.Problem, error)
	Delete(ctx context.Context, id int) error
}

// ProblemService encapsulates problem use-cases.
type ProblemService struct {
	repo ProblemRepository
}

func NewProblemService(repo ProblemRepository) *ProblemService {
	return &ProblemService{repo: repo}
}

func (s *ProblemService) List(ctx context.Context, offset, limit int) ([]types.Problem, int, error) {
	return s.repo.List(ctx, offset, limit)
}

func (s *ProblemService) Get(ctx context.Context, id int) (types.Problem, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProblemService) Create(ctx context.Context, problem types.Problem) (types.Problem, error) {
	return s.repo.Create(ctx, problem)
}

func (s *ProblemService) Update(ctx context.Context, problem types.Problem) (types.Problem, error) {
	return s.repo.Update(ctx, problem)
}

func (s *ProblemService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
