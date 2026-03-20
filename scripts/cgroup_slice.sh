#!/bin/bash
set -e

SLICE_PATH="/sys/fs/cgroup/lime.slice"

mkdir -p "$SLICE_PATH"

# These are the files lime needs to write to
chown 1000:1000 "$SLICE_PATH"
chown 1000:1000 "$SLICE_PATH/cgroup.procs"
chown 1000:1000 "$SLICE_PATH/cgroup.subtree_control"

echo "ok!"
