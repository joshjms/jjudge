# Getting Started

## Enable cgroups for unprivileged user
Lime uses cgroup v2 to enforce limits. You need a delegated cgroup subtree that your
user can write to. If you already have one, skip this and set `LIME_CGROUP_ROOT`.

Example (systemd user slice; path varies by distro/user id):

```bash
export LIME_CGROUP_ROOT=/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/lime.slice
mkdir -p "$LIME_CGROUP_ROOT"
```

Don't forget to enable controllers for the subtree:

```bash
echo "+cpu +cpuset +memory +pids +io" > /sys/fs/cgroup/cgroup.subtree_control
```

## Build

```bash
make
```

## Requirements

- Linux with cgroup v2 enabled
- `newuidmap` and `newgidmap` available in `PATH` (typically from `uidmap` package)
- A rootfs directory (e.g., a minimal Debian/Ubuntu rootfs)

## Run a job

You can run Lime by piping a JSON config to `lime run`:

```bash
cat <<'JSON' | ./build/lime run
{
  "id": "example",
  "args": ["/bin/echo", "hello from lime"],
  "envp": ["PATH=/usr/bin:/usr/local/bin:/bin"],
  "cpu_time_limit_us": 1000000,
  "wall_time_limit_us": 2000000,
  "memory_limit_bytes": 268435456,
  "max_processes": 1,
  "output_limit_bytes": 8388608,
  "max_open_files": 16,
  "stack_limit_bytes": 8388608,
  "stdin": "",
  "rootfs_path": "/path/to/rootfs",
  "bind_mounts": [],
  "use_overlayfs": true
}
JSON
```

For a ready-to-run example, see `scripts/run_lime.sh`.

## Notes

- `rootfs_path` must point to a directory.
- `bind_mounts` entries are strings of the form `src:dst[:ro|rw]`.
- If you use `stdin`, the child reads exactly that content, then EOF.
