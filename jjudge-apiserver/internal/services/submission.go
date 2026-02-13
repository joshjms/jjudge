package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/mq"
	"github.com/jjudge-oj/apiserver/internal/storage"
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
	repo    SubmissionRepository
	storage *storage.Storage
	mq      *mq.MQ
}

func NewSubmissionService(repo SubmissionRepository, storageClient *storage.Storage, mqClient *mq.MQ) *SubmissionService {
	return &SubmissionService{repo: repo, storage: storageClient, mq: mqClient}
}

func (s *SubmissionService) Get(ctx context.Context, id int64) (types.Submission, error) {
	return s.repo.Get(ctx, id)
}

func (s *SubmissionService) Create(ctx context.Context, submission types.Submission) (types.Submission, error) {
	return s.repo.Create(ctx, submission)
}

func (s *SubmissionService) CreateAndEnqueue(ctx context.Context, submission types.Submission, problem types.Problem) (types.Submission, string, error) {
	if s.storage == nil {
		return types.Submission{}, "", errors.New("object storage is not configured")
	}
	if s.mq == nil {
		return types.Submission{}, "", errors.New("message queue is not configured")
	}

	submission.Code = strings.TrimSpace(submission.Code)
	if submission.Code == "" {
		return types.Submission{}, "", errors.New("source code is required")
	}

	created, err := s.repo.Create(ctx, submission)
	if err != nil {
		return types.Submission{}, "", err
	}

	artifactKey, err := s.uploadSource(ctx, created)
	if err != nil {
		_ = s.repo.Delete(ctx, int64(created.ID))
		return types.Submission{}, "", err
	}

	job := types.SubmissionJob{
		Submission: created,
		Problem:    problem,
	}
	payload, err := json.Marshal(job)
	if err != nil {
		_ = s.storage.Delete(ctx, artifactKey)
		_ = s.repo.Delete(ctx, int64(created.ID))
		return types.Submission{}, "", err
	}

	attrs := map[string]string{
		"submission_id": strconv.Itoa(created.ID),
		"problem_id":    strconv.Itoa(created.ProblemID),
		"user_id":       strconv.Itoa(created.UserID),
	}
	if _, err := s.mq.Publish(ctx, submissionQueue, payload, attrs); err != nil {
		_ = s.storage.Delete(ctx, artifactKey)
		_ = s.repo.Delete(ctx, int64(created.ID))
		return types.Submission{}, "", err
	}

	return created, artifactKey, nil
}

func (s *SubmissionService) Update(ctx context.Context, submission types.Submission) (types.Submission, error) {
	return s.repo.Update(ctx, submission)
}

func (s *SubmissionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

const submissionQueue = "submissions"

func (s *SubmissionService) uploadSource(ctx context.Context, submission types.Submission) (string, error) {
	codeBytes := []byte(submission.Code)
	hash := sha256.Sum256(codeBytes)
	digest := hex.EncodeToString(hash[:])
	objectKey := fmt.Sprintf("submissions/%d/source-%s.txt", submission.ID, digest)

	if err := s.storage.Put(ctx, objectKey, bytes.NewReader(codeBytes), int64(len(codeBytes)), "text/plain; charset=utf-8"); err != nil {
		return "", fmt.Errorf("failed to upload submission source: %w", err)
	}
	return objectKey, nil
}
