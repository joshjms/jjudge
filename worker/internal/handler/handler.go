package handler

import (
	"github.com/jjudge/worker/internal/config"
	"github.com/jjudge/worker/pkg/result"
)

// The handler package is responsible for handling the execution of code submissions.
// It defines the handler logic for each programming language as well as manages appropriate verdicts.

type Handler interface {
	Handle(*config.Config) result.Result
}
