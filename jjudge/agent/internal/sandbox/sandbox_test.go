package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxWorkdir(t *testing.T) {
	spec := SandboxSpec{
		Rootfs:             "/var/lib/sandbox/images/rootfs",
		BoxDir:             "./box",
		WorkingDir:         "/box",
		Args:               []string{"pwd"},
		Env:                []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		HostUID:            65536,
		HostGID:            65536,
		MemoryLimitMB:      512,
		CPUQuotaMillis:     1000,
		Timeout:            30,
		SeccompProfilePath: "",
	}

	sandbox, err := NewSandbox("test-sandbox", spec, "/var/lib/sandbox/jobs/test-sandbox")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if sandbox.id != "test-sandbox" {
		t.Fatalf("Expected sandbox ID 'test-sandbox', got '%s'", sandbox.id)
	}

	res, err := sandbox.Run()
	if err != nil {
		t.Fatalf("Failed to run sandbox: %v", err)
	}

	stdoutString := string(res.Stdout)
	if strings.TrimSpace(stdoutString) != spec.WorkingDir {
		t.Fatalf("Expected wd: %s, got: %s", spec.WorkingDir, stdoutString)
	}
}

func TestSandboxCreateFile(t *testing.T) {
	spec := SandboxSpec{
		Rootfs:             "/var/lib/sandbox/images/rootfs",
		BoxDir:             "./box",
		WorkingDir:         "/box",
		Args:               []string{"sh", "-c", "echo 'i love her' > hello.txt"},
		Env:                []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		HostUID:            65536,
		HostGID:            65536,
		MemoryLimitMB:      512,
		CPUQuotaMillis:     1000,
		Timeout:            30,
		SeccompProfilePath: "",
	}

	sandbox, err := NewSandbox("test-sandbox", spec, "/var/lib/sandbox/jobs/test-sandbox")
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	_, err = sandbox.Run()
	if err != nil {
		t.Fatalf("Failed to run sandbox: %v", err)
	}

	bundlePath := "/var/lib/sandbox/jobs/test-sandbox/"
	b, err := os.ReadFile(filepath.Join(bundlePath, "upper", "hello.txt"))
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}

	if strings.TrimSpace(string(b)) != "i love her" {
		t.Fatalf("Expected text: %v, got: %v", "i love her", string(b))
	}
}

func TestSandboxMultipleSandboxes(t *testing.T) {
	containerIds := []string{"sandbox-01", "sandbox-02", "sandbox-03"}

	spec := SandboxSpec{
		Rootfs:             "/var/lib/sandbox/images/rootfs",
		BoxDir:             "./box",
		WorkingDir:         "/box",
		Args:               []string{"sleep", "10"},
		Env:                []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		HostUID:            65536,
		HostGID:            65536,
		MemoryLimitMB:      512,
		CPUQuotaMillis:     1000,
		Timeout:            30,
		SeccompProfilePath: "",
	}

	ch := make(chan error)

	for _, id := range containerIds {
		go func() {
			sandbox, err := NewSandbox(id, spec, fmt.Sprintf("/var/lib/sandbox/jobs/%s", id))
			if err != nil {
				ch <- err
			}

			_, err = sandbox.Run()
			if err != nil {
				ch <- err
			}

			ch <- nil
		}()
	}

	for i := 0; i < len(containerIds); i++ {
		err := <-ch
		if err != nil {
			t.Fatalf("Failed to run sandbox: %v", err)
		}
	}
}
