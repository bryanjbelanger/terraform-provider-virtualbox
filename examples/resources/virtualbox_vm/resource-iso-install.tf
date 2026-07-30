# A virtual machine that boots from an installation ISO onto a blank disk and is
# started automatically in headless mode. The boot order is set to DVD then disk.
resource "virtualbox_vm" "installer" {
  name    = "ubuntu-installer"
  os_type = "Ubuntu_64"
  memory  = 4096
  cpus    = 4
  vram    = 32

  disk_size_mb = 40960
  iso_path     = "/isos/ubuntu-24.04-live-server-amd64.iso"

  start_on_create = true
}
