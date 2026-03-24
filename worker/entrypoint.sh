#!/bin/sh
# Install rootfs, then set up cgroup v2 hierarchy for lime.
#
# The worker runs as a non-root user whose systemd session manager has already
# delegated a cgroup subtree (user.slice/user-1000.slice/user@1000.service/)
# with all required controllers enabled. We simply create a lime subdirectory
# within LIME_CGROUP_ROOT — no SYS_ADMIN or privileged container needed.

# ── Rootfs installation ──────────────────────────────────────────────────────
_ROOTFS_DIR="${JUDGE_ROOTFS_DIR:-/rootfs}"

if [ ! -f "$_ROOTFS_DIR/.installed" ]; then
    if [ -z "$ROOTFS_IMG_SRC" ]; then
        echo "entrypoint: rootfs not found at $_ROOTFS_DIR and ROOTFS_IMG_SRC is not set" >&2
        exit 1
    fi
    echo "entrypoint: installing rootfs from $ROOTFS_IMG_SRC ..."
    mkdir -p "$_ROOTFS_DIR"
    _ROOTFS_TMP="$(mktemp)"
    if ! curl -fsSL "$ROOTFS_IMG_SRC" -o "$_ROOTFS_TMP"; then
        echo "entrypoint: failed to download rootfs from $ROOTFS_IMG_SRC" >&2
        rm -f "$_ROOTFS_TMP"
        exit 1
    fi
    if ! tar -xz --strip-components=1 -C "$_ROOTFS_DIR" < "$_ROOTFS_TMP"; then
        echo "entrypoint: failed to extract rootfs archive" >&2
        rm -f "$_ROOTFS_TMP"
        exit 1
    fi
    rm -f "$_ROOTFS_TMP"
    touch "$_ROOTFS_DIR/.installed"
    echo "entrypoint: rootfs installed at $_ROOTFS_DIR"
else
    echo "entrypoint: rootfs already installed at $_ROOTFS_DIR, skipping"
fi

# ── Cgroup v2 setup ──────────────────────────────────────────────────────────
if [ -z "$LIME_CGROUP_ROOT" ]; then
    echo "entrypoint: LIME_CGROUP_ROOT is not set" >&2
    exit 1
fi

mkdir -p "$LIME_CGROUP_ROOT"
echo "+cpu +cpuset +memory +pids +io" > "$LIME_CGROUP_ROOT/cgroup.subtree_control"
echo "entrypoint: lime cgroup ready at $LIME_CGROUP_ROOT"

exec /usr/local/bin/worker "$@"
