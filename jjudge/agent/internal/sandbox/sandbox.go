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

	"github.com/containerd/cgroups/v3/cgroup2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	sandboxRoot = "/var/lib/sandbox"
	cgroupSlice = "jjudge.slice"
)

// Sandbox represents a sandbox environment for running isolated processes.
type Sandbox struct {
	id string

	spec SandboxSpec

	// rootfs is the path to the root filesystem to use for the sandbox.
	// It is expected to be a valid OCI root filesystem that can be used to create the container.
	rootfs string

	// bundlePath is the absolute path to the directory containing the OCI bundle.
	// It includes the root filesystem, configuration file, and other necessary files.
	// The bundle directory structure is expected to follow the OCI runtime specification.
	bundlePath string
}

func NewSandbox(id string, spec SandboxSpec, rootfs string) (*Sandbox, error) {
	return &Sandbox{
		id:         id,
		spec:       spec,
		rootfs:     rootfs,
		bundlePath: filepath.Join(sandboxRoot, "jobs", id),
	}, nil
}

// SandboxSpec defines the properties of a sandbox environment.
// It includes the root filesystem, working directory, box directory, command-line arguments,
// and user IDs (UID and GID) for the sandbox.
// It is used to generate an OCI runtime specification for the sandbox.
type SandboxSpec struct {
	// WorkingDir is the working directory inside the container.
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
	Stdout string
	Stderr string

	MemoryPeakBytes uint64
	CpuUsageUsec    uint64
}

func (s *Sandbox) GetID() string {
	return s.id
}

func (s *Sandbox) Run() (*Result, error) {
	s.bundlePath = filepath.Join(sandboxRoot, "jobs", s.id)
	if err := os.MkdirAll(s.bundlePath, 0755); err != nil {
		return nil, fmt.Errorf("error creating sandbox directory: %w", err)
	}

	spec, err := s.generateOCISpec()
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

	if err := s.prepareRootfs(); err != nil {
		return nil, fmt.Errorf("error preparing rootfs: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("runc", "run", "--bundle", s.bundlePath, "--keep", s.id)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	defer func() {
		exec.Command("runc", "delete", s.id).Run()
	}()

	res := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		return res, fmt.Errorf("error runc run: %w", err)
	}

	m, err := cgroup2.LoadSystemd(cgroupSlice, fmt.Sprintf("%s.scope", s.id))
	if err != nil {
		return nil, fmt.Errorf("cannot load cgroup: %w", err)
	}

	metrics, err := m.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot get cgroup metrics: %w", err)
	}

	memPeak := metrics.GetMemory().GetMaxUsage()
	cpuUsage := metrics.GetCPU().GetUsageUsec()

	res.MemoryPeakBytes = memPeak
	res.CpuUsageUsec = cpuUsage

	return res, nil
}

func (s *Sandbox) generateOCISpec() (specs.Spec, error) {
	memBytes := s.spec.MemoryLimitMB * 1024 * 1024
	cpuQuotaMicros := s.spec.CPUQuotaMillis * 1000
	cpuPeriod := uint64(100000)

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
			Path:     filepath.Join(s.bundlePath, "rootfs"),
			Readonly: false,
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
		},
		Linux: &specs.Linux{
			CgroupsPath: fmt.Sprintf("/%s/%s.scope", cgroupSlice, s.id),
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{Limit: &memBytes, Swap: &memBytes},
				CPU: &specs.LinuxCPU{
					Quota:  &cpuQuotaMicros,
					Period: &cpuPeriod,
				},
				Network: nil,
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

func (s *Sandbox) prepareRootfs() error {
	dirs := []string{"upper", "work", "rootfs"}
	for _, d := range dirs {
		path := filepath.Join(s.bundlePath, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	lowerImage := s.rootfs

	upperDir := filepath.Join(s.bundlePath, "upper")
	workDir := filepath.Join(s.bundlePath, "work")
	overlayDir := filepath.Join(s.bundlePath, "rootfs")

	if err := os.Chown(upperDir, int(s.spec.HostUID), int(s.spec.HostGID)); err != nil {
		return fmt.Errorf("chown upperDir: %w", err)
	}
	if err := os.Chown(workDir, int(s.spec.HostUID), int(s.spec.HostGID)); err != nil {
		return fmt.Errorf("chown workDir: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(upperDir, s.spec.WorkingDir), 0755); err != nil {
		return fmt.Errorf("error creating upper directory: %w", err)
	}

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
