package handler

import (
	"fmt"
	"os"
	"strings"

	"github.com/jjudge/worker/internal/config"
	"github.com/jjudge/worker/internal/executor"
	"github.com/jjudge/worker/pkg/resource"
	"github.com/jjudge/worker/pkg/result"
)

// The handler package is responsible for handling the execution of code submissions.

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Init() error {
	if err := os.Mkdir("/tmp/fs", 0755); err != nil {
		return err
	}
	if err := os.Chdir("/tmp/fs"); err != nil {
		return err
	}
	return nil
}

func (h *Handler) Handle(cfg *config.Config) map[string]result.Result {
	for _, file := range cfg.Files {
		if err := os.WriteFile(file.Name, []byte(file.Content), 0644); err != nil {
			return nil
		}
	}

	results := make(map[string]result.Result)

	for _, step := range cfg.Steps {
		res := h.handleStep(step)
		results[step.Name] = res
	}
	return results
}

func (h *Handler) handleStep(step config.Step) result.Result {
	if err := os.Chown("/tmp/fs", int(step.UID), int(step.GID)); err != nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("chown error: %w", err))
	}

	if err := os.Chdir("/tmp/fs"); err != nil {
		return result.ResultWithError(result.VerdictInternalError, fmt.Errorf("chdir error: %w", err))
	}

	e := executor.NewJobExecutor(
		strings.Fields(step.Cmd),
		[]byte(step.Stdin),
		resource.Limits{
			Memory:    step.MemoryLimit,
			Time:      step.TimeLimit,
			Processes: step.MaxProcesses,
		},
		step.UID,
		step.GID,
	)

	res, err := e.Run()
	if err != nil {
		return result.ResultWithError(result.VerdictInternalError, err)
	}

	return res
}
