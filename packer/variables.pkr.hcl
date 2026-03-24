variable "ubuntu_cloud_image_url" {
  description = "URL of the Ubuntu 24.04 cloud image"
  type        = string
  default     = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
}

variable "ubuntu_cloud_image_checksum" {
  description = "SHA256 checksum of the Ubuntu 24.04 cloud image (prefix with 'sha256:')"
  type        = string
  default     = "file:https://cloud-images.ubuntu.com/noble/current/SHA256SUMS"
}

variable "ssh_password" {
  description = "Temporary SSH password used during Packer provisioning (not persisted)"
  type        = string
  default     = "packer"
  sensitive   = true
}

variable "go_version" {
  description = "Go toolchain version used to build the worker binary"
  type        = string
  default     = "1.25.0"
}

variable "disk_size" {
  description = "VM disk size in MiB"
  type        = number
  default     = 20480 # 20 GiB
}

variable "memory" {
  description = "RAM allocated to the Packer build VM in MiB"
  type        = number
  default     = 4096
}

variable "cpus" {
  description = "vCPUs allocated to the Packer build VM"
  type        = number
  default     = 4
}
