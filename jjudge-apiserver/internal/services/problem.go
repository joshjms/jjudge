package services

import (
	"context"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/storage"
)

// ProblemRepository defines persistence operations for problems.
type ProblemRepository interface {
	List(ctx context.Context, offset, limit int) ([]types.Problem, int, error)
	Get(ctx context.Context, id int) (types.Problem, error)
	GetWithTestcases(ctx context.Context, id int) (types.Problem, error)
	Create(ctx context.Context, problem types.Problem) (types.Problem, error)
	Update(ctx context.Context, problem types.Problem) (types.Problem, error)
	Delete(ctx context.Context, id int) error
	SaveTestcaseGroups(ctx context.Context, problemID int, groups []types.TestcaseGroup) error
}

// ProblemService encapsulates problem use-cases.
type ProblemService struct {
	repo    ProblemRepository
	storage *storage.Storage
}

func NewProblemService(repo ProblemRepository, storageClient *storage.Storage) *ProblemService {
	return &ProblemService{repo: repo, storage: storageClient}
}

func (s *ProblemService) List(ctx context.Context, offset, limit int) ([]types.Problem, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, offset, limit)
}

func (s *ProblemService) Get(ctx context.Context, id int) (types.Problem, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProblemService) GetWithTestcases(ctx context.Context, id int) (types.Problem, error) {
	return s.repo.GetWithTestcases(ctx, id)
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

func (s *ProblemService) SaveTestcaseGroups(ctx context.Context, problemID int, groups []types.TestcaseGroup) error {
	return s.repo.SaveTestcaseGroups(ctx, problemID, groups)
}
