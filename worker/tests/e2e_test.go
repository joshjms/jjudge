// Package tests contains end-to-end tests for the worker's per-slot uid
// isolation. They verify that:
//   - each slot in the pool is assigned a unique, non-zero uid
//   - lime is exec'd as the slot uid (confirmed via /proc/self/uid_map)
//   - the work directory is chowned to the slot uid during execution and
//     restored to root (uid 0) after lime.Run returns
//   - a process running as one slot uid cannot access another slot's work dir
//   - concurrent executions on different slots use distinct outer uids
//   - the full C++ compile-and-run pipeline works correctly under uid isolation
//
// Requirements:
//
//	LIME_INTEGRATION=1   enable the tests
//	LIME_CGROUP_ROOT     path to a delegated cgroup subtree
//	LIME_ROOTFS          path to the rootfs (default /rootfs)
//	lime binary in PATH
//	run as root (uid 0) — required for chown and CAP_SETUID
package tests_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/jjudge-oj/worker/config"
	"github.com/jjudge-oj/worker/internal/lime"
)

// ── test bootstrap ────────────────────────────────────────────────────────────

var testRootfs string

func TestMain(m *testing.M) {
	testRootfs = os.Getenv("LIME_ROOTFS")
	if testRootfs == "" {
		testRootfs = "/rootfs"
	}
	os.Exit(m.Run())
}

// ── helpers ───────────────────────────────────────────────────────────────────

func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("LIME_INTEGRATION") == "" {
		t.Skip("set LIME_INTEGRATION=1 to run integration tests")
	}
	if os.Getenv("LIME_CGROUP_ROOT") == "" {
		t.Skip("LIME_CGROUP_ROOT not set")
	}
	if _, err := exec.LookPath("lime"); err != nil {
		t.Skip("lime binary not found in PATH")
	}
	if os.Getuid() != 0 {
		t.Skip("must run as root (uid 0)")
	}
	if _, err := os.Stat(testRootfs); err != nil {
		t.Skipf("rootfs not found at %s", testRootfs)
	}
}

// testConfig returns a minimal *config.Config sufficient for lime.Run.
func testConfig() *config.Config {
	return &config.Config{
		Judge: &config.JudgeConfig{
			RootfsDir:    testRootfs,
			OverlayFSDir: os.TempDir(),
		},
	}
}

// newPool creates a SlotPool pinned to CPU 0 with uid isolation starting at base.
func newPool(uidBase int) *lime.SlotPool {
	return lime.NewSlotPool(
		lime.WithSlotUIDs(uidBase),
		lime.WithCPUs("0"),
	)
}

// compileBinary compiles src (.cpp file) inside a lime sandbox using pool and
// returns the work directory containing the "solution" binary.
func compileBinary(t *testing.T, pool *lime.SlotPool, src string) string {
	t.Helper()

	workDir := t.TempDir()
	srcName := filepath.Base(src)
	copyFile(t, src, filepath.Join(workDir, srcName))

	report, err := lime.Run(
		context.Background(), testConfig(), pool, workDir, testRootfs,
		[]string{"/usr/bin/g++", "-std=c++20", "-O2", "-o", "/work/solution", "/work/" + srcName},
		"", 30_000_000, 512*1024*1024, 32, false,
	)
	if err != nil {
		t.Fatalf("compile %s: %v", src, err)
	}
	if report.ExitCode != 0 {
		t.Fatalf("compile %s failed (exit %d):\n%s", src, report.ExitCode, report.Stderr)
	}
	return workDir
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("copy: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", dst, err)
	}
}

// parseOuterUID parses the outer (host) uid from /proc/self/uid_map output.
// The format is "inner  outer  count"; we want the outer field.
func parseOuterUID(t *testing.T, uidMapOutput string) int {
	t.Helper()
	fields := strings.Fields(strings.TrimSpace(uidMapOutput))
	if len(fields) < 3 {
		t.Fatalf("unexpected uid_map output: %q", uidMapOutput)
	}
	outer, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("parse outer uid from %q: %v", uidMapOutput, err)
	}
	return outer
}

// ── unit-level tests (no lime required) ──────────────────────────────────────

// TestSlotPoolUIDAssignment verifies that every slot is assigned a unique,
// consecutive uid starting at uidBase, with no slot having uid == 0.
func TestSlotPoolUIDAssignment(t *testing.T) {
	const base = 60000
	const n = 4

	pool := lime.NewSlotPool(
		lime.WithSlotUIDs(base),
		lime.WithCPUs(fmt.Sprintf("0-%d", n-1)),
	)

	ctx := context.Background()
	allocs := make([]*lime.Allocation, n)
	seen := make(map[int]bool)

	for i := range allocs {
		a, err := pool.Allocate(ctx)
		if err != nil {
			t.Fatalf("allocate slot %d: %v", i, err)
		}
		allocs[i] = a

		uid := a.Slot().UID
		if uid == 0 {
			t.Errorf("slot %d has uid 0 — isolation is disabled", i)
		}
		if seen[uid] {
			t.Errorf("duplicate uid %d at slot %d", uid, i)
		}
		seen[uid] = true
	}

	for _, a := range allocs {
		a.Release()
	}

	for i := 0; i < n; i++ {
		if !seen[base+i] {
			t.Errorf("expected uid %d not assigned to any slot", base+i)
		}
	}
}

// TestSlotPoolZeroBaseDisablesIsolation verifies that uidBase == 0 leaves all
// slot UIDs as 0 (the sentinel meaning "no isolation").
func TestSlotPoolZeroBaseDisablesIsolation(t *testing.T) {
	pool := lime.NewSlotPool(
		lime.WithSlotUIDs(0),
		lime.WithCPUs("0,1"),
	)

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		a, err := pool.Allocate(ctx)
		if err != nil {
			t.Fatalf("allocate: %v", err)
		}
		if a.Slot().UID != 0 {
			t.Errorf("slot %d: expected uid 0 when base is 0, got %d", i, a.Slot().UID)
		}
		a.Release()
	}
}

// TestWorkDirIsolationPermissions verifies that chown+chmod 0700 prevents a
// process running as one slot uid from accessing another slot's work directory.
// This models the access a sandbox escapee would have on the host.
func TestWorkDirIsolationPermissions(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root to chown directories")
	}

	const uid0, uid1 = 60000, 60001

	// t.TempDir() creates dirs under a root-owned 0700 parent, which slot uids
	// cannot traverse. Use a world-executable parent instead.
	parent := filepath.Join(os.TempDir(), fmt.Sprintf("jjudge-isolation-test-%d", os.Getpid()))
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(parent) })

	dir0 := filepath.Join(parent, "slot0")
	dir1 := filepath.Join(parent, "slot1")
	for _, d := range []string{dir0, dir1} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	setupSlotDir := func(dir string, uid int) {
		if err := os.Chown(dir, uid, uid); err != nil {
			t.Fatalf("chown %s → %d: %v", dir, uid, err)
		}
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatalf("chmod %s: %v", dir, err)
		}
		secret := filepath.Join(dir, "secret.txt")
		if err := os.WriteFile(secret, []byte("secret"), 0600); err != nil {
			t.Fatalf("write secret in %s: %v", dir, err)
		}
		if err := os.Chown(secret, uid, uid); err != nil {
			t.Fatalf("chown secret: %v", err)
		}
	}

	setupSlotDir(dir0, uid0)
	setupSlotDir(dir1, uid1)

	tryAccess := func(asUID uint32, targetDir string) error {
		cmd := exec.Command("/bin/ls", targetDir)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: asUID, Gid: asUID},
		}
		_, err := cmd.CombinedOutput()
		return err
	}

	if err := tryAccess(uid0, dir1); err == nil {
		t.Errorf("uid %d should NOT be able to list uid %d's work dir", uid0, uid1)
	}
	if err := tryAccess(uid1, dir0); err == nil {
		t.Errorf("uid %d should NOT be able to list uid %d's work dir", uid1, uid0)
	}

	// Sanity check: each uid can access its own dir.
	if err := tryAccess(uid0, dir0); err != nil {
		t.Errorf("uid %d should be able to access its own dir: %v", uid0, err)
	}
	if err := tryAccess(uid1, dir1); err != nil {
		t.Errorf("uid %d should be able to access its own dir: %v", uid1, err)
	}
}

// ── integration tests (require lime + rootfs + root) ─────────────────────────

// TestLimeRunsAsSlotUID reads /proc/self/uid_map from inside the sandbox.
// The entry "0  <outer>  1" tells us the host uid lime was exec'd as; we
// verify it equals the slot's assigned uid.
func TestLimeRunsAsSlotUID(t *testing.T) {
	skipUnlessIntegration(t)

	const uidBase = 60000
	pool := newPool(uidBase)

	workDir := compileBinary(t, pool, "testdata/uid_map.cpp")

	report, err := lime.Run(
		context.Background(), testConfig(), pool, workDir, testRootfs,
		[]string{"/work/solution"}, "", 5_000_000, 256*1024*1024, 1, false,
	)
	if err != nil {
		t.Fatalf("run uid_map: %v", err)
	}
	if report.ExitCode != 0 {
		t.Fatalf("uid_map exited %d:\n%s", report.ExitCode, report.Stderr)
	}

	outerUID := parseOuterUID(t, report.Stdout)
	if outerUID != uidBase {
		t.Errorf("outer uid = %d, want %d (slot 0 with base %d)", outerUID, uidBase, uidBase)
	}
}

// TestWorkDirRestoredAfterRun verifies that lime.Run's deferred reset returns
// the work directory's owner to root (uid 0) after execution, allowing the
// worker to remove it.
func TestWorkDirRestoredAfterRun(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newPool(60000)
	workDir := t.TempDir()

	_, err := lime.Run(
		context.Background(), testConfig(), pool, workDir, testRootfs,
		[]string{"/usr/bin/python3", "-c", "print('ok')"},
		"", 5_000_000, 256*1024*1024, 1, false,
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	info, err := os.Stat(workDir)
	if err != nil {
		t.Fatalf("stat work dir: %v", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("cannot cast to syscall.Stat_t")
	}
	if stat.Uid != 0 {
		t.Errorf("work dir uid after run = %d, want 0 — deferred chown reset failed", stat.Uid)
	}
}

// TestConcurrentSlotsHaveDistinctUIDs runs two uid_map binaries simultaneously
// on a two-slot pool and asserts the reported outer uids are different.
func TestConcurrentSlotsHaveDistinctUIDs(t *testing.T) {
	skipUnlessIntegration(t)

	const uidBase = 60000
	pool := lime.NewSlotPool(
		lime.WithSlotUIDs(uidBase),
		lime.WithCPUs("0,1"),
	)

	// Compile once; copy the binary into a second work dir for the second slot.
	workDir0 := compileBinary(t, pool, "testdata/uid_map.cpp")
	workDir1 := t.TempDir()
	copyFile(t, filepath.Join(workDir0, "solution"), filepath.Join(workDir1, "solution"))

	type result struct {
		uid int
		err error
	}
	ch := make(chan result, 2)

	runSlot := func(workDir string) {
		r, err := lime.Run(
			context.Background(), testConfig(), pool, workDir, testRootfs,
			[]string{"/work/solution"}, "", 5_000_000, 256*1024*1024, 1, false,
		)
		if err != nil {
			ch <- result{err: err}
			return
		}
		ch <- result{uid: parseOuterUID(t, r.Stdout)}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); runSlot(workDir0) }()
	go func() { defer wg.Done(); runSlot(workDir1) }()
	wg.Wait()
	close(ch)

	var uids []int
	for res := range ch {
		if res.err != nil {
			t.Fatalf("concurrent run: %v", res.err)
		}
		uids = append(uids, res.uid)
	}

	if len(uids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(uids))
	}
	if uids[0] == uids[1] {
		t.Errorf("both concurrent executions ran as uid %d — slots are not isolated", uids[0])
	}
}

// TestEndToEndCppWithUIDIsolation compiles hello.cpp and runs it through the
// full lime pipeline with uid isolation enabled, checking stdout is correct.
func TestEndToEndCppWithUIDIsolation(t *testing.T) {
	skipUnlessIntegration(t)

	pool := newPool(60000)
	workDir := compileBinary(t, pool, "testdata/hello.cpp")

	report, err := lime.Run(
		context.Background(), testConfig(), pool, workDir, testRootfs,
		[]string{"/work/solution"}, "World\n", 5_000_000, 256*1024*1024, 1, true,
	)
	if err != nil {
		t.Fatalf("run hello: %v", err)
	}
	if report.ExitCode != 0 {
		t.Fatalf("hello exited %d:\n%s", report.ExitCode, report.Stderr)
	}

	const want = "Hello, World!\n"
	if report.Stdout != want {
		t.Errorf("stdout = %q, want %q", report.Stdout, want)
	}
}
