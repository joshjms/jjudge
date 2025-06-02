#!/usr/bin/env bash

set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is not installed. Please install Docker to use this script."
    exit 1
fi

DOCKER_IMAGE=${DOCKER_IMAGE:-"ubuntu:latest"}
DST_DIR=${DST_DIR:-"./.rootfs"}

if [ ! -d "$DST_DIR" ]; then
    echo "Destination directory $DST_DIR does not exist. Creating it."
    mkdir -p "$DST_DIR"
fi

CONTAINER_ID=$(docker create "$DOCKER_IMAGE")
if [ -z "$CONTAINER_ID" ]; then
    echo "Failed to create Docker container from image $DOCKER_IMAGE."
    exit 1
fi

docker export "$CONTAINER_ID" | tar -x -C "$DST_DIR"
if [ $? -ne 0 ]; then
    echo "Failed to export Docker container to $DST_DIR."
    docker rm "$CONTAINER_ID" >/dev/null 2>&1
    exit 1
fi

docker rm "$CONTAINER_ID" >/dev/null 2>&1
echo "Docker container exported successfully to $DST_DIR."
