#!/usr/bin/env bash
# test.sh — run worker tests.
#
# Unit tests run always. Integration tests (require lime + rootfs + root)
# are enabled by passing --integration.
#
# Usage:
#   ./test.sh                  # unit tests only
#   ./test.sh --integration    # unit + integration tests

set -euo pipefail

INTEGRATION=0
GOTEST_ARGS=()
for arg in "$@"; do
    case "$arg" in
        --integration) INTEGRATION=1 ;;
        *) GOTEST_ARGS+=("$arg") ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [ "$INTEGRATION" -eq 1 ]; then
    if [ "$(id -u)" -ne 0 ]; then
        echo "error: --integration requires root (uid 0)" >&2
        exit 1
    fi
    if [ -z "${LIME_CGROUP_ROOT:-}" ]; then
        LIME_CGROUP_ROOT="/sys/fs/cgroup/lime.slice/test"
        mkdir -p "$LIME_CGROUP_ROOT"
        echo "+cpu +cpuset +memory +pids +io" > "$LIME_CGROUP_ROOT/cgroup.subtree_control" 2>/dev/null || true
    fi
    export LIME_INTEGRATION=1
    export LIME_CGROUP_ROOT
    export LIME_ROOTFS="${LIME_ROOTFS:-/rootfs}"
    echo "==> integration tests enabled (cgroup=$LIME_CGROUP_ROOT, rootfs=$LIME_ROOTFS)"
fi

echo "==> running tests..."
go test ./... -count=1 "${GOTEST_ARGS[@]}"
