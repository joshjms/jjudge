package services

import (
	"context"

	"github.com/jjudge-oj/apiserver/types"
)

// SubmissionRepository defines persistence operations for submissions.
type SubmissionRepository interface {
	Get(ctx context.Context, id int64) (types.Submission, error)
	Create(ctx context.Context, submission types.Submission) (types.Submission, error)
	Update(ctx context.Context, submission types.Submission) (types.Submission, error)
	Delete(ctx context.Context, id int64) error
}

// SubmissionService encapsulates submission use-cases.
type SubmissionService struct {
	repo SubmissionRepository
}

func NewSubmissionService(repo SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

func (s *SubmissionService) Get(ctx context.Context, id int64) (types.Submission, error) {
	return s.repo.Get(ctx, id)
}

func (s *SubmissionService) Create(ctx context.Context, submission types.Submission) (types.Submission, error) {
	return s.repo.Create(ctx, submission)
}

func (s *SubmissionService) Update(ctx context.Context, submission types.Submission) (types.Submission, error) {
	return s.repo.Update(ctx, submission)
}

func (s *SubmissionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
