#!/usr/bin/env bash
# setup-rootfs.sh — build the /rootfs execution environment used by lime.
#
# lime mounts /rootfs as the lower OverlayFS layer when sandboxing user code.
# The rootfs must contain all compilers and runtimes that the worker invokes:
#
#   C++ compilation:  /usr/local/bin/g++  (processor.go hardcodes this path)
#   C++ execution:    standard runtime libs (libstdc++, libc)
#   Python execution: /usr/bin/python3
#
# We use debootstrap to create a minimal Ubuntu 24.04 (Noble) filesystem.
set -euo pipefail

ROOTFS_DIR="${JUDGE_ROOTFS_DIR:-/rootfs}"

echo "==> setup-rootfs: creating minimal Ubuntu 24.04 rootfs at ${ROOTFS_DIR}..."

debootstrap \
    --variant=minbase \
    --include=\
g++,\
gcc,\
python3,\
libstdc++-13-dev,\
libc6-dev,\
binutils \
    noble \
    "${ROOTFS_DIR}" \
    http://archive.ubuntu.com/ubuntu

# ── Symlink /usr/bin/g++ → /usr/local/bin/g++ ────────────────────────────────
# processor.go calls the compiler as /usr/local/bin/g++ inside the sandbox.
# debootstrap installs g++ at /usr/bin/g++, so we add a symlink.
ln -sf /usr/bin/g++ "${ROOTFS_DIR}/usr/local/bin/g++"
ln -sf /usr/bin/gcc "${ROOTFS_DIR}/usr/local/bin/gcc"

# ── Minimal /etc/passwd and /etc/group so executables can resolve nobody ──────
# Some programs stat these files; keep them even in minbase.
[ -f "${ROOTFS_DIR}/etc/passwd" ] || echo "root:x:0:0:root:/root:/bin/sh" > "${ROOTFS_DIR}/etc/passwd"
[ -f "${ROOTFS_DIR}/etc/group"  ] || echo "root:x:0:"                     > "${ROOTFS_DIR}/etc/group"

# ── Mark the rootfs as installed ─────────────────────────────────────────────
touch "${ROOTFS_DIR}/.installed"

chown -R 1000:1000 "${ROOTFS_DIR}"

echo "==> setup-rootfs: rootfs ready at ${ROOTFS_DIR}."
