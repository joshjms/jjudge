packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

source "qemu" "worker" {
  # ── Base image ────────────────────────────────────────────────────────────
  iso_url      = var.ubuntu_cloud_image_url
  iso_checksum = var.ubuntu_cloud_image_checksum
  disk_image   = true # iso_url is a pre-built disk image, not an ISO

  # ── Disk ──────────────────────────────────────────────────────────────────
  disk_size        = var.disk_size
  format           = "qcow2"
  output_directory = "output-worker"
  vm_name          = "jjudge-worker.qcow2"

  # ── CPU / Memory ──────────────────────────────────────────────────────────
  accelerator = "kvm"
  cpus        = var.cpus
  memory      = var.memory

  # ── Cloud-init (NoCloud datasource via CD-ROM) ────────────────────────────
  # Provides the temporary password and SSH config Packer needs to connect.
  cd_files = [
    "${path.root}/http/meta-data",
    "${path.root}/http/user-data",
  ]
  cd_label = "cidata"

  # ── SSH ───────────────────────────────────────────────────────────────────
  ssh_username = "ubuntu"
  ssh_password = var.ssh_password
  ssh_timeout  = "15m"

  # ── Shutdown ──────────────────────────────────────────────────────────────
  shutdown_command = "echo '${var.ssh_password}' | sudo -S shutdown -P now"

  headless = true
}

build {
  name    = "jjudge-worker"
  sources = ["source.qemu.worker"]

  # ── Upload Go workspace (needed to build the worker binary) ───────────────
  provisioner "file" {
    sources     = ["${path.root}/../go.work", "${path.root}/../go.work.sum"]
    destination = "/tmp/"
  }
  provisioner "file" {
    source      = "${path.root}/../api/"
    destination = "/tmp/api/"
  }
  provisioner "file" {
    source      = "${path.root}/../apiserver/"
    destination = "/tmp/apiserver/"
  }
  provisioner "file" {
    source      = "${path.root}/../grader/"
    destination = "/tmp/grader/"
  }
  provisioner "file" {
    source      = "${path.root}/../worker/"
    destination = "/tmp/worker/"
  }

  # ── Upload lime source ────────────────────────────────────────────────────
  provisioner "file" {
    source      = "${path.root}/../lime/"
    destination = "/tmp/lime/"
  }

  # ── Upload service / config files ─────────────────────────────────────────
  provisioner "file" {
    source      = "${path.root}/files/"
    destination = "/tmp/packer-files/"
  }

  # ── Provisioning scripts ──────────────────────────────────────────────────
  provisioner "shell" {
    environment_vars = [
      "GO_VERSION=${var.go_version}",
      "DEBIAN_FRONTEND=noninteractive",
    ]
    execute_command = "echo '${var.ssh_password}' | sudo -SE bash '{{ .Path }}'"
    scripts = [
      "${path.root}/scripts/provision.sh",
      "${path.root}/scripts/setup-rootfs.sh",
      "${path.root}/scripts/setup-services.sh",
    ]
  }

  # ── Clean up build artefacts ─────────────────────────────────────────────
  provisioner "shell" {
    execute_command = "echo '${var.ssh_password}' | sudo -S bash -c '{{ .Vars }} {{ .Path }}'"
    inline = [
      "rm -rf /tmp/api /tmp/apiserver /tmp/grader /tmp/worker /tmp/lime",
      "rm -f /tmp/go.work /tmp/go.work.sum",
      "rm -rf /tmp/packer-files",
      "rm -rf /usr/local/go",           # remove Go toolchain from runtime image
      "apt-get purge -y gcc make 2>/dev/null || true",
      "apt-get autoremove -y",
      "apt-get clean",
      "sync",
    ]
  }
}
