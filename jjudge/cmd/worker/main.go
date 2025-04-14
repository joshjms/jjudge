package main

import (
	"encoding/json"
	"log"

	"github.com/jjudge/worker/internal/config"
	"github.com/jjudge/worker/internal/handler"
	"github.com/jjudge/worker/internal/handler/cpp"
	"github.com/jjudge/worker/internal/result"
	"github.com/spf13/pflag"
)

func initFlags(c *config.Config) {
	pflag.StringVarP(&c.Dir, "dir", "d", "/data/c", "Directory of the source code")
	pflag.StringVarP(&c.Stdin, "stdin", "i", "/data/in", "Input file")
	pflag.StringVar(&c.Language, "lang", "cpp", "Language of the code")
	pflag.Int64Var(&c.MemoryLimit, "ml", 256, "Memory limit of the running code (in MB)")
	pflag.Int64Var(&c.TimeLimit, "tl", 1, "Time limit of the running code (in seconds)")
	pflag.IntVar(&c.MaxProcesses, "proc", 1, "Maximum number of processes")
	pflag.Uint32VarP(&c.UID, "uid", "u", 65534, "UID of the process to monitor")
	pflag.Uint32VarP(&c.GID, "gid", "g", 65534, "GID of the process to monitor")
}

func main() {
	cfg := config.Config{}
	initFlags(&cfg)

	pflag.Parse()

	var languageHandlers = map[string]func() handler.Handler{
		"cpp": cpp.NewCppHandler,
	}

	if handlerInitFunc, ok := languageHandlers[cfg.Language]; ok {
		h := handlerInitFunc()
		res := h.Handle(&cfg)

		printResult(res)
	} else {
		log.Fatal("unsupported language")
	}
}

func printResult(res result.Result) {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := log.Writer().Write(b); err != nil {
		log.Fatal(err)
	}
}
