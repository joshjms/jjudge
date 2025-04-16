package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/jjudge/worker/pkg/resource"
	"github.com/jjudge/worker/pkg/result"
)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB

	Seconds = 1
)

// Executor is an interface that defines the Run method. Responsible for running the code.
type Executor interface {
	Run() (result.Result, error)
}

// jobExecutor is a struct that implements the Executor interface
type jobExecutor struct {
	Cmd   []string
	Stdin []byte

	RLimit resource.Limits

	// some configs from the master
	Uid uint32
	Gid uint32
}

func NewJobExecutor(cmd []string, stdin []byte, rlimit resource.Limits, uid, gid uint32) Executor {
	return &jobExecutor{
		Cmd:    cmd,
		Stdin:  stdin,
		RLimit: rlimit,
		Uid:    uid,
		Gid:    gid,
	}
}

func (e *jobExecutor) Run() (result.Result, error) {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	// Set the resource limits for the command
	// There are some hard limits here, will change them to be configurable later
	args := []string{
		fmt.Sprintf("--nproc=%d", e.RLimit.Processes),
		fmt.Sprintf("--as=%d:%d", e.RLimit.Memory*MB, 2*GB),
		fmt.Sprintf("--cpu=%d:%d", e.RLimit.Time, 20*Seconds),
		fmt.Sprintf("--fsize=%d", 128*MB),
		fmt.Sprintf("--nofile=%d", 1024),
		"--",
	}

	args = append(args, e.Cmd...)

	cmd := exec.Command(
		"prlimit",
		args...,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: e.Uid,
			Gid: e.Gid,
		},
		Setpgid: true,
	}

	cmd.Stdin = bytes.NewBuffer(e.Stdin)
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer

	if err := cmd.Start(); err != nil {
		return result.Result{
			Verdict: result.VerdictRuntimeError,
			Stderr:  stderrBuffer.String(),
		}, nil
	}

	cmd.Wait()

	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return result.Result{
			Verdict: result.VerdictInternalError,
		}, fmt.Errorf("failed to get exit status")
	}

	verdict, signal := classifyExit(status.ExitStatus(), status.Signal())
	if verdict != result.VerdictOk {
		return result.Result{
			Verdict: verdict,
			Signal:  signal,
			Stderr:  stderrBuffer.String(),
		}, nil
	}

	rusage := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if rusage == nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("failed to get resource usage")), fmt.Errorf("failed to get resource usage")
	}

	return result.Result{
		Verdict: result.VerdictOk,
		Stdout:  stdoutBuffer.String(),
		Usage: resource.Usage{
			Memory:        int64(rusage.Maxrss) * KB,
			UserCpuTime:   float64(rusage.Utime.Sec) + float64(rusage.Utime.Usec)/1e6,
			SystemCpuTime: float64(rusage.Stime.Sec) + float64(rusage.Stime.Usec)/1e6,
		},
	}, nil
}

// classifyExit classifies the exit code and signal into a verdict and a signal string
func classifyExit(exitCode int, signal syscall.Signal) (string, string) {
	if exitCode == 0 {
		return result.VerdictOk, ""
	}

	switch signal {
	case syscall.SIGXCPU:
		return result.VerdictTimeLimitExceeded, "SIGXCPU"
	case syscall.SIGXFSZ:
		return result.VerdictOutputLimitExceeded, "SIGXFSZ"
	case syscall.SIGKILL:
		return result.VerdictMemoryLimitExceeded, "SIGKILL"
	case syscall.SIGSEGV:
		return result.VerdictRuntimeError, "SIGSEGV"
	default:
		return result.VerdictRuntimeError, signal.String()
	}
}
