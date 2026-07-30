# Create several identically-sized worker VMs with for_each.
resource "virtualbox_vm" "worker" {
  for_each = toset(["worker-1", "worker-2", "worker-3"])

  name    = each.key
  os_type = "Ubuntu_64"
  memory  = 1024
  cpus    = 1

  disk_size_mb = 10240
}
