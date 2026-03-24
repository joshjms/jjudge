#!/usr/bin/env bash
# setup-services.sh — install systemd units that run the jjudge worker on boot.
set -euo pipefail

FILES_DIR="/tmp/packer-files"

echo "==> setup-services: installing systemd units..."

install -m 755 "${FILES_DIR}/jjudge-cgroup-setup.sh"   /usr/local/bin/jjudge-cgroup-setup.sh
install -m 644 "${FILES_DIR}/jjudge-cgroup.service"    /etc/systemd/system/jjudge-cgroup.service
install -m 644 "${FILES_DIR}/jjudge-worker.service"    /etc/systemd/system/jjudge-worker.service

# ── Config directory and example env file ─────────────────────────────────────
mkdir -p /etc/jjudge
install -m 640 "${FILES_DIR}/worker.env.example" /etc/jjudge/worker.env.example
chown root:ubuntu /etc/jjudge/worker.env.example

echo "==> setup-services: enabling systemd units..."
systemctl daemon-reload
systemctl enable jjudge-cgroup.service
systemctl enable jjudge-worker.service

echo "==> setup-services: done."
echo ""
echo "    To start the worker, copy /etc/jjudge/worker.env.example to"
echo "    /etc/jjudge/worker.env, fill in the required values, and run:"
echo "      sudo systemctl start jjudge-worker"
