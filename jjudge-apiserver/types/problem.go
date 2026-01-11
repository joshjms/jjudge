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

	// TestcaseBundle references the collection of test case groups used
	// to evaluate submissions. The bundle may be stored externally
	// (e.g., in an object store) and identified by a content hash.
	TestcaseBundle TestcaseBundle `json:"testcase_bundle" db:"testcase_bundle"`

	// Tags are free-form labels associated with the problem, used for
	// categorization, filtering, and search.
	Tags []string `json:"tags" db:"tags"`

	// CreatedAt is the timestamp at which the problem was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp of the most recent update to the problem.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TestcaseBundle represents a versioned collection of test case groups
// used to evaluate submissions for a problem.
//
// The bundle may be stored externally (e.g., in an object store such as MinIO)
// and referenced by ObjectKey. The SHA256 hash uniquely identifies the
// bundle contents and can be used for integrity verification and caching.
type TestcaseBundle struct {
	// ObjectKey is the identifier or path of the bundle in object storage
	// (e.g., a MinIO object key).
	ObjectKey string `json:"object_key" db:"object_key"`

	// SHA256 is the cryptographic SHA-256 hash of the bundle contents,
	// encoded as a hexadecimal string.
	SHA256 string `json:"sha256" db:"sha256"`

	// TestcaseGroups is the ordered collection of test case groups that
	// make up this bundle.
	TestcaseGroups []TestcaseGroup `json:"testcase_groups" db:"testcase_groups"`

	// Version indicates the version number of this testcase bundle.
	Version int `json:"version" db:"version"`
}

// TestcaseGroup represents a logical grouping of test cases within a problem.
// Groups are evaluated together and may contribute a fixed number of points
// toward the final score.
type TestcaseGroup struct {
	// ID is the unique identifier of the test case group.
	ID int `json:"id" db:"id"`

	// OrderID defines the evaluation order of this group relative to
	// other groups in the same problem.
	OrderID int `json:"order_id" db:"order_id"`

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

	// OrderID defines the evaluation order of this test case within its group.
	OrderID int `json:"order_id" db:"order_id"`

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
