# A minimal virtual machine: 2 vCPUs, 2 GB RAM, and a 20 GB disk.
resource "virtualbox_vm" "web" {
  name    = "web-server"
  os_type = "Ubuntu_64"
  memory  = 2048
  cpus    = 2

  disk_size_mb = 20480
}
