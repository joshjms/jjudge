package types

import (
	"encoding/json"
	"time"
)

// Submission represents a user's submission to a problem.
// It contains source code, execution metadata, and the final judging outcome.
type Submission struct {
	// ID is the unique identifier of the submission.
	ID int `json:"id" db:"id"`

	// ProblemID identifies the problem this submission is for.
	ProblemID int `json:"problem_id" db:"problem_id"`

	// UserID identifies the user who made the submission.
	UserID int `json:"user_id" db:"user_id"`

	// Code is the source code submitted by the user.
	Code string `json:"code" db:"code"`

	// Language is the identifier of the programming language used.
	Language string `json:"language" db:"language"`

	// Verdict is the final outcome of judging the submission.
	Verdict Verdict `json:"verdict" db:"verdict"`

	// Score is the total score awarded for this submission.
	Score int `json:"score" db:"score"`

	// CPUTime is the total CPU time consumed by the submission,
	// expressed in milliseconds.
	CPUTime int64 `json:"cpu_time" db:"cpu_time"`

	// Memory is the peak memory usage of the submission,
	// expressed in bytes.
	Memory int64 `json:"memory" db:"memory"`

	// Message contains additional information about the verdict,
	// such as compilation errors or system messages.
	Message string `json:"message" db:"message"`

	// TestsPassed is the number of test cases successfully passed.
	TestsPassed int `json:"tests_passed" db:"tests_passed"`

	// TestsTotal is the total number of test cases executed.
	TestsTotal int `json:"tests_total" db:"tests_total"`

	// CreatedAt is the timestamp when the submission was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp when the submission was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// TestcaseResults holds per-test-case execution results when available.
	// This field may be omitted for summary or list views.
	TestcaseResults []TestcaseResult `json:"testcase_results" db:"testcase_results"`
}

// TestcaseResult represents the result of executing a single test case
// as part of judging a submission.
type TestcaseResult struct {
	// SubmissionID identifies the submission this result belongs to.
	SubmissionID int64 `json:"submission_id" db:"submission_id"`

	// TestcaseID identifies the test case that was executed.
	TestcaseID int `json:"testcase_id" db:"testcase_id"`

	// Verdict is the outcome of this specific test case.
	Verdict Verdict `json:"verdict" db:"verdict"`

	// CPUTime is the CPU time consumed by this test case,
	// expressed in milliseconds.
	CPUTime int64 `json:"cpu_time" db:"cpu_time"`

	// Memory is the peak memory usage for this test case,
	// expressed in bytes.
	Memory int64 `json:"memory" db:"memory"`

	// Input is the input provided to the program for this test case.
	// This field is omitted when the test case is hidden.
	Input string `json:"input,omitempty" db:"input,omitempty"`

	// ExpectedOutput is the correct output expected for this test case.
	// This field is omitted when the test case is hidden.
	ExpectedOutput string `json:"expected_output,omitempty" db:"expected_output,omitempty"`

	// ActualOutput is the output produced by the user's program.
	// This field is omitted when the test case is hidden.
	ActualOutput string `json:"actual_output,omitempty" db:"actual_output,omitempty"`

	// ErrorMessage contains runtime or system error messages, if any.
	ErrorMessage string `json:"error_message,omitempty" db:"error_message,omitempty"`
}

// Language represents a supported programming language configuration
// used by the judge system.
type Language struct {
	// Name is the human-readable name of the language.
	Name string `json:"name"`

	// Extension is the default file extension for source files.
	Extension string `json:"extension"`

	// CompileCommand is the command used to compile source code.
	// This may be empty for interpreted languages.
	CompileCommand string `json:"compile_command"`

	// ExecuteCommand is the command used to execute the compiled
	// or interpreted program.
	ExecuteCommand string `json:"execute_command"`

	// Version indicates the compiler or interpreter version.
	Version string `json:"version"`

	// TimeMultiplier is a factor applied to time limits for this language.
	TimeMultiplier float64 `json:"time_multiplier"`

	// MemoryMultiplier is a factor applied to memory limits for this language.
	MemoryMultiplier float64 `json:"memory_multiplier"`
}

// Verdict represents the outcome of judging a submission or test case.
type Verdict int

// Supported verdict values.
const (
	// VerdictPending indicates the submission has been received
	// but has not started judging yet.
	VerdictPending Verdict = iota

	// VerdictJudging indicates the submission is currently being judged.
	VerdictJudging

	// VerdictAccepted indicates the submission passed all test cases.
	VerdictAccepted

	// VerdictWrongAnswer indicates the submission produced incorrect output.
	VerdictWrongAnswer

	// VerdictTimeLimitExceeded indicates the submission exceeded the time limit.
	VerdictTimeLimitExceeded

	// VerdictMemoryLimitExceeded indicates the submission exceeded the memory limit.
	VerdictMemoryLimitExceeded

	// VerdictRuntimeError indicates a runtime error occurred during execution.
	VerdictRuntimeError

	// VerdictCompilationError indicates the submission failed to compile.
	VerdictCompilationError

	// VerdictSystemError indicates an internal system failure occurred.
	VerdictSystemError

	// VerdictInternalError indicates an unexpected internal error.
	VerdictInternalError

	// VerdictSkipped indicates the submission or test case was skipped.
	VerdictSkipped
)

// String returns the compact string representation of the verdict
// used in API responses and logs.
func (v Verdict) String() string {
	switch v {
	case VerdictPending:
		return "PENDING"
	case VerdictJudging:
		return "JUDGING"
	case VerdictAccepted:
		return "AC"
	case VerdictWrongAnswer:
		return "WA"
	case VerdictTimeLimitExceeded:
		return "TLE"
	case VerdictMemoryLimitExceeded:
		return "MLE"
	case VerdictRuntimeError:
		return "RE"
	case VerdictCompilationError:
		return "CE"
	case VerdictSystemError:
		return "SE"
	case VerdictInternalError:
		return "IE"
	case VerdictSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

func (v Verdict) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}
