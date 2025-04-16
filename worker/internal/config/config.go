package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Files []File `yaml:"files"`
	Steps []Step `yaml:"steps"`
}

type File struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`	
}

type Step struct {
	Name  string `yaml:"name"`
	Cmd   string `yaml:"cmd"`
	Stdin string `yaml:"stdin"`

	MemoryLimit  int64 `yaml:"memory_limit"`
	TimeLimit    int64 `yaml:"time_limit"`
	MaxProcesses int   `yaml:"max_processes"`

	UID uint32 `yaml:"uid"`
	GID uint32 `yaml:"gid"`
}

func (c *Config) Load(configFileDir string) error {
	b, err := os.ReadFile(configFileDir)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	if len(c.Files) == 0 {
		return fmt.Errorf("no files found in config file")
	}
	if len(c.Steps) == 0 {
		return fmt.Errorf("no steps found in config file")
	}

	return nil
}
