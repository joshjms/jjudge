#!/usr/bin/env bash

set -euo pipefail

SANDBOX_ROOT_DIR=/var/lib/sandbox

mkdir -p "$SANDBOX_ROOT_DIR"

dirs=("images" "inner" "jobs")

for dir in "${dirs[@]}"; do
    if [ ! -d "$SANDBOX_ROOT_DIR/$dir" ]; then
        echo "Creating directory $SANDBOX_ROOT_DIR/$dir"
        mkdir -p "$SANDBOX_ROOT_DIR/$dir"
    fi
done

DOCKER_IMAGE=ubuntu:latest DST_DIR="$SANDBOX_ROOT_DIR/images/ubuntu" scripts/rootfs.sh
