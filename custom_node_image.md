# Creating a Custom Worker Node Image Programmatically

This document explains how to build a custom OS image for jjudge worker nodes using
[Packer](https://www.packer.io). The image bakes in all host-level requirements for
`lime` so that nodes are ready to run worker pods immediately on boot, without any
runtime setup.

---

## Why Bake vs. Runtime Setup?

| Approach | How | Problem |
|---|---|---|
| **Baked image** | Packer builds image once; all nodes use it | Nodes are ready immediately |
| **cloud-init / DaemonSet** | Script runs on every new node at boot | Node is unavailable until script finishes; failure leaves node broken |
| **Privileged init container** | Pod runs setup before worker starts | Races with pod scheduling; re-runs every pod restart |

Baking is the most reliable approach for requirements that are hard to change at runtime
(kernel modules, subuid maps, grub parameters).

---

## Prerequisites

- [Packer](https://developer.hashicorp.com/packer/install) >= 1.10
- Cloud provider credentials:
  - **GCP**: `gcloud auth application-default login`
  - **AWS**: `aws configure` or `AWS_*` environment variables
- Sufficient IAM permissions to create and manage VM instances and disk images

---

## Repository Layout

Organize your Packer configuration alongside the rest of the project:

```
jjudge-project/
└── packer/
    ├── worker-node.pkr.hcl       # Packer template
    ├── variables.pkrvars.hcl     # Input variables (not committed if sensitive)
    └── scripts/
        └── setup.sh              # Provisioning script
```

---

## Provisioning Script

This script runs inside the VM during the build. It installs and configures everything
`lime` requires on the host.

```bash
# packer/scripts/setup.sh
#!/bin/bash
set -euo pipefail

echo "==> Installing uidmap (provides newuidmap/newgidmap)"
apt-get update -qq
apt-get install -y uidmap

echo "==> Creating judge user at UID 1000"
id -u 1000 &>/dev/null || useradd -u 1000 -m -s /bin/bash judge

echo "==> Configuring subordinate UID/GID ranges for UID 1000"
# These ranges allow lime to map up to 65536 UIDs inside a user namespace.
# Append only if not already present to make the script idempotent.
grep -q "^1000:" /etc/subuid || echo "1000:100000:65536" >> /etc/subuid
grep -q "^1000:" /etc/subgid || echo "1000:100000:65536" >> /etc/subgid

echo "==> Loading OverlayFS at boot"
echo "overlay" >> /etc/modules-load.d/overlay.conf
modprobe overlay

echo "==> Enabling cgroup v2"
# Adds the kernel parameter so cgroup v2 is the default hierarchy on boot.
sed -i 's/GRUB_CMDLINE_LINUX="\(.*\)"/GRUB_CMDLINE_LINUX="\1 systemd.unified_cgroup_hierarchy=1"/' \
    /etc/default/grub
update-grub

echo "==> Verifying setup"
stat -fc %T /sys/fs/cgroup | grep -q cgroup2fs \
    || echo "WARNING: cgroup v2 not active yet (will be after reboot)"
grep -q "^1000:" /etc/subuid || { echo "ERROR: subuid not set"; exit 1; }
which newuidmap || { echo "ERROR: newuidmap not found"; exit 1; }

echo "==> Done"
```

---

## Packer Template

### GCP

```hcl
# packer/worker-node.pkr.hcl

packer {
  required_version = ">= 1.10"
  required_plugins {
    googlecompute = {
      source  = "github.com/hashicorp/googlecompute"
      version = "~> 1"
    }
  }
}

variable "project_id" { type = string }
variable "zone"       { type = string  default = "us-central1-a" }
variable "image_name" { type = string  default = "jjudge-worker" }

source "googlecompute" "worker" {
  project_id          = var.project_id
  zone                = var.zone
  source_image_family = "ubuntu-2404-lts"
  machine_type        = "n2-standard-2"
  disk_size           = 50
  image_name          = "${var.image_name}-{{timestamp}}"
  image_family        = var.image_name
  image_description   = "jjudge worker node — lime runtime dependencies"
  ssh_username        = "packer"
}

build {
  name    = "jjudge-worker"
  sources = ["source.googlecompute.worker"]

  provisioner "shell" {
    script          = "scripts/setup.sh"
    execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
  }
}
```

### AWS

```hcl
# packer/worker-node.pkr.hcl

packer {
  required_version = ">= 1.10"
  required_plugins {
    amazon = {
      source  = "github.com/hashicorp/amazon"
      version = "~> 1"
    }
  }
}

variable "region"     { type = string  default = "us-east-1" }
variable "image_name" { type = string  default = "jjudge-worker" }

source "amazon-ebs" "worker" {
  region        = var.region
  instance_type = "t3.medium"
  ami_name      = "${var.image_name}-{{timestamp}}"
  ami_description = "jjudge worker node — lime runtime dependencies"

  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]  # Canonical
  }

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 50
    volume_type           = "gp3"
    delete_on_termination = true
  }

  ssh_username = "ubuntu"

  tags = {
    Name    = "${var.image_name}-{{timestamp}}"
    Project = "jjudge"
    Role    = "worker"
  }
}

build {
  name    = "jjudge-worker"
  sources = ["source.amazon-ebs.worker"]

  provisioner "shell" {
    script          = "scripts/setup.sh"
    execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
  }
}
```

---

## Variables File

Store non-sensitive input variables here. Do not commit secrets to version control.

```hcl
# packer/variables.pkrvars.hcl
project_id = "your-gcp-project-id"   # GCP only
region     = "us-east-1"              # AWS only
image_name = "jjudge-worker"
zone       = "us-central1-a"          # GCP only
```

---

## Building the Image

```bash
cd packer

# Download required Packer plugins
packer init worker-node.pkr.hcl

# Validate the template before building
packer validate -var-file=variables.pkrvars.hcl worker-node.pkr.hcl

# Build the image
packer build -var-file=variables.pkrvars.hcl worker-node.pkr.hcl
```

On success, Packer prints the image ID:
```
==> Builds finished. The artifacts of successful builds are:
--> jjudge-worker.googlecompute.worker: A disk image was created: jjudge-worker-1742123456
```

---

## Using the Image in Your Node Pool

### GKE

```bash
gcloud container node-pools create worker-pool \
  --cluster=<your-cluster> \
  --region=us-central1 \
  --machine-type=n2-standard-4 \
  --num-nodes=1 \
  --min-nodes=1 \
  --max-nodes=10 \
  --enable-autoscaling \
  --node-taints=jjudge=worker:NoSchedule \
  --node-labels=jjudge/role=worker \
  --image-type=CUSTOM \
  --image=projects/<your-project>/global/images/jjudge-worker-<timestamp>
```

To always use the latest image from the family instead of pinning a specific timestamp:
```bash
--image-family=jjudge-worker \
--image-project=<your-project>
```

### EKS

Create a launch template referencing the AMI, then attach it to a managed node group:

```bash
# Create launch template with the custom AMI
aws ec2 create-launch-template \
  --launch-template-name jjudge-worker \
  --version-description "lime runtime" \
  --launch-template-data '{
    "ImageId": "ami-0abc123...",
    "InstanceType": "t3.xlarge",
    "BlockDeviceMappings": [{
      "DeviceName": "/dev/sda1",
      "Ebs": { "VolumeSize": 50, "VolumeType": "gp3" }
    }]
  }'

# Create the node group using the launch template
aws eks create-nodegroup \
  --cluster-name <your-cluster> \
  --nodegroup-name worker-pool \
  --scaling-config minSize=1,maxSize=10,desiredSize=1 \
  --launch-template name=jjudge-worker,version=1 \
  --labels jjudge/role=worker \
  --taints key=jjudge,value=worker,effect=NO_SCHEDULE
```

---

## Automating Image Rebuilds (CI/CD)

Rebuild the image automatically when `packer/scripts/setup.sh` changes.

Example GitHub Actions workflow:

```yaml
# .github/workflows/build-worker-image.yml
name: Build Worker Node Image

on:
  push:
    paths:
      - packer/**

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2        # or aws-actions/configure-aws-credentials
        with:
          credentials_json: ${{ secrets.GCP_SA_KEY }}

      - uses: hashicorp/setup-packer@main
        with:
          version: "1.11"

      - run: packer init packer/worker-node.pkr.hcl

      - run: packer validate -var-file=packer/variables.pkrvars.hcl packer/worker-node.pkr.hcl

      - run: packer build -var-file=packer/variables.pkrvars.hcl packer/worker-node.pkr.hcl
        env:
          PKR_VAR_project_id: ${{ secrets.GCP_PROJECT_ID }}
```

---

## Verifying a Node Built From the Image

After a node boots, SSH in and run:

```bash
# cgroup v2 active
stat -fc %T /sys/fs/cgroup
# expected: cgroup2fs

# subuid/subgid configured for UID 1000
grep 1000 /etc/subuid /etc/subgid
# expected: /etc/subuid:1000:100000:65536
#           /etc/subgid:1000:100000:65536

# newuidmap/newgidmap available
which newuidmap newgidmap
# expected: /usr/bin/newuidmap  /usr/bin/newgidmap

# OverlayFS loaded
lsmod | grep overlay
# expected: overlay  ...
```

All four checks must pass before worker pods will run successfully.
