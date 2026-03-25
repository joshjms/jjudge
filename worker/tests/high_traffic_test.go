package tests_test

// High-traffic e2e tests.
//
// These tests fire 50 concurrent submissions through the full lime pipeline and
// verify correctness, isolation, and slot-pool queuing under load.
//
// Requirements (same as the other integration tests):
//   LIME_INTEGRATION=1
//   LIME_CGROUP_ROOT
//   lime binary in PATH
//   run as root (uid 0)

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jjudge-oj/worker/internal/lime"
)

const (
	highTrafficN       = 50
	highTrafficUIDBase = 61000 // 61000 – 61049; does not overlap with other tests
	highTrafficTimeUs  = 5_000_000
	highTrafficMemory  = 256 * 1024 * 1024
)

// newHighTrafficPool creates a slot pool with min(NumCPU, highTrafficN) slots
// so that the test exercises both immediate dispatch and queuing.
func newHighTrafficPool() *lime.SlotPool {
	n := runtime.NumCPU()
	if n > highTrafficN {
		n = highTrafficN
	}
	cpus := make([]string, n)
	for i := range cpus {
		cpus[i] = strconv.Itoa(i)
	}
	return lime.NewSlotPool(
		lime.WithSlotUIDs(highTrafficUIDBase),
		lime.WithCPUs(strings.Join(cpus, ",")),
	)
}

type submitResult struct {
	idx      int
	report   *lime.Report
	err      error
	duration time.Duration
}

// runConcurrent launches n goroutines, each calling fn(idx), and returns all
// results once every goroutine has finished.
func runConcurrent(n int, fn func(idx int) submitResult) []submitResult {
	results := make([]submitResult, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = fn(idx)
		}(i)
	}
	wg.Wait()
	return results
}

// writePythonAdd writes an add.py to dir and returns the path.
func writePythonAdd(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "add.py")
	src := "a, b = map(int, input().split())\nprint(a + b)\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write add.py: %v", err)
	}
	return path
}

// checkResults fails the test if any submission returned an error or a
// non-zero exit code, and logs a summary of timings.
func checkResults(t *testing.T, results []submitResult) {
	t.Helper()
	var total time.Duration
	failures := 0
	for _, r := range results {
		total += r.duration
		if r.err != nil {
			t.Errorf("submission %d: lime error: %v", r.idx, r.err)
			failures++
			continue
		}
		if r.report.ExitCode != 0 || r.report.Signal != 0 {
			t.Errorf("submission %d: exitCode=%d signal=%d stderr=%q",
				r.idx, r.report.ExitCode, r.report.Signal, r.report.Stderr)
			failures++
		}
	}
	n := len(results)
	t.Logf("%d submissions: %d ok, %d failed, avg wall time %v",
		n, n-failures, failures, total/time.Duration(n))
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestHighTrafficConcurrentCpp compiles add.cpp once then runs 50 copies
// concurrently, each with a unique input, and asserts every output is correct.
func TestHighTrafficConcurrentCpp(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newHighTrafficPool()

	// Compile once into a shared work dir; each submission gets its own copy.
	srcDir := compileBinary(t, pool, "testdata/add.cpp")
	binaryPath := filepath.Join(srcDir, "solution")

	results := runConcurrent(highTrafficN, func(idx int) submitResult {
		start := time.Now()

		workDir := t.TempDir()
		dst := filepath.Join(workDir, "solution")
		copyFile(t, binaryPath, dst)
		if err := os.Chmod(dst, 0755); err != nil {
			return submitResult{idx: idx, err: fmt.Errorf("chmod: %w", err)}
		}

		a, b := idx+1, idx*2+1
		stdin := fmt.Sprintf("%d %d\n", a, b)
		want := fmt.Sprintf("%d\n", a+b)

		report, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			[]string{"/work/solution"}, stdin,
			highTrafficTimeUs, highTrafficMemory, 1, true,
		)
		if err != nil {
			return submitResult{idx: idx, err: err, duration: time.Since(start)}
		}
		r := submitResult{idx: idx, report: report, duration: time.Since(start)}
		if report.Stdout != want {
			t.Errorf("submission %d: stdout=%q want=%q", idx, report.Stdout, want)
		}
		return r
	})

	checkResults(t, results)
}

// TestHighTrafficConcurrentPython runs 50 Python add.py submissions
// concurrently, each with unique input, and asserts every output is correct.
func TestHighTrafficConcurrentPython(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newHighTrafficPool()

	results := runConcurrent(highTrafficN, func(idx int) submitResult {
		start := time.Now()

		workDir := t.TempDir()
		writePythonAdd(t, workDir)

		a, b := idx+1, idx*2+1
		stdin := fmt.Sprintf("%d %d\n", a, b)
		want := fmt.Sprintf("%d\n", a+b)

		report, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			[]string{"/usr/bin/python3", "/work/add.py"}, stdin,
			highTrafficTimeUs, highTrafficMemory, 1, true,
		)
		if err != nil {
			return submitResult{idx: idx, err: err, duration: time.Since(start)}
		}
		r := submitResult{idx: idx, report: report, duration: time.Since(start)}
		if report.Stdout != want {
			t.Errorf("submission %d: stdout=%q want=%q", idx, report.Stdout, want)
		}
		return r
	})

	checkResults(t, results)
}

// TestHighTrafficNoOutputContamination runs 50 submissions where each has a
// unique expected output. Any cross-contamination between sandboxes would
// produce a wrong answer that the per-submission check would catch.
func TestHighTrafficNoOutputContamination(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newHighTrafficPool()
	srcDir := compileBinary(t, pool, "testdata/add.cpp")
	binaryPath := filepath.Join(srcDir, "solution")

	// Use inputs that produce outputs spread across a large range so any
	// bleed-over between sandbox outputs is immediately detectable.
	results := runConcurrent(highTrafficN, func(idx int) submitResult {
		start := time.Now()

		workDir := t.TempDir()
		dst := filepath.Join(workDir, "solution")
		copyFile(t, binaryPath, dst)
		if err := os.Chmod(dst, 0755); err != nil {
			return submitResult{idx: idx, err: fmt.Errorf("chmod: %w", err)}
		}

		// Each submission computes idx * 1_000_000 + 1 so outputs are
		// globally unique and collisions are obvious.
		a := idx * 1_000_000
		b := 1
		stdin := fmt.Sprintf("%d %d\n", a, b)
		want := fmt.Sprintf("%d\n", a+b)

		report, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			[]string{"/work/solution"}, stdin,
			highTrafficTimeUs, highTrafficMemory, 1, true,
		)
		if err != nil {
			return submitResult{idx: idx, err: err, duration: time.Since(start)}
		}
		r := submitResult{idx: idx, report: report, duration: time.Since(start)}
		if report.Stdout != want {
			t.Errorf("contamination detected at submission %d: stdout=%q want=%q",
				idx, report.Stdout, want)
		}
		return r
	})

	checkResults(t, results)
}

// TestHighTrafficSlotQueuing explicitly uses a pool with fewer slots than
// submissions to verify that submissions queue and complete correctly without
// deadlock or slot leaks.
func TestHighTrafficSlotQueuing(t *testing.T) {
	skipUnlessIntegration(t)

	// 4 slots, 50 submissions → 46 must queue.
	const slots = 4
	pool := lime.NewSlotPool(
		lime.WithSlotUIDs(highTrafficUIDBase+100),
		lime.WithCPUs("0,1,2,3"),
	)

	srcDir := compileBinary(t, pool, "testdata/add.cpp")
	binaryPath := filepath.Join(srcDir, "solution")

	start := time.Now()
	results := runConcurrent(highTrafficN, func(idx int) submitResult {
		t0 := time.Now()

		workDir := t.TempDir()
		dst := filepath.Join(workDir, "solution")
		copyFile(t, binaryPath, dst)
		if err := os.Chmod(dst, 0755); err != nil {
			return submitResult{idx: idx, err: fmt.Errorf("chmod: %w", err)}
		}

		stdin := fmt.Sprintf("%d %d\n", idx, idx)
		want := fmt.Sprintf("%d\n", idx*2)

		report, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			[]string{"/work/solution"}, stdin,
			highTrafficTimeUs, highTrafficMemory, 1, true,
		)
		if err != nil {
			return submitResult{idx: idx, err: err, duration: time.Since(t0)}
		}
		r := submitResult{idx: idx, report: report, duration: time.Since(t0)}
		if report.Stdout != want {
			t.Errorf("submission %d: stdout=%q want=%q", idx, report.Stdout, want)
		}
		return r
	})

	wallTime := time.Since(start)
	t.Logf("all %d submissions through %d slots in %v", highTrafficN, slots, wallTime)

	checkResults(t, results)

	// Verify the pool has exactly `slots` slots available again (no leaks).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	drained := 0
	for drained < slots {
		a, err := pool.Allocate(ctx)
		if err != nil {
			break
		}
		drained++
		a.Release()
	}
	if drained != slots {
		t.Errorf("slot pool has %d slots after run, want %d — possible slot leak", drained, slots)
	}
}

// TestHighTrafficMixed runs 50 submissions split evenly between C++ and Python,
// all concurrently, asserting every submission produces correct output.
func TestHighTrafficMixed(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newHighTrafficPool()
	srcDir := compileBinary(t, pool, "testdata/add.cpp")
	binaryPath := filepath.Join(srcDir, "solution")

	results := runConcurrent(highTrafficN, func(idx int) submitResult {
		start := time.Now()

		workDir := t.TempDir()
		a, b := idx+1, idx+2
		stdin := fmt.Sprintf("%d %d\n", a, b)
		want := fmt.Sprintf("%d\n", a+b)

		var args []string
		if idx%2 == 0 {
			// C++
			dst := filepath.Join(workDir, "solution")
			copyFile(t, binaryPath, dst)
			if err := os.Chmod(dst, 0755); err != nil {
				return submitResult{idx: idx, err: fmt.Errorf("chmod: %w", err)}
			}
			args = []string{"/work/solution"}
		} else {
			// Python
			writePythonAdd(t, workDir)
			args = []string{"/usr/bin/python3", "/work/add.py"}
		}

		report, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			args, stdin,
			highTrafficTimeUs, highTrafficMemory, 1, true,
		)
		if err != nil {
			return submitResult{idx: idx, err: err, duration: time.Since(start)}
		}
		r := submitResult{idx: idx, report: report, duration: time.Since(start)}
		if report.Stdout != want {
			lang := "cpp"
			if idx%2 != 0 {
				lang = "python"
			}
			t.Errorf("submission %d (%s): stdout=%q want=%q", idx, lang, report.Stdout, want)
		}
		return r
	})

	checkResults(t, results)
}
