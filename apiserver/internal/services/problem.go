package services

import (
	"context"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/storage"
)

// ProblemRepository defines persistence operations for problems.
type ProblemRepository interface {
	List(ctx context.Context, offset, limit int, callerID int, isAdmin bool) ([]types.Problem, int, error)
	ListPending(ctx context.Context, offset, limit int) ([]types.Problem, int, error)
	ListByCreator(ctx context.Context, creatorID, offset, limit int) ([]types.Problem, int, error)
	Get(ctx context.Context, id int) (types.Problem, error)
	GetWithTestcases(ctx context.Context, id int) (types.Problem, error)
	Create(ctx context.Context, problem types.Problem) (types.Problem, error)
	Update(ctx context.Context, problem types.Problem) (types.Problem, error)
	Delete(ctx context.Context, id int) error
	SaveTestcaseGroups(ctx context.Context, problemID int, groups []types.TestcaseGroup) error
	Approve(ctx context.Context, id int) error
	Reject(ctx context.Context, id int) error
}

// ProblemService encapsulates problem use-cases.
type ProblemService struct {
	repo    ProblemRepository
	storage *storage.Storage
}

func NewProblemService(repo ProblemRepository, storageClient *storage.Storage) *ProblemService {
	return &ProblemService{repo: repo, storage: storageClient}
}

func (s *ProblemService) List(ctx context.Context, offset, limit int, callerID int, isAdmin bool) ([]types.Problem, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, offset, limit, callerID, isAdmin)
}

func (s *ProblemService) ListPending(ctx context.Context, offset, limit int) ([]types.Problem, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListPending(ctx, offset, limit)
}

func (s *ProblemService) ListByCreator(ctx context.Context, creatorID, offset, limit int) ([]types.Problem, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListByCreator(ctx, creatorID, offset, limit)
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

func (s *ProblemService) Approve(ctx context.Context, id int) error {
	return s.repo.Approve(ctx, id)
}

func (s *ProblemService) Reject(ctx context.Context, id int) error {
	return s.repo.Reject(ctx, id)
}
