package services

import (
	"context"
	"errors"

	"github.com/jjudge-oj/apiserver/internal/storage"
	"github.com/jjudge-oj/apiserver/internal/store"
	"github.com/jjudge-oj/apiserver/types"
)

// ProblemRepository defines persistence operations for problems.
type ProblemRepository interface {
	List(ctx context.Context, offset, limit int) ([]types.Problem, int, error)
	Get(ctx context.Context, id int) (types.Problem, error)
	Create(ctx context.Context, problem types.Problem) (types.Problem, error)
	Update(ctx context.Context, problem types.Problem) (types.Problem, error)
	Delete(ctx context.Context, id int) error
	GetLatestTestcaseBundle(ctx context.Context, problemID int) (types.TestcaseBundle, error)
	AddTestcaseBundleVersion(ctx context.Context, problemID int, bundle types.TestcaseBundle) error
}

// ProblemService encapsulates problem use-cases.
type ProblemService struct {
	repo    ProblemRepository
	storage storage.Storage
}

func NewProblemService(repo ProblemRepository) *ProblemService {
	return &ProblemService{repo: repo}
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

func (s *ProblemService) Create(ctx context.Context, problem types.Problem) (types.Problem, error) {
	if problem.TestcaseBundle.Version == 0 {
		problem.TestcaseBundle.Version = 1
	}
	return s.repo.Create(ctx, problem)
}

func (s *ProblemService) Update(ctx context.Context, problem types.Problem) (types.Problem, error) {
	return s.repo.Update(ctx, problem)
}

func (s *ProblemService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProblemService) UpdateTestcaseBundle(ctx context.Context, problemID int, bundle types.TestcaseBundle) error {
	current, err := s.repo.GetLatestTestcaseBundle(ctx, problemID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		problem, fetchErr := s.repo.Get(ctx, problemID)
		if fetchErr != nil {
			return fetchErr
		}
		current = problem.TestcaseBundle
	}

	if err == nil && current.SHA256 != "" && current.SHA256 == bundle.SHA256 {
		return nil
	}

	if current.Version == 0 {
		bundle.Version = 1
	} else {
		bundle.Version = current.Version + 1
	}

	return s.repo.AddTestcaseBundleVersion(ctx, problemID, bundle)
}
