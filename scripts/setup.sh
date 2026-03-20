#!/usr/bin/env bash
# scripts/setup.sh — one-time Ubuntu setup for JJudge (single-node)
#
# Run as root or with sudo:
#   sudo bash scripts/setup.sh
#
# What this does:
#   1. Installs Docker CE, Docker Compose plugin, Node.js 22, build tools
#   2. Ensures cgroup v2 unified hierarchy (edits GRUB if needed)
#   3. Creates a persistent systemd service to provision lime.slice
#   4. Configures AppArmor / sysctl for lime's user-namespace sandboxing
#   5. Adds the invoking user to the docker group
#   6. Isolate CPU via cgroups
#   7. Creates .env at the project root with sane defaults
#   8. Installs frontend npm dependencies

set -euo pipefail

# ── Helpers ──────────────────────────────────────────────────────────────────

RED='\033[0;31m'; YELLOW='\033[1;33m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
info()    { echo -e "${CYAN}[setup]${NC} $*"; }
success() { echo -e "${GREEN}[setup]${NC} $*"; }
warn()    { echo -e "${YELLOW}[setup]${NC} $*"; }
die()     { echo -e "${RED}[setup] ERROR:${NC} $*" >&2; exit 1; }

# ── Root check ────────────────────────────────────────────────────────────────

[[ $EUID -eq 0 ]] || die "Run this script as root:  sudo bash scripts/setup.sh"

# Capture the user who invoked sudo (or root if run directly)
REAL_USER="${SUDO_USER:-root}"
REAL_HOME=$(getent passwd "$REAL_USER" | cut -d: -f6)

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ── OS check ──────────────────────────────────────────────────────────────────

if ! grep -qi ubuntu /etc/os-release 2>/dev/null; then
    warn "This script targets Ubuntu. Proceeding anyway, but some steps may not apply."
fi

. /etc/os-release
UBUNTU_VERSION="${VERSION_ID:-unknown}"
info "Detected: $PRETTY_NAME"

# ── 1. System packages ────────────────────────────────────────────────────────

info "Updating apt and installing base packages..."
apt-get update -qq
apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    build-essential \
    gcc \
    make \
    git \
    uidmap \
    jq \
    net-tools \
    apt-transport-https \
    software-properties-common

# ── 2. Docker CE ──────────────────────────────────────────────────────────────

if ! command -v docker &>/dev/null; then
    info "Installing Docker CE..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
        https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable --now docker
    success "Docker installed: $(docker --version)"
else
    success "Docker already installed: $(docker --version)"
fi

# Ensure Docker Compose plugin is available
if ! docker compose version &>/dev/null; then
    info "Installing docker-compose-plugin..."
    apt-get install -y docker-compose-plugin
fi
success "Docker Compose: $(docker compose version --short)"

# ── 3. Node.js 22 ─────────────────────────────────────────────────────────────

if ! command -v node &>/dev/null || [[ "$(node -e 'process.stdout.write(process.version.slice(1).split(".")[0])')" -lt 22 ]]; then
    info "Installing Node.js 22 via NodeSource..."
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt-get install -y nodejs
    success "Node.js installed: $(node --version)"
else
    success "Node.js already installed: $(node --version)"
fi

# ── 4. Add user to docker group ───────────────────────────────────────────────

if [[ "$REAL_USER" != "root" ]]; then
    if ! id -nG "$REAL_USER" | grep -qw docker; then
        info "Adding $REAL_USER to the docker group..."
        usermod -aG docker "$REAL_USER"
        warn "You will need to log out and back in (or run 'newgrp docker') for docker group to take effect."
    else
        success "$REAL_USER is already in the docker group."
    fi
fi

# ── 5. cgroup v2 (unified hierarchy) ─────────────────────────────────────────
#
# lime requires cgroup v2. Docker's cgroup: host also works best on v2.
# On Ubuntu 20.04 the default is cgroup v1; on 21.10+ it is v2 already.

CGROUP_TYPE=$(stat -fc %T /sys/fs/cgroup/ 2>/dev/null || echo "unknown")
NEED_REBOOT=false
TOTAL_CPUS=$(nproc --all)

if [[ "$CGROUP_TYPE" != "cgroup2fs" ]]; then
    warn "cgroup v2 (unified) is NOT currently active (found: $CGROUP_TYPE)."
    info "Enabling cgroup v2 via GRUB kernel parameters..."

    GRUB_FILE=/etc/default/grub
    GRUB_BACKUP="${GRUB_FILE}.bak.$(date +%s)"
    cp "$GRUB_FILE" "$GRUB_BACKUP"
    info "GRUB config backed up to $GRUB_BACKUP"

    # Add parameters if not already present
    for PARAM in "systemd.unified_cgroup_hierarchy=1" "cgroup_no_v1=all"; do
        if ! grep -q "$PARAM" "$GRUB_FILE"; then
            sed -i "s|^GRUB_CMDLINE_LINUX=\"\(.*\)\"|GRUB_CMDLINE_LINUX=\"\1 $PARAM\"|" "$GRUB_FILE"
        fi
    done

    update-grub
    NEED_REBOOT=true
    warn "cgroup v2 will be active after reboot. Run setup.sh again after rebooting."
else
    success "cgroup v2 unified hierarchy is active."
fi

# Enable required cgroup controllers at the root level (best-effort;
# some may already be active — errors here are non-fatal).
if [[ "$CGROUP_TYPE" == "cgroup2fs" ]]; then
    info "Enabling cgroup v2 controllers at root level..."
    for ctrl in cpu cpuset memory pids io; do
        echo "+${ctrl}" > /sys/fs/cgroup/cgroup.subtree_control 2>/dev/null || true
    done
fi

# ── 6. lime.slice — persistent cgroup slice ───────────────────────────────────
#
# docker-compose.yml declares:
#   cgroup: host
#   cgroup_parent: lime.slice
# so /sys/fs/cgroup/lime.slice must exist before docker starts the worker.
#
# We install a small systemd service that (re)creates the slice at boot,
# enables its controllers, and delegates ownership to uid 1000 so the
# worker process can manage sub-cgroups without root.

LIME_SERVICE=/etc/systemd/system/jjudge-cgroup.service

info "Installing jjudge-cgroup.service..."
cat > "$LIME_SERVICE" <<'EOF'
[Unit]
Description=Create and configure lime.slice cgroup for JJudge worker
Documentation=https://github.com/jjudge-oj
# Must be ready before Docker starts
Before=docker.service
After=-.mount
DefaultDependencies=no

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/jjudge-cgroup-init.sh

[Install]
WantedBy=multi-user.target
EOF

cat > /usr/local/bin/jjudge-cgroup-init.sh <<'EOF'
#!/bin/bash
# Provision /sys/fs/cgroup/lime.slice for the JJudge worker.
set -e

SLICE=/sys/fs/cgroup/lime.slice
WORKER_UID=1000   # uid of the 'ubuntu' user inside the worker container

# Enable required controllers at the root cgroup
for ctrl in cpu cpuset memory pids io; do
    echo "+${ctrl}" > /sys/fs/cgroup/cgroup.subtree_control 2>/dev/null || true
done

# Create the slice directory
mkdir -p "$SLICE"

# Delegate controllers into the slice
echo "+cpu +cpuset +memory +pids +io" > "$SLICE/cgroup.subtree_control" 2>/dev/null || true

# Hand ownership to the worker uid so lime can manage sub-cgroups without root
chown "${WORKER_UID}:${WORKER_UID}" "$SLICE"
chown "${WORKER_UID}:${WORKER_UID}" "$SLICE/cgroup.procs"    2>/dev/null || true
chown "${WORKER_UID}:${WORKER_UID}" "$SLICE/cgroup.subtree_control" 2>/dev/null || true

echo "jjudge-cgroup: lime.slice ready at $SLICE"
EOF

chmod +x /usr/local/bin/jjudge-cgroup-init.sh

systemctl daemon-reload
systemctl enable jjudge-cgroup.service

if [[ "$CGROUP_TYPE" == "cgroup2fs" ]]; then
    systemctl start jjudge-cgroup.service
    success "lime.slice provisioned and jjudge-cgroup.service enabled."
else
    warn "jjudge-cgroup.service installed but not started (waiting for cgroup v2 reboot)."
fi

# ── 7. AppArmor / sysctl — user-namespace sandboxing ─────────────────────────
#
# lime uses Linux user namespaces for its sandbox. Two things can block this:
#
# a) Ubuntu 24.04+ restricts *unprivileged* user-namespace creation via AppArmor
#    (kernel.apparmor_restrict_unprivileged_userns = 1). The worker container
#    runs with privileged: true so AppArmor does NOT apply to it — the worker
#    is already unaffected. We still disable the restriction so lime can be
#    tested outside Docker if needed.
#
# b) Older Ubuntu / Debian kernels gate unpriv userns behind
#    kernel.unprivileged_userns_clone. We ensure it is on.

SYSCTL_CONF=/etc/sysctl.d/99-jjudge.conf

info "Configuring sysctl / AppArmor for user-namespace support..."
cat > "$SYSCTL_CONF" <<'EOF'
# JJudge / lime: allow unprivileged user-namespace creation
kernel.unprivileged_userns_clone = 1

# Ubuntu 24.04 AppArmor restriction — disable so lime can run outside Docker
# (privileged containers already bypass this; this is for bare-metal testing)
kernel.apparmor_restrict_unprivileged_userns = 0
EOF

sysctl --system -q 2>/dev/null || true   # suppress "missing file" warnings on older kernels
success "sysctl rules written to $SYSCTL_CONF"

# Load immediately (best-effort — unknown keys on older kernels are harmless)
sysctl -w kernel.unprivileged_userns_clone=1 2>/dev/null || true
sysctl -w kernel.apparmor_restrict_unprivileged_userns=0 2>/dev/null || true

# AppArmor: load permissive profile for lime if AppArmor is active
if command -v aa-status &>/dev/null && aa-status --enabled 2>/dev/null; then
    info "AppArmor is active. Ensuring Docker's default profile allows user namespaces..."
    # The privileged: true flag in docker-compose tells Docker to run the
    # container with --security-opt apparmor=unconfined, so no extra profile
    # is needed. This message is purely informational.
    success "Docker privileged containers are exempt from AppArmor profiles."
fi

# ── 8. subuid / subgid for newuidmap / newgidmap ─────────────────────────────
#
# The worker image ships with uid 1000 (ubuntu). The Dockerfile already writes
# sub-ranges into /etc/subuid and /etc/subgid inside the image, but the host
# also needs a range for the REAL_USER so tools like newuidmap work if lime is
# ever invoked directly on the host.

if [[ "$REAL_USER" != "root" ]]; then
    if ! grep -q "^${REAL_USER}:" /etc/subuid 2>/dev/null; then
        info "Adding subuid range for $REAL_USER..."
        usermod --add-subuids 100000-165535 "$REAL_USER" 2>/dev/null || \
            echo "${REAL_USER}:100000:65536" >> /etc/subuid
    fi
    if ! grep -q "^${REAL_USER}:" /etc/subgid 2>/dev/null; then
        info "Adding subgid range for $REAL_USER..."
        usermod --add-subgids 100000-165535 "$REAL_USER" 2>/dev/null || \
            echo "${REAL_USER}:100000:65536" >> /etc/subgid
    fi
    success "subuid/subgid configured for $REAL_USER."
fi

# ── 9. Docker daemon — use systemd cgroup driver ─────────────────────────────
#
# When cgroup v2 is active Docker should use the systemd cgroup driver
# (rather than cgroupfs) to avoid double management.

DOCKER_DAEMON=/etc/docker/daemon.json
if [[ ! -f "$DOCKER_DAEMON" ]] || ! grep -q '"exec-opts"' "$DOCKER_DAEMON" 2>/dev/null; then
    info "Configuring Docker daemon to use systemd cgroup driver..."
    mkdir -p /etc/docker
    if [[ -f "$DOCKER_DAEMON" ]]; then
        # Merge rather than overwrite
        EXISTING=$(cat "$DOCKER_DAEMON")
        echo "$EXISTING" | jq '. + {"exec-opts": ["native.cgroupdriver=systemd"], "log-driver": "json-file", "log-opts": {"max-size": "100m"}}' \
            > "$DOCKER_DAEMON" 2>/dev/null || true
    else
        cat > "$DOCKER_DAEMON" <<'EOF'
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": { "max-size": "100m" }
}
EOF
    fi
    if systemctl is-active --quiet docker; then
        systemctl restart docker
        success "Docker daemon restarted with systemd cgroup driver."
    fi
fi

# ── 10. .env at project root ──────────────────────────────────────────────────

ENV_FILE="$PROJECT_ROOT/.env"
if [[ ! -f "$ENV_FILE" ]]; then
    info "Creating $ENV_FILE with default values..."
    cat > "$ENV_FILE" <<'EOF'
# JJudge environment — edit before deploying to production
# ──────────────────────────────────────────────────────────

# Admin credentials (auto-created on first start)
JJUDGE_ADMIN_USER=admin
JJUDGE_ADMIN_PASSWORD=changeme

# Public URL of the API server (used by the browser)
# Change to your server's IP or domain if not running locally
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080

# Internal URL used by the Next.js SSR layer inside Docker
API_BASE_URL=http://server:8080
EOF
    chown "$REAL_USER:$REAL_USER" "$ENV_FILE"
    success ".env created at $ENV_FILE — edit it before starting."
else
    success ".env already exists at $ENV_FILE — skipping."
fi

# Append LIME_CPUS to .env if not already present (compute a sensible default)
if ! grep -q '^LIME_CPUS=' "$ENV_FILE" 2>/dev/null; then
    LIME_HALF=$(( TOTAL_CPUS / 2 ))
    [[ $LIME_HALF -lt 1 ]] && LIME_HALF=1
    DEFAULT_LIME_CPUS="${LIME_HALF}-$(( TOTAL_CPUS - 1 ))"
    {
        echo ""
        echo "# CPUs reserved exclusively for the lime sandbox workers."
        echo "# Format: cpuset range/list, e.g. \"2-3\" or \"2,3\" or \"2-3,6-7\"."
        echo "# These CPUs are isolated from the Linux scheduler (init/system/user slices)"
        echo "# so that grading latency is not affected by other system processes."
        echo "LIME_CPUS=${DEFAULT_LIME_CPUS}"
    } >> "$ENV_FILE"
    info "LIME_CPUS=${DEFAULT_LIME_CPUS} appended to .env (last half of ${TOTAL_CPUS} CPUs)"
fi

# Also create frontend/.env.local if missing
FRONTEND_ENV="$PROJECT_ROOT/frontend/.env.local"
if [[ ! -f "$FRONTEND_ENV" ]]; then
    info "Creating $FRONTEND_ENV..."
    echo "NEXT_PUBLIC_API_BASE_URL=http://localhost:8080" > "$FRONTEND_ENV"
    chown "$REAL_USER:$REAL_USER" "$FRONTEND_ENV"
fi

# ── 11. CPU isolation for lime.slice ─────────────────────────────────────────
#
# Following https://documentation.ubuntu.com/real-time/latest/how-to/isolate-workload-cpusets/
#
# lime.slice (cgroup_parent for the worker containers) is restricted to LIME_CPUS.
# init.scope, system.slice, and user.slice are restricted to the complementary set
# so the scheduler never touches the lime CPUs for regular system work.

# Read LIME_CPUS from .env
LIME_CPUS_VALUE=$(grep -E '^LIME_CPUS=' "$ENV_FILE" 2>/dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" | xargs)

if [[ $TOTAL_CPUS -lt 2 ]]; then
    warn "Only ${TOTAL_CPUS} CPU detected — skipping CPU isolation."
elif [[ -z "$LIME_CPUS_VALUE" ]]; then
    warn "LIME_CPUS not set in .env — skipping CPU isolation."
else
    # Compute the complement (system CPUs = all CPUs minus LIME_CPUS)
    SYSTEM_CPUS=$(python3 - <<PYEOF
total = $TOTAL_CPUS
lime = set()
for part in "${LIME_CPUS_VALUE}".split(","):
    part = part.strip()
    if "-" in part:
        a, b = part.split("-", 1)
        lime.update(range(int(a), int(b) + 1))
    elif part:
        lime.add(int(part))
system = sorted(i for i in range(total) if i not in lime)
if not system:
    print("0")
else:
    ranges, start, end = [], system[0], system[0]
    for cpu in system[1:]:
        if cpu == end + 1:
            end = cpu
        else:
            ranges.append(str(start) if start == end else f"{start}-{end}")
            start = end = cpu
    ranges.append(str(start) if start == end else f"{start}-{end}")
    print(",".join(ranges))
PYEOF
)

    info "CPU isolation: lime.slice → ${LIME_CPUS_VALUE}  |  system → ${SYSTEM_CPUS}"

    # ── Persistent drop-in config files ──────────────────────────────────────
    mkdir -p /etc/systemd/system/init.scope.d/
    mkdir -p /etc/systemd/system/system.slice.d/
    mkdir -p /etc/systemd/system/user.slice.d/

    cat > /etc/systemd/system/init.scope.d/50-cpu-isolation.conf <<EOF
[Scope]
AllowedCPUs=${SYSTEM_CPUS}
EOF

    for _unit in system.slice user.slice; do
        cat > /etc/systemd/system/${_unit}.d/50-cpu-isolation.conf <<EOF
[Slice]
AllowedCPUs=${SYSTEM_CPUS}
EOF
    done

    # Create (or update) lime.slice as a real systemd unit so AllowedCPUs persists
    # across reboots and systemd handles cgroup activation before Docker starts.
    cat > /etc/systemd/system/lime.slice <<EOF
[Unit]
Description=lime sandbox workload slice for JJudge workers
Documentation=https://github.com/jjudge-oj
Before=docker.service

[Slice]
AllowedCPUs=${LIME_CPUS_VALUE}
EOF

    systemctl daemon-reload

    # ── Apply at runtime ──────────────────────────────────────────────────────
    if [[ "$CGROUP_TYPE" == "cgroup2fs" ]]; then
        systemctl set-property --runtime lime.slice   AllowedCPUs="${LIME_CPUS_VALUE}" 2>/dev/null || true
        systemctl set-property --runtime init.scope   AllowedCPUs="${SYSTEM_CPUS}"     2>/dev/null || true
        systemctl set-property --runtime system.slice AllowedCPUs="${SYSTEM_CPUS}"     2>/dev/null || true
        systemctl set-property --runtime user.slice   AllowedCPUs="${SYSTEM_CPUS}"     2>/dev/null || true
        success "CPU isolation applied at runtime (lime → ${LIME_CPUS_VALUE}, system → ${SYSTEM_CPUS})."
    else
        warn "CPU isolation config written but not applied at runtime — waiting for cgroup v2 reboot."
    fi
fi

# ── 12. Frontend npm dependencies ─────────────────────────────────────────────

info "Installing frontend npm dependencies..."
if [[ "$REAL_USER" != "root" ]]; then
    sudo -u "$REAL_USER" bash -c "cd '$PROJECT_ROOT/frontend' && npm install"
else
    (cd "$PROJECT_ROOT/frontend" && npm install)
fi
success "Frontend dependencies installed."

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  JJudge setup complete!${NC}"
echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
echo ""
echo "  Project root : $PROJECT_ROOT"
echo "  .env         : $ENV_FILE"
echo ""
echo "  Next steps:"

if $NEED_REBOOT; then
    echo -e "  ${RED}1. REBOOT required to activate cgroup v2.${NC}"
    echo "     After reboot, run 'sudo bash scripts/setup.sh' once more"
    echo "     to provision lime.slice, then proceed below."
    echo ""
else
    echo "  1. Review / edit .env if needed:"
    echo "       $ENV_FILE"
    echo ""
    echo "  2. Start development stack:"
    echo "       make dev"
    echo ""
    echo "  3. Or deploy in the background:"
    echo "       make deploy"
    echo ""
    if [[ "$REAL_USER" != "root" ]]; then
        echo -e "  ${YELLOW}NOTE: Log out and back in (or run 'newgrp docker') so${NC}"
        echo -e "  ${YELLOW}      your user can run docker without sudo.${NC}"
    fi
fi
echo ""
