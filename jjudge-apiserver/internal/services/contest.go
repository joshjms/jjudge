package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/internal/mq"
	"github.com/jjudge-oj/apiserver/internal/storage"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// ContestRepository defines the persistence interface used by ContestService.
type ContestRepository interface {
	ListContests(ctx context.Context, offset, limit int) ([]types.Contest, int, error)
	GetContest(ctx context.Context, id int) (types.Contest, error)
	GetContestWithProblems(ctx context.Context, id int) (types.Contest, error)
	CreateContest(ctx context.Context, c types.Contest) (types.Contest, error)
	UpdateContest(ctx context.Context, c types.Contest) (types.Contest, error)
	DeleteContest(ctx context.Context, id int) error

	ListContestProblems(ctx context.Context, contestID int) ([]types.ContestProblem, error)
	AddContestProblem(ctx context.Context, cp types.ContestProblem) (types.ContestProblem, error)
	RemoveContestProblem(ctx context.Context, contestID, problemID int) error
	ReorderContestProblems(ctx context.Context, contestID int, ordinals map[int]int) error

	Register(ctx context.Context, contestID, userID int) error
	Unregister(ctx context.Context, contestID, userID int) error
	IsRegistered(ctx context.Context, contestID, userID int) (bool, error)
	ListRegistrations(ctx context.Context, contestID int) ([]types.ContestRegistration, error)

	CreateContestSubmission(ctx context.Context, cs types.ContestSubmission) (types.ContestSubmission, error)
	GetContestSubmission(ctx context.Context, id int64) (types.ContestSubmission, error)
	UpdateContestSubmission(ctx context.Context, cs types.ContestSubmission) (types.ContestSubmission, error)
	ListContestSubmissions(ctx context.Context, contestID, problemID, userID int) ([]types.ContestSubmission, error)
	DeleteContestSubmission(ctx context.Context, id int64) error

	GetLeaderboardRows(ctx context.Context, contestID int) ([]store.LeaderboardRow, error)
	ListSubmissionsForContestProblem(ctx context.Context, contestID, problemID int) ([]types.ContestSubmission, error)
}

const contestSubmissionQueue = "contest-submissions"

// ContestService encapsulates contest use-cases.
type ContestService struct {
	repo    ContestRepository
	storage *storage.Storage
	mq      *mq.MQ
}

func NewContestService(repo ContestRepository, storageClient *storage.Storage, mqClient *mq.MQ) *ContestService {
	return &ContestService{repo: repo, storage: storageClient, mq: mqClient}
}

// ---------- Contest CRUD ----------

func (s *ContestService) ListContests(ctx context.Context, offset, limit int) ([]types.Contest, int, error) {
	return s.repo.ListContests(ctx, offset, limit)
}

func (s *ContestService) GetContest(ctx context.Context, id int) (types.Contest, error) {
	return s.repo.GetContest(ctx, id)
}

func (s *ContestService) GetContestWithProblems(ctx context.Context, id int) (types.Contest, error) {
	return s.repo.GetContestWithProblems(ctx, id)
}

func (s *ContestService) CreateContest(ctx context.Context, c types.Contest) (types.Contest, error) {
	return s.repo.CreateContest(ctx, c)
}

func (s *ContestService) UpdateContest(ctx context.Context, c types.Contest) (types.Contest, error) {
	return s.repo.UpdateContest(ctx, c)
}

func (s *ContestService) DeleteContest(ctx context.Context, id int) error {
	return s.repo.DeleteContest(ctx, id)
}

// ---------- Contest Problems ----------

func (s *ContestService) ListContestProblems(ctx context.Context, contestID int) ([]types.ContestProblem, error) {
	return s.repo.ListContestProblems(ctx, contestID)
}

func (s *ContestService) AddContestProblem(ctx context.Context, cp types.ContestProblem) (types.ContestProblem, error) {
	return s.repo.AddContestProblem(ctx, cp)
}

func (s *ContestService) RemoveContestProblem(ctx context.Context, contestID, problemID int) error {
	return s.repo.RemoveContestProblem(ctx, contestID, problemID)
}

func (s *ContestService) ReorderContestProblems(ctx context.Context, contestID int, ordinals map[int]int) error {
	return s.repo.ReorderContestProblems(ctx, contestID, ordinals)
}

// ---------- Registrations ----------

func (s *ContestService) Register(ctx context.Context, contestID, userID int) error {
	return s.repo.Register(ctx, contestID, userID)
}

func (s *ContestService) Unregister(ctx context.Context, contestID, userID int) error {
	return s.repo.Unregister(ctx, contestID, userID)
}

func (s *ContestService) IsRegistered(ctx context.Context, contestID, userID int) (bool, error) {
	return s.repo.IsRegistered(ctx, contestID, userID)
}

func (s *ContestService) ListRegistrations(ctx context.Context, contestID int) ([]types.ContestRegistration, error) {
	return s.repo.ListRegistrations(ctx, contestID)
}

// ---------- Contest Submissions ----------

func (s *ContestService) GetContestSubmission(ctx context.Context, id int64) (types.ContestSubmission, error) {
	return s.repo.GetContestSubmission(ctx, id)
}

func (s *ContestService) ListContestSubmissions(ctx context.Context, contestID, problemID, userID int) ([]types.ContestSubmission, error) {
	return s.repo.ListContestSubmissions(ctx, contestID, problemID, userID)
}

func (s *ContestService) UpdateContestSubmission(ctx context.Context, cs types.ContestSubmission) (types.ContestSubmission, error) {
	return s.repo.UpdateContestSubmission(ctx, cs)
}

// CreateAndEnqueueContestSubmission validates eligibility, persists the submission, uploads the
// source artifact, and publishes a job to the contest-submissions queue.
func (s *ContestService) CreateAndEnqueueContestSubmission(
	ctx context.Context,
	cs types.ContestSubmission,
	problem types.Problem,
	contest types.Contest,
) (types.ContestSubmission, string, error) {
	if s.storage == nil {
		return types.ContestSubmission{}, "", errors.New("object storage is not configured")
	}
	if s.mq == nil {
		return types.ContestSubmission{}, "", errors.New("message queue is not configured")
	}

	cs.Code = strings.TrimSpace(cs.Code)
	if cs.Code == "" {
		return types.ContestSubmission{}, "", errors.New("source code is required")
	}

	// Check registration
	registered, err := s.repo.IsRegistered(ctx, cs.ContestID, cs.UserID)
	if err != nil {
		return types.ContestSubmission{}, "", fmt.Errorf("check registration: %w", err)
	}
	if !registered {
		return types.ContestSubmission{}, "", ErrNotRegistered
	}

	// Validate contest window
	now := time.Now()
	if now.Before(contest.StartTime) || now.After(contest.EndTime) {
		return types.ContestSubmission{}, "", ErrContestNotActive
	}

	created, err := s.repo.CreateContestSubmission(ctx, cs)
	if err != nil {
		return types.ContestSubmission{}, "", err
	}

	artifactKey, err := s.uploadContestSource(ctx, created)
	if err != nil {
		_ = s.repo.DeleteContestSubmission(ctx, created.ID)
		return types.ContestSubmission{}, "", err
	}

	job := types.ContestSubmissionJob{
		ContestSubmission: created,
		Problem:           problem,
	}
	payload, err := json.Marshal(job)
	if err != nil {
		_ = s.storage.Delete(ctx, artifactKey)
		_ = s.repo.DeleteContestSubmission(ctx, created.ID)
		return types.ContestSubmission{}, "", err
	}

	attrs := map[string]string{
		"contest_submission_id": strconv.FormatInt(created.ID, 10),
		"contest_id":            strconv.Itoa(created.ContestID),
		"problem_id":            strconv.Itoa(created.ProblemID),
		"user_id":               strconv.Itoa(created.UserID),
	}
	if _, err := s.mq.Publish(ctx, contestSubmissionQueue, payload, attrs); err != nil {
		_ = s.storage.Delete(ctx, artifactKey)
		_ = s.repo.DeleteContestSubmission(ctx, created.ID)
		return types.ContestSubmission{}, "", err
	}

	return created, artifactKey, nil
}

// RejudgeContestProblem resets all submissions for a contest+problem to PENDING and re-enqueues them.
func (s *ContestService) RejudgeContestProblem(ctx context.Context, contestID, problemID int, problem types.Problem, contest types.Contest) error {
	submissions, err := s.repo.ListSubmissionsForContestProblem(ctx, contestID, problemID)
	if err != nil {
		return err
	}

	for _, sub := range submissions {
		sub.Verdict = types.VerdictPending
		sub.Score = 0
		sub.CPUTime = 0
		sub.Memory = 0
		sub.Message = ""
		sub.TestsPassed = 0
		sub.TestsTotal = 0
		sub.TestcaseResults = nil

		updated, err := s.repo.UpdateContestSubmission(ctx, sub)
		if err != nil {
			return fmt.Errorf("reset submission %d: %w", sub.ID, err)
		}

		job := types.ContestSubmissionJob{
			ContestSubmission: updated,
			Problem:           problem,
		}
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}

		attrs := map[string]string{
			"contest_submission_id": strconv.FormatInt(updated.ID, 10),
			"contest_id":            strconv.Itoa(contestID),
			"problem_id":            strconv.Itoa(problemID),
			"user_id":               strconv.Itoa(updated.UserID),
		}
		if _, err := s.mq.Publish(ctx, contestSubmissionQueue, payload, attrs); err != nil {
			return fmt.Errorf("re-enqueue submission %d: %w", updated.ID, err)
		}
	}
	return nil
}

// ---------- Leaderboard ----------

// GetLeaderboard computes standings for a contest.
func (s *ContestService) GetLeaderboard(ctx context.Context, contest types.Contest) ([]types.ContestLeaderboardEntry, error) {
	rows, err := s.repo.GetLeaderboardRows(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	// Group rows by user
	type userKey = int
	type perUser struct {
		username string
		problems map[int]*store.LeaderboardRow
	}
	users := map[userKey]*perUser{}

	for i := range rows {
		row := &rows[i]
		u, ok := users[row.UserID]
		if !ok {
			u = &perUser{username: row.Username, problems: map[int]*store.LeaderboardRow{}}
			users[row.UserID] = u
		}
		u.problems[row.ProblemID] = row
	}

	entries := make([]types.ContestLeaderboardEntry, 0, len(users))
	for userID, u := range users {
		entry := types.ContestLeaderboardEntry{
			UserID:         userID,
			Username:       u.username,
			ProblemResults: make(map[int]types.ContestProblemResult, len(u.problems)),
		}

		for probID, row := range u.problems {
			pr := types.ContestProblemResult{
				ProblemID: probID,
				Score:     row.BestScore,
				Accepted:  row.Accepted,
				Attempts:  int(row.Attempts),
			}

			switch contest.ScoringType {
			case types.ScoringICPC:
				if row.Accepted && row.AcceptSeconds != nil {
					// penalty = accept_seconds + 20*60*(attempts-1)
					penalty := int(*row.AcceptSeconds) + 20*60*(int(row.Attempts)-1)
					pr.PenaltySeconds = penalty
					entry.PenaltySeconds += penalty
				}
				if row.Accepted {
					entry.TotalScore++
				}
			case types.ScoringIOI:
				entry.TotalScore += row.BestScore
			}

			entry.ProblemResults[probID] = pr
		}

		entries = append(entries, entry)
	}

	// Sort
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		switch contest.ScoringType {
		case types.ScoringICPC:
			if a.TotalScore != b.TotalScore {
				return a.TotalScore > b.TotalScore
			}
			return a.PenaltySeconds < b.PenaltySeconds
		default: // IOI
			return a.TotalScore > b.TotalScore
		}
	})

	// Assign ranks (ties share same rank)
	for i := range entries {
		if i == 0 {
			entries[i].Rank = 1
			continue
		}
		prev := entries[i-1]
		cur := entries[i]
		sameRank := cur.TotalScore == prev.TotalScore
		if contest.ScoringType == types.ScoringICPC {
			sameRank = sameRank && cur.PenaltySeconds == prev.PenaltySeconds
		}
		if sameRank {
			entries[i].Rank = prev.Rank
		} else {
			entries[i].Rank = i + 1
		}
	}

	return entries, nil
}

// ---------- Helpers ----------

func (s *ContestService) uploadContestSource(ctx context.Context, cs types.ContestSubmission) (string, error) {
	codeBytes := []byte(cs.Code)
	hash := sha256.Sum256(codeBytes)
	digest := hex.EncodeToString(hash[:])
	objectKey := fmt.Sprintf("contest-submissions/%d/source-%s.txt", cs.ID, digest)

	if err := s.storage.Put(ctx, objectKey, bytes.NewReader(codeBytes), int64(len(codeBytes)), "text/plain; charset=utf-8"); err != nil {
		return "", fmt.Errorf("failed to upload contest submission source: %w", err)
	}
	return objectKey, nil
}

// ErrNotRegistered is returned when a user is not registered for a contest.
var ErrNotRegistered = errors.New("not registered for contest")

// ErrContestNotActive is returned when a submission is attempted outside the contest window.
var ErrContestNotActive = errors.New("contest is not currently active")
