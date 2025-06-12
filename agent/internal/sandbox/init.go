package sandbox

import (
	"errors"

	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup2"
)

func Init() error {
	if !checkCgroupV2() {
		return errors.New("cgroup v2 not found")
	}

	resources := cgroup2.Resources{}

	_, err := cgroup2.NewSystemd("/", "jjudge.slice", -1, &resources)
	if err != nil {
		return err
	}

	return nil
}

func checkCgroupV2() bool {
	var cgroupV2 bool
	if cgroups.Mode() == cgroups.Unified {
		cgroupV2 = true
	}

	return cgroupV2
}
