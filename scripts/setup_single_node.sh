#!/usr/bin/env bash
# scripts/setup_single_node.sh
#
# One-time host setup for running jjudge on a single Ubuntu 24.04 node.
#
# What it does:
#   1. Installs Docker (if not already present)
#   2. Adds the calling user to the docker group
#   3. Installs /usr/local/bin/jjudge-cgroup-init.sh
#   4. Installs and enables a systemd service that creates the lime.slice
#      cgroup hierarchy before Docker starts
#
# Usage:
#   sudo bash scripts/setup_single_node.sh
#   sudo bash scripts/setup_single_node.sh --user <username>   # explicit user for docker group

set -euo pipefail

# ── helpers ───────────────────────────────────────────────────────────────────

info()  { echo "[setup] $*"; }
warn()  { echo "[setup] WARNING: $*" >&2; }
die()   { echo "[setup] ERROR: $*" >&2; exit 1; }

require_root() {
    [ "$(id -u)" -eq 0 ] || die "must be run as root (sudo bash $0)"
}

# ── args ──────────────────────────────────────────────────────────────────────

SUDO_USER_NAME="${SUDO_USER:-}"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --user) SUDO_USER_NAME="$2"; shift 2 ;;
        *)      die "unknown argument: $1" ;;
    esac
done

require_root

# ── 1. OS check ───────────────────────────────────────────────────────────────

if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [[ "${ID:-}" != "ubuntu" ]]; then
        warn "this script targets Ubuntu; detected '${ID:-unknown}' — proceeding anyway"
    fi
fi

# Verify cgroup v2
if ! grep -q cgroup2 /proc/filesystems 2>/dev/null; then
    die "cgroup v2 is not available on this kernel"
fi
if ! mountpoint -q /sys/fs/cgroup; then
    die "/sys/fs/cgroup is not mounted"
fi
if ! grep -q "cgroup2" /proc/mounts 2>/dev/null; then
    die "/sys/fs/cgroup is not a cgroup v2 mount — this host may be using cgroup v1"
fi

info "OS and cgroup v2 check passed"

# ── 2. Docker ─────────────────────────────────────────────────────────────────

if command -v docker &>/dev/null; then
    info "Docker already installed: $(docker --version)"
else
    info "Installing Docker..."
    apt-get update -qq
    apt-get install -y -qq ca-certificates curl gnupg

    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg

    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
        https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
        > /etc/apt/sources.list.d/docker.list

    apt-get update -qq
    apt-get install -y -qq \
        docker-ce docker-ce-cli containerd.io \
        docker-buildx-plugin docker-compose-plugin

    systemctl enable --now docker
    info "Docker installed: $(docker --version)"
fi

# ── 3. Add user to docker group ───────────────────────────────────────────────

if [ -n "$SUDO_USER_NAME" ]; then
    if id "$SUDO_USER_NAME" &>/dev/null; then
        if groups "$SUDO_USER_NAME" | grep -qw docker; then
            info "User '$SUDO_USER_NAME' is already in the docker group"
        else
            usermod -aG docker "$SUDO_USER_NAME"
            info "Added '$SUDO_USER_NAME' to the docker group (log out and back in to apply)"
        fi
    else
        warn "User '$SUDO_USER_NAME' does not exist — skipping docker group setup"
    fi
else
    warn "No user specified for docker group (pass --user <username> or run via sudo)"
fi

# ── 4. cgroup init script ─────────────────────────────────────────────────────

CGROUP_INIT_SCRIPT=/usr/local/bin/jjudge-cgroup-init.sh

info "Installing $CGROUP_INIT_SCRIPT..."

cat > "$CGROUP_INIT_SCRIPT" << 'EOF'
#!/usr/bin/env bash
# jjudge-cgroup-init.sh
# Creates the lime.slice cgroup hierarchy required by the worker container.
# Run at boot (via systemd) and on-demand by 'make dev/deploy'.

set -euo pipefail

LIME_SLICE=/sys/fs/cgroup/lime.slice

mkdir -p "$LIME_SLICE"

# Enable the controllers that lime uses for sandboxing.
# These must already be available in the root cgroup's subtree_control.
CONTROLLERS="+cpu +cpuset +memory +pids +io"
if ! echo "$CONTROLLERS" > "$LIME_SLICE/cgroup.subtree_control" 2>/dev/null; then
    # Verify which controllers are actually available
    AVAILABLE=$(cat /sys/fs/cgroup/cgroup.controllers 2>/dev/null || echo "")
    echo "jjudge-cgroup-init: warning: could not enable all controllers" >&2
    echo "jjudge-cgroup-init: available controllers: $AVAILABLE" >&2
fi

echo "jjudge-cgroup-init: lime.slice ready"
EOF

chmod +x "$CGROUP_INIT_SCRIPT"
info "Installed $CGROUP_INIT_SCRIPT"

# ── 5. systemd service ────────────────────────────────────────────────────────

SYSTEMD_UNIT=/etc/systemd/system/jjudge-cgroup.service

info "Installing systemd unit $SYSTEMD_UNIT..."

cat > "$SYSTEMD_UNIT" << EOF
[Unit]
Description=jjudge lime cgroup hierarchy initialisation
# Must run before Docker so the cgroup exists when the worker container starts.
Before=docker.service
DefaultDependencies=no
After=local-fs.target systemd-cgroupsv2-fix-cpuset.service

[Service]
Type=oneshot
ExecStart=$CGROUP_INIT_SCRIPT
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable jjudge-cgroup.service
systemctl restart jjudge-cgroup.service

info "jjudge-cgroup.service enabled and started"

# ── 6. Verify ─────────────────────────────────────────────────────────────────

if [ -d /sys/fs/cgroup/lime.slice ]; then
    info "lime.slice cgroup exists at /sys/fs/cgroup/lime.slice"
    CTRL=$(cat /sys/fs/cgroup/lime.slice/cgroup.subtree_control 2>/dev/null || echo "(empty)")
    info "subtree_control: $CTRL"
else
    die "lime.slice cgroup was not created — check: journalctl -u jjudge-cgroup.service"
fi

# ── done ──────────────────────────────────────────────────────────────────────

echo ""
info "Setup complete. Next steps:"
if [ -n "$SUDO_USER_NAME" ]; then
    info "  1. Log out and back in as '$SUDO_USER_NAME' for docker group to take effect"
    info "  2. cd $(dirname "$(realpath "$0")")/.."
    info "  3. make dev    # build and start all services"
else
    info "  1. cd <repo root>"
    info "  2. make dev    # build and start all services"
fi
