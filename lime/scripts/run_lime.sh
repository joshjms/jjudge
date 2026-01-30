#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
	echo "usage: $0 <source-file> [compile_id] [exec_id]" >&2
	exit 2
fi

src_file="$1"
compile_id="${2:-compile}"
exec_id="${3:-exec}"

rm -rf /tmp/lime_test_compile_input /tmp/lime_test_exec_input

mkdir /tmp/lime_test_compile_input
mkdir /tmp/lime_test_exec_input

cp "tests/test_files/${src_file}" /tmp/lime_test_compile_input/

export LIME_CGROUP_ROOT="${LIME_CGROUP_ROOT:-/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/lime.slice}"

LIME_BIN="./build/lime"
LIME_ID="${compile_id}"
LIME_ARGS_JSON="[\"/usr/local/bin/g++\", \"-o\", \"/tmp/output/a.out\", \"/tmp/input/${src_file}\"]"
LIME_ENVP_JSON="[\"PATH=/usr/bin:/usr/local/bin:/bin\"]"
LIME_CPU_TIME_LIMIT_US="1000000"
LIME_WALL_TIME_LIMIT_US="2000000"
LIME_MEMORY_LIMIT_BYTES="268435456"
LIME_MAX_PROCESSES="10"
LIME_OUTPUT_LIMIT_BYTES="8388608"
LIME_MAX_OPEN_FILES="16"
LIME_STACK_LIMIT_BYTES="8388608"
LIME_USE_CPUS="4"
LIME_USE_MEMS="0"
LIME_STDIN=""
LIME_ROOTFS_PATH="/var/castletown/images/gcc-15-bookworm"
LIME_BIND_MOUNTS_JSON="[\"/tmp/lime_test_compile_input:/tmp/input:ro\", \"/tmp/lime_test_exec_input:/tmp/output\"]"
LIME_USE_OVERLAYFS="true"

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
	"use_cpus": "${LIME_USE_CPUS}",
	"use_mems": "${LIME_USE_MEMS}",
	"stdin": "${LIME_STDIN}",
	"rootfs_path": "${LIME_ROOTFS_PATH}",
	"bind_mounts": ${LIME_BIND_MOUNTS_JSON},
	"use_overlayfs": ${LIME_USE_OVERLAYFS}
}
JSON

LIME_ID="${exec_id}"
LIME_ARGS_JSON="[\"/tmp/input/a.out\"]"
LIME_BIND_MOUNTS_JSON="[\"/tmp/lime_test_exec_input:/tmp/input:ro\"]"
LIME_MAX_PROCESSES="1"

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
	"use_cpus": "${LIME_USE_CPUS}",
	"use_mems": "${LIME_USE_MEMS}",
	"stdin": "${LIME_STDIN}",
	"rootfs_path": "${LIME_ROOTFS_PATH}",
	"bind_mounts": ${LIME_BIND_MOUNTS_JSON},
	"use_overlayfs": ${LIME_USE_OVERLAYFS}
}
JSON
