resource "virtualbox_vm" "example" {
  name    = "example-vm"
  os_type = "Ubuntu_64"
  memory  = 2048
  cpus    = 2
  vram    = 16

  # Optional: create and attach a 20 GB disk (set to 0 to skip disk creation).
  disk_size_mb = 20480
}
