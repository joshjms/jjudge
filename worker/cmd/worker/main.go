package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jjudge/worker/internal/config"
	"github.com/jjudge/worker/internal/handler"
	"github.com/spf13/pflag"
)

func initConfig(c *config.Config) error {
	var dir string

	pflag.StringVarP(&dir, "dir", "d", "/data/config.yaml", "Directory of the configuration file")
	pflag.Parse()

	if dir == "" {
		return fmt.Errorf("config file directory is empty")
	}

	if err := c.Load(dir); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	return nil
}

func main() {
	cfg := config.Config{}
	if err := initConfig(&cfg); err != nil {
		os.Exit(1)
	}

	h := handler.NewHandler()
	if err := h.Init(); err != nil {
		os.Exit(1)
	}
	results := h.Handle(&cfg)
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		os.Exit(1)
	}
	if _, err := log.Writer().Write(b); err != nil {
		os.Exit(1)
	}
}
