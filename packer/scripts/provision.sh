#!/usr/bin/env bash
# provision.sh — install system packages, build lime and worker binaries.
# Runs as root inside the Packer build VM.
set -euo pipefail

GO_VERSION="${GO_VERSION:-1.25.0}"

echo "==> provision: updating apt..."
apt-get update -qq

echo "==> provision: installing build dependencies..."
apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gcc \
    make \
    debootstrap \
    uidmap

echo "==> provision: installing Go ${GO_VERSION}..."
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz
export PATH="/usr/local/go/bin:${PATH}"
go version

echo "==> provision: building lime..."
mkdir -p /tmp/lime/build
cd /tmp/lime
make
install -m 755 build/lime /usr/local/bin/lime
echo "==> provision: lime installed at /usr/local/bin/lime"

echo "==> provision: building worker..."
mkdir -p /build
cp /tmp/go.work /tmp/go.work.sum /build/
cp -r /tmp/api /tmp/apiserver /tmp/grader /tmp/worker /build/

cd /build
PATH="/usr/local/go/bin:${PATH}" \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /usr/local/bin/worker ./worker/
chmod +x /usr/local/bin/worker
install -m 755 /tmp/worker/entrypoint.sh /usr/local/bin/entrypoint.sh
rm -rf /build
echo "==> provision: worker installed at /usr/local/bin/worker"

echo "==> provision: creating runtime directories..."
mkdir -p /tmp/judge/submissions /tmp/judge/work /tmp/judge/overlayfs /tmp/judge/rootfs
chown -R 1000:1000 /tmp/judge

mkdir -p /rootfs
chown 1000:1000 /rootfs

echo "==> provision: configuring subuid/subgid..."
grep -qF 'ubuntu:100000:65536' /etc/subuid 2>/dev/null || \
    echo 'ubuntu:100000:65536' >> /etc/subuid
grep -qF 'ubuntu:100000:65536' /etc/subgid 2>/dev/null || \
    echo 'ubuntu:100000:65536' >> /etc/subgid

echo "==> provision: enabling user-namespace sandboxing..."
sysctl -w kernel.unprivileged_userns_clone=1 2>/dev/null || true
echo 'kernel.unprivileged_userns_clone=1' > /etc/sysctl.d/60-jjudge-userns.conf

if [ -f /proc/sys/kernel/apparmor_restrict_unprivileged_userns ]; then
    sysctl -w kernel.apparmor_restrict_unprivileged_userns=0 2>/dev/null || true
    echo 'kernel.apparmor_restrict_unprivileged_userns=0' \
        >> /etc/sysctl.d/60-jjudge-userns.conf
fi

echo "==> provision: verifying cgroup v2..."
if ! grep -q cgroup2 /proc/filesystems; then
    echo "ERROR: cgroup v2 is not available in this kernel" >&2
    exit 1
fi

echo "==> provision: done."
