package cpp

import (
	"fmt"
	"os"

	"github.com/jjudge/worker/internal/config"
	"github.com/jjudge/worker/internal/executor"
	"github.com/jjudge/worker/internal/handler"
	"github.com/jjudge/worker/internal/resource"
	"github.com/jjudge/worker/internal/result"
)

var (
	CPP_COMPILE_MEMORY_LIMIT int64 = 256 // MB
	CPP_COMPILE_CPU_LIMIT    int64 = 5   // seconds
	CPP_COMPILE_NPROC_LIMIT  int   = 100
)

type cppHandler struct{}

func NewCppHandler() handler.Handler {
	return cppHandler{}
}

func (h cppHandler) Handle(cfg *config.Config) result.Result {
	code, err := h.readCode(cfg.Dir)
	if err != nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("read code error: %w", err))
	}

	stdin, err := h.readStdin(cfg.Stdin)
	if err != nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("read stdin error: %w", err))
	}

	h.prepareEnv(code, cfg.UID, cfg.GID)

	if err := h.compile(
		resource.Limits{
			Memory:    CPP_COMPILE_MEMORY_LIMIT,
			Time:      CPP_COMPILE_CPU_LIMIT,
			Processes: CPP_COMPILE_NPROC_LIMIT,
		},
		cfg.UID,
		cfg.GID,
	); err != nil {
		return result.ResultWithError(result.VerdictCompileError, fmt.Errorf("compile error: %w", err))
	}

	if _, err := os.Stat("a"); err != nil {
		if os.IsNotExist(err) {
			return result.ResultWithError(result.VerdictCompileError, fmt.Errorf("compiled file not found: %w", err))
		}
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("stat error: %w", err))
	}

	res, err := h.execute(
		stdin,
		resource.Limits{
			Memory:    cfg.MemoryLimit,
			Time:      cfg.TimeLimit,
			Processes: cfg.MaxProcesses,
		},
		cfg.UID,
		cfg.GID,
	)
	if err != nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("execution error: %w", err))
	}

	return res
}

func (h cppHandler) readCode(dir string) ([]byte, error) {
	return os.ReadFile(dir)
}

func (h cppHandler) readStdin(dir string) ([]byte, error) {
	return os.ReadFile(dir)
}

func (h cppHandler) prepareEnv(c []byte, uid, gid uint32) error {
	if err := os.Mkdir("/tmp/t", 0777); err != nil {
		return err
	}
	if err := os.Chown("/tmp/t", int(uid), int(gid)); err != nil {
		return err
	}
	if err := os.Chdir("/tmp/t"); err != nil {
		return err
	}
	if err := createFile("a.cpp", c); err != nil {
		return err
	}
	return nil
}

func (h cppHandler) compile(rlimit resource.Limits, uid, gid uint32) error {
	e := executor.NewJobExecutor(
		[]string{"g++", "a.cpp", "-o", "a"},
		[]byte{},
		rlimit,
		uid,
		gid,
	)

	_, err := e.Run()
	if err != nil {
		return err
	}

	return nil
}

func (h cppHandler) execute(stdin []byte, rlimit resource.Limits, uid, gid uint32) (result.Result, error) {
	e := executor.NewJobExecutor(
		[]string{"./a"},
		stdin,
		rlimit,
		uid,
		gid,
	)

	return e.Run()
}

func createFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
