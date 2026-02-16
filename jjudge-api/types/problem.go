package types

import "time"

// Problem represents a coding problem in the jjudge system.
// It contains metadata, constraints, and a reference to the testcases
// used for evaluating submissions.
type Problem struct {
	// ID is the unique identifier of the problem.
	ID int `json:"id" db:"id"`

	// Title is the human-readable name of the problem.
	Title string `json:"title" db:"title"`

	// Description contains the full problem statement, including
	// input/output specifications and examples.
	Description string `json:"description" db:"description"`

	// Difficulty indicates the relative difficulty level of the problem.
	// Uses Codeforces difficulty scale (800 to 3500).
	Difficulty int `json:"difficulty" db:"difficulty"`

	// TimeLimit is the maximum allowed execution time per test case,
	// expressed in milliseconds.
	TimeLimit int64 `json:"time_limit" db:"time_limit"`

	// MemoryLimit is the maximum allowed memory usage per submission,
	// expressed in bytes.
	MemoryLimit int64 `json:"memory_limit" db:"memory_limit"`

	// TestcaseGroups is the ordered list of test case groups associated with
	// this problem. Each group contains one or more test cases and contributes
	// a fixed number of points toward the final score.
	TestcaseGroups []TestcaseGroup `json:"testcase_groups" db:"testcase_groups"`

	// Tags are free-form labels associated with the problem, used for
	// categorization, filtering, and search.
	Tags []string `json:"tags" db:"tags"`

	// CreatedAt is the timestamp at which the problem was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp of the most recent update to the problem.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TestcaseGroup represents a logical grouping of test cases within a problem.
// Groups are evaluated together and may contribute a fixed number of points
// toward the final score.
type TestcaseGroup struct {
	// ID is the unique identifier of the test case group.
	ID int `json:"id" db:"id"`

	// Ordinal defines the evaluation order of this group relative to
	// other groups in the same problem.
	Ordinal int `json:"ordinal" db:"ordinal"`

	// ProblemID is the identifier of the problem this group belongs to.
	ProblemID int `json:"problem_id" db:"problem_id"`

	// Name is a human-readable name for the test case group.
	Name string `json:"name" db:"name"`

	// Testcases is the ordered list of test cases contained in this group.
	Testcases []Testcase `json:"testcases" db:"testcases"`

	// Points is the number of points awarded if all test cases in this
	// group pass successfully.
	Points int `json:"points" db:"points"`
}

// Testcase represents a single input/output pair used to evaluate a submission.
type Testcase struct {
	// ID is the unique identifier of the test case.
	ID int `json:"id" db:"id"`

	// Ordinal defines the evaluation order of this test case within its group.
	Ordinal int `json:"ordinal" db:"ordinal"`

	// TestcaseGroupID is the identifier of the group this test case belongs to.
	TestcaseGroupID int `json:"testcase_group_id" db:"testcase_group_id"`

	// Input is the input data provided to the user's program.
	Input string `json:"input" db:"input"`

	// Output is the expected output produced by a correct solution.
	Output string `json:"output" db:"output"`

	// IsHidden indicates whether this test case is hidden from users.
	// Hidden test cases are typically used to prevent hard-coded solutions.
	IsHidden bool `json:"is_hidden" db:"is_hidden"`
}
