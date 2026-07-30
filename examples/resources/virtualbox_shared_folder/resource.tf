# A writable shared folder, auto-mounted in the guest (requires Guest Additions).
resource "virtualbox_shared_folder" "data" {
  vm_name   = virtualbox_vm.web.name
  name      = "data"
  host_path = "/srv/data"
  writable  = true
  automount = true
}
