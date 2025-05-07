package sandbox

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	CGROUP_PATH = "/sys/fs/cgroup/jjudge.slice"
)

type Sandbox struct {
	Id    string
	Spec  SandboxSpec
	Files map[string]string
}

type SandboxSpec struct {
	Rootfs     string
	WorkingDir string
	BoxDir     string
	Args       []string
	UID        uint32
	GID        uint32
}

func (s *Sandbox) GenerateOCISpec() specs.Spec {
	spec := specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			Terminal: false,
			Cwd:      s.Spec.WorkingDir,
			Args:     s.Spec.Args,
			User: specs.User{
				UID: s.Spec.UID,
				GID: s.Spec.GID,
			},
			Env: []string{"PATH=/usr/bin:/bin"},
		},
		Root: &specs.Root{
			Path:     s.Spec.Rootfs,
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
				Source:      s.Spec.BoxDir,
				Options:     []string{"rbind", "rw"},
			},
		},
		Linux: &specs.Linux{
			CgroupsPath: fmt.Sprintf("/jjudge.slice/%s", s.Id),
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{
					Limit: func(i int64) *int64 { return &i }(512 * 1024 * 1024),
				},
			},
			UIDMappings: []specs.LinuxIDMapping{
				{
					HostID:      s.Spec.UID,
					ContainerID: 0,
					Size:        1,
				},
			},
			GIDMappings: []specs.LinuxIDMapping{
				{
					HostID:      s.Spec.GID,
					ContainerID: 0,
					Size:        1,
				},
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.MountNamespace},
				{Type: specs.UTSNamespace},
				{Type: specs.IPCNamespace},
				{Type: specs.PIDNamespace},
				{Type: specs.UserNamespace},
				{Type: specs.NetworkNamespace},
			},
		},
	}

	return spec
}
