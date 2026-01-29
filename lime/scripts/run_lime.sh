#!/usr/bin/env bash
set -euo pipefail

export LIME_CGROUP_ROOT="${LIME_CGROUP_ROOT:-/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/lime.slice}"

: "${LIME_BIN:=./build/lime}"
: "${LIME_ID:=test}"
: "${LIME_ARGS_JSON:=[\"/bin/echo\", \"Hello, from Lime!\"]}"
: "${LIME_ENVP_JSON:=[\"PATH=/usr/bin:/usr/local/bin:/bin\"]}"
: "${LIME_CPU_TIME_LIMIT_US:=1000000}"
: "${LIME_WALL_TIME_LIMIT_US:=2000000}"
: "${LIME_MEMORY_LIMIT_BYTES:=268435456}"
: "${LIME_MAX_PROCESSES:=1}"
: "${LIME_OUTPUT_LIMIT_BYTES:=8388608}"
: "${LIME_MAX_OPEN_FILES:=16}"
: "${LIME_STACK_LIMIT_BYTES:=8388608}"
: "${LIME_STDIN:=}"
: "${LIME_ROOTFS_PATH:=/var/castletown/images/gcc-15-bookworm}"
: "${LIME_BIND_MOUNTS_JSON:=[]}"
: "${LIME_USE_OVERLAYFS:=true}"

cat <<JSON | "${LIME_BIN}" run
{
	"id": "${LIME_ID}",
	"args": ${LIME_ARGS_JSON},
	"envp": ${LIME_ENVP_JSON},
	"cpu_time_limit_us": ${LIME_CPU_TIME_LIMIT_US},
	"wall_time_limit_us": ${LIME_WALL_TIME_LIMIT_US},
	"memory_limit_bytes": ${LIME_MEMORY_LIMIT_BYTES},
	"max_processes": ${LIME_MAX_PROCESSES},
	"output_limit_bytes": ${LIME_OUTPUT_LIMIT_BYTES},
	"max_open_files": ${LIME_MAX_OPEN_FILES},
	"stack_limit_bytes": ${LIME_STACK_LIMIT_BYTES},
	"stdin": "${LIME_STDIN}",
	"rootfs_path": "${LIME_ROOTFS_PATH}",
	"bind_mounts": ${LIME_BIND_MOUNTS_JSON},
	"use_overlayfs": ${LIME_USE_OVERLAYFS}
}
JSON
