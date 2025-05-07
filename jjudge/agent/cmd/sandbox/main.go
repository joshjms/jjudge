package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jjudge/agent/internal/sandbox"
)

// usage: sandbox [flags]
func main() {
	initCgroup()

	os.Mkdir("pids", 0755)

	sandboxDir := "/tmp/jjudge"
	jobsDir := filepath.Join(sandboxDir, "jobs")
	jobID := "job11"
	jobDir := filepath.Join(jobsDir, jobID)
	boxDir := filepath.Join(jobDir, "box")

	configPath := filepath.Join(jobDir, "config.json")

	must(os.MkdirAll(boxDir, 0755))
	copyMainBinary(boxDir)

	sandbox := &sandbox.Sandbox{
		Id: jobID,
		Spec: sandbox.SandboxSpec{
			Rootfs:     "/home/joshjms/jjudge/agent/internal/rootfs",
			WorkingDir: "/box",
			BoxDir:     boxDir,
			Args:       []string{"sh", "-c", "set -x; ./main"},
			UID:        65534,
			GID:        65534,
		},
	}

	spec := sandbox.GenerateOCISpec()

	f, err := os.Create(configPath)
	must(err)
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(spec))

	log.Printf("configPath: %s", configPath)

	runGVisorSandbox(jobDir, jobID)
}

func runGVisorSandbox(bundlePath, containerID string) {
	cmd := exec.Command("runsc", "run", "--bundle", bundlePath, "--pid-file", "pid.txt", containerID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func copyMainBinary(destDir string) {
	srcFile := "./main"
	dstFile := filepath.Join(destDir, "main")

	src, err := os.Open(srcFile)
	must(err)
	defer src.Close()

	dst, err := os.Create(dstFile)
	must(err)
	defer dst.Close()

	_, err = io.Copy(dst, src)
	must(err)
	must(dst.Chmod(0755))
}

func initCgroup() {
	os.MkdirAll("/sys/fs/cgroup/jjudge", 0755)

	err := os.WriteFile("/sys/fs/cgroup/jjudge/cgroup.subtree_control", []byte("+memory +cpu +io\n"), 0644)
	if err != nil {
		log.Fatalf("cannot enable controllers: %v\n", err)
	}

	os.Mkdir("/sys/fs/cgroup/jjudge/daemon", 0755)
}
