package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func createOCIBundle(bundlePath string, boxPath string) {
	// Make directories
	must(os.MkdirAll(filepath.Join(bundlePath, "rootfs"), 0755))

	// Create config.json
	spec := specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			Terminal: false,
			Cwd:      "/",
			Args:     []string{"/box/main"},
			User: specs.User{
				UID: 65534,
				GID: 65534,
			},
			Env: []string{"PATH=/usr/bin:/bin"},
		},
		Root: &specs.Root{
			Path:     "rootfs",
			Readonly: true,
		},
		Mounts: []specs.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
			},
			{
				Destination: "/dev",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=755"},
			},
			{
				Destination: "/box",
				Type:        "bind",
				Source:      boxPath,
				Options:     []string{"rbind", "rw"},
			},
		},
		Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.MountNamespace},
				{Type: specs.UTSNamespace},
				{Type: specs.IPCNamespace},
				{Type: specs.PIDNamespace},
				{Type: specs.NetworkNamespace},
			},
		},
	}

	f, err := os.Create(filepath.Join(bundlePath, "config.json"))
	must(err)
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(spec))
}

func runWithRunsc(bundlePath string, id string) {
	cmd := exec.Command("runsc", "run", "--bundle", bundlePath, id)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	must(cmd.Run())
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

func main() {
	bundlePath := "/tmp/job-runsc"
	boxPath := "/tmp/job-runsc/box"

	must(os.MkdirAll(boxPath, 0755))
	copyMainBinary(boxPath)

	createOCIBundle(bundlePath, boxPath)
	runWithRunsc(bundlePath, "job1")
}
