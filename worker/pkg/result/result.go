package result

import "github.com/jjudge/worker/internal/resource"

type Result struct {
	Verdict  string
	Stdout   string
	Stderr   string
	Signal   string
	ExitCode int
	Usage    resource.Usage
}

func ResultWithError(v string, err error) Result {
	return Result{
		Verdict: v,
		Stdout:  "",
		Stderr:  err.Error(),
		Signal:  "",
	}
}
