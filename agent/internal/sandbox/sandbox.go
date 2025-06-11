package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	SANDBOX_ROOT = "/var/lib/sandbox"
	CGROUP_PATH  = "/sys/fs/cgroup/jjudge.slice"
)

// Sandbox represents a sandbox environment for running isolated processes.
type Sandbox struct {
	id string

	spec SandboxSpec

	// BundlePath is the absolute path to the directory containing the OCI bundle.
	// It includes the root filesystem, configuration file, and other necessary files.
	// The bundle directory structure is expected to follow the OCI runtime specification.
	bundlePath string

	// Stdout is the absolute path to the file where standard output will be written.
	stdout string

	// Stderr is the absolute path to the file where standard error will be written.
	stderr string
}

func NewSandbox(id string, spec SandboxSpec, bundlePath string) (*Sandbox, error) {
	if spec.Rootfs == "" || spec.BoxDir == "" || spec.WorkingDir == "" || len(spec.Args) == 0 {
		return nil, fmt.Errorf("invalid sandbox spec: missing required fields")
	}

	return &Sandbox{
		id:         id,
		spec:       spec,
		bundlePath: bundlePath,
		stdout:     filepath.Join(bundlePath, "stdout.txt"),
		stderr:     filepath.Join(bundlePath, "stderr.txt"),
	}, nil
}

// SandboxSpec defines the properties of a sandbox environment.
// It includes the root filesystem, working directory, box directory, command-line arguments,
// and user IDs (UID and GID) for the sandbox.
// It is used to generate an OCI runtime specification for the sandbox.
type SandboxSpec struct {
	// Rootfs is the path to the container's root filesystem.
	Rootfs string

	// BoxDir is a host directory that is bind-mounted into the container.
	// We use this to place the user's code and other files that the container needs to access
	// to run the code.
	BoxDir string

	// WorkingDir is the working directory inside the container (relative to Rootfs).
	WorkingDir string

	// Args are the command and arguments to execute inside the container.
	Args []string

	// Env are the environment variables set inside the container
	// (e.g. ["PATH=/usr/bin:/bin"]).
	Env []string

	// MemoryLimitMB is the memory limit of the container in megabytes.
	MemoryLimitMB int64

	// CPUQuotaMillis is the CPU time quota in milliseconds.
	CPUQuotaMillis int64

	// Timeout is the wall-clock timeout of the container (host time).
	// The container is stopped after this duration.
	Timeout time.Duration

	// HostUID and HostGID is the host UID and host GID to be mapped to
	// the container's root (UID 0 and GID 0) respectively.
	HostUID uint32
	HostGID uint32

	// SeccompProfilePath is an optional path to a custom seccomp JSON.
	// If empty, we use a minimal built-in whitelist.
	SeccompProfilePath string
}

// Result defines the result of running a command in a container.
type Result struct {
	Stdout []byte
	Stderr []byte

	MemoryPeak int64
	CpuUser    int64
	CpuSystem  int64
}

func (s *Sandbox) GenerateOCISpec() (specs.Spec, error) {
	memBytes := s.spec.MemoryLimitMB * 1024 * 1024
	cpuQuotaMicros := s.spec.CPUQuotaMillis * 1000
	cpuPeriod := uint64(100000)

	overlayDir := filepath.Join(s.bundlePath, "overlay")

	ociSpec := specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			Terminal:        false,
			Cwd:             s.spec.WorkingDir,
			Args:            s.spec.Args,
			Env:             s.spec.Env,
			User:            specs.User{UID: 0, GID: 0},
			NoNewPrivileges: true,
			Capabilities: &specs.LinuxCapabilities{
				Bounding:    []string{},
				Permitted:   []string{},
				Inheritable: []string{},
				Effective:   []string{},
				Ambient:     []string{},
			},
		},
		Root: &specs.Root{
			Path:     s.spec.Rootfs,
			Readonly: true,
		},
		Mounts: []specs.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/dev",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			},
			{
				Destination: "/dev/null",
				Type:        "bind",
				Source:      "/dev/null",
				Options:     []string{"bind", "ro"},
			},
			{
				Destination: "/dev/zero",
				Type:        "bind",
				Source:      "/dev/zero",
				Options:     []string{"bind", "ro"},
			},
			{
				Destination: "/dev/random",
				Type:        "bind",
				Source:      "/dev/urandom",
				Options:     []string{"bind", "ro"},
			},
			{
				Destination: "/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=1777", "size=65536k"},
			},
			{
				Destination: "/box",
				Type:        "bind",
				Source:      overlayDir,
				Options:     []string{"rbind", "rw", "exec", "nosuid", "nodev"},
			},
		},
		Linux: &specs.Linux{
			CgroupsPath: fmt.Sprintf("/jjudge.slice/%s", s.id),
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{Limit: &memBytes, Swap: &memBytes},
				CPU: &specs.LinuxCPU{
					Quota:  &cpuQuotaMicros,
					Period: &cpuPeriod,
				},
			},
			UIDMappings: []specs.LinuxIDMapping{
				{
					HostID:      s.spec.HostUID,
					ContainerID: 0,
					Size:        1,
				},
			},
			GIDMappings: []specs.LinuxIDMapping{
				{
					HostID:      s.spec.HostGID,
					ContainerID: 0,
					Size:        1,
				},
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.UserNamespace},
				{Type: specs.PIDNamespace},
				{Type: specs.MountNamespace},
				{Type: specs.UTSNamespace},
				{Type: specs.IPCNamespace},
				{Type: specs.NetworkNamespace},
				{Type: specs.CgroupNamespace},
			},
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
	}

	return ociSpec, nil
}

func (s *Sandbox) Run() (*Result, error) {
	// Write the OCI spec to <BundlePath>/config.json
	spec, err := s.GenerateOCISpec()
	if err != nil {
		return nil, fmt.Errorf("error generating oci spec: %w", err)
	}

	if err := os.MkdirAll(s.bundlePath, 0755); err != nil {
		return nil, fmt.Errorf("error creating sandbox directory: %w", err)
	}

	specFile := filepath.Join(s.bundlePath, "config.json")
	if err := writeSpecToFile(&spec, specFile); err != nil {
		return nil, fmt.Errorf("error writing oci spec to file: %w", err)
	}

	if err := prepareRootfs(s.bundlePath); err != nil {
		return nil, fmt.Errorf("error preparing rootfs: %w", err)
	}

	cgroupPath := filepath.Join(CGROUP_PATH, s.id)
	if err := os.Mkdir(cgroupPath, 0644); err != nil {
		return nil, fmt.Errorf("error preparing cgroup: %w", err)
	}
	defer os.RemoveAll(cgroupPath)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("runc", "run", "--bundle", s.bundlePath, "--keep", s.id)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	defer func() {
		exec.Command("runc", "delete", s.id).Run()
	}()
	if err != nil {
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		return nil, fmt.Errorf("error runc run: %w", err)
	}

	res := &Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}

	return res, nil
}

func prepareRootfs(bundlePath string) error {
	dirs := []string{"upper", "work", "overlay"}
	for _, d := range dirs {
		path := filepath.Join(bundlePath, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	lowerImage := filepath.Join(SANDBOX_ROOT, "lower", "rootfs", "box")

	upperDir := filepath.Join(bundlePath, "upper")
	workDir := filepath.Join(bundlePath, "work")
	overlayDir := filepath.Join(bundlePath, "overlay")

	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerImage, upperDir, workDir)
	if err := syscall.Mount("overlay", overlayDir, "overlay", 0, options); err != nil {
		return fmt.Errorf("mount overlay failed: %w", err)
	}

	return nil
}

func writeSpecToFile(spec *specs.Spec, path string) error {
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0644)
}
