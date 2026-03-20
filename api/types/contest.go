package types

import "time"

// ScoringType determines how a contest computes standings.
type ScoringType string

const (
	ScoringICPC ScoringType = "icpc"
	ScoringIOI  ScoringType = "ioi"
)

// Contest holds metadata for a contest event.
type Contest struct {
	ID          int            `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	ScoringType ScoringType    `json:"scoring_type"`
	Visibility  string         `json:"visibility"`
	OwnerID     int            `json:"owner_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	// ApprovalStatus is the admin approval state: "pending", "approved", or "rejected".
	ApprovalStatus string `json:"approval_status"`
	Problems    []ContestProblem `json:"problems,omitempty"`
}

// ContestProblem is a problem entry within a contest.
type ContestProblem struct {
	ContestID int      `json:"contest_id"`
	ProblemID int      `json:"problem_id"`
	Ordinal   int      `json:"ordinal"`
	MaxPoints int      `json:"max_points"`
	Problem   *Problem `json:"problem,omitempty"`
}

// ContestRegistration records a user's registration in a contest.
type ContestRegistration struct {
	ContestID    int       `json:"contest_id"`
	UserID       int       `json:"user_id"`
	Username     string    `json:"username,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

// ContestSubmission is a submission made within a contest context.
type ContestSubmission struct {
	ID              int64            `json:"id"`
	ContestID       int              `json:"contest_id"`
	ProblemID       int              `json:"problem_id"`
	UserID          int              `json:"user_id"`
	Username        string           `json:"username,omitempty"`
	Code            string           `json:"code"`
	Language        string           `json:"language"`
	Verdict         Verdict          `json:"verdict"`
	Score           int              `json:"score"`
	CPUTime         int64            `json:"cpu_time"`
	Memory          int64            `json:"memory"`
	Message         string           `json:"message"`
	TestsPassed     int              `json:"tests_passed"`
	TestsTotal      int              `json:"tests_total"`
	TestcaseResults []TestcaseResult `json:"testcase_results"`
	SubmittedAt     time.Time        `json:"submitted_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// ContestSubmissionJob is the message queue payload for judging a contest submission.
type ContestSubmissionJob struct {
	ContestSubmission ContestSubmission `json:"contest_submission"`
	Problem           Problem           `json:"problem"`
}

// ContestProblemResult holds per-problem standing data for one user.
type ContestProblemResult struct {
	ProblemID      int  `json:"problem_id"`
	Score          int  `json:"score"`
	Accepted       bool `json:"accepted"`
	Attempts       int  `json:"attempts"`
	PenaltySeconds int  `json:"penalty_seconds"`
}

// ContestLeaderboardEntry is one row in the standings table.
type ContestLeaderboardEntry struct {
	Rank           int                       `json:"rank"`
	UserID         int                       `json:"user_id"`
	Username       string                    `json:"username"`
	TotalScore     int                       `json:"total_score"`
	PenaltySeconds int                       `json:"penalty_seconds"`
	ProblemResults map[int]ContestProblemResult `json:"problem_results"`
}
