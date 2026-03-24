#!/usr/bin/env bash
# jjudge-cgroup-setup.sh — create and delegate the lime.slice cgroup subtree.
#
# Called by jjudge-cgroup.service (Type=oneshot) before the worker starts.
# Requires CAP_SYS_ADMIN (the service runs as root for this one step).
set -euo pipefail

LIME_SLICE="/sys/fs/cgroup/lime.slice"

# Create the slice directory if it does not exist.
mkdir -p "${LIME_SLICE}"

# Enable the controllers the worker needs.
echo "+cpu +cpuset +memory +pids +io" > "${LIME_SLICE}/cgroup.subtree_control"

# Delegate ownership to the ubuntu user (uid/gid 1000) so the worker
# can create child cgroups without elevated privileges.
chown -R 1000:1000 "${LIME_SLICE}"

echo "jjudge-cgroup-setup: lime.slice ready at ${LIME_SLICE}"
