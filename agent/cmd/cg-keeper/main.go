package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/daemon"
)

const cgRoot = "/sys/fs/cgroup"

func die(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}

func checkCgroupFS() {
	if _, err := os.Stat(cgRoot); err != nil {
		die("cannot find cgroup fs", err)
	}
	if _, err := os.Stat(filepath.Join(cgRoot, "unified")); err == nil {
		die("hybrid v1+v2 mode not supported", nil)
	}
	if _, err := os.Stat(filepath.Join(cgRoot, "cgroup.subtree_control")); err != nil {
		die("cgroup v2 not found", err)
	}
}

func getMyCgroup() string {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		die("open /proc/self/cgroup", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "0::") {
			rel := strings.TrimPrefix(line, "0::")
			return filepath.Join(cgRoot, rel)
		}
	}
	die("could not find own cgroup entry", nil)
	return ""
}

func writeAttr(path, val string) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		die("open "+path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(val); err != nil {
		die("write "+path, err)
	}
}

func main() {
	checkCgroupFS()

	base := getMyCgroup()
	if _, err := os.Stat(base); err != nil {
		die("base cgroup does not exist", err)
	}

	sub := filepath.Join(base, "daemon")
	if err := os.Mkdir(sub, 0777); err != nil && !os.IsExist(err) {
		die("mkdir "+sub, err)
	}

	writeAttr(filepath.Join(sub, "cgroup.procs"), fmt.Sprintf("%d\n", os.Getpid()))

	writeAttr(filepath.Join(base, "cgroup.subtree_control"), "+cpuset +memory\n")

	sent, err := daemon.SdNotify(false, "READY=1")
	if err != nil {
		die("notify systemd", err)
	}
	if !sent {
		fmt.Fprintln(os.Stderr, "warning: notify was not sent (not running under systemd?)")
	}

	// Sleep
	select {}
}
