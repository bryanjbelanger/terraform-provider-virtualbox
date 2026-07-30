# A read-only shared folder, auto-mounted in the guest (requires Guest Additions).
resource "virtualbox_shared_folder" "config" {
  vm_name   = virtualbox_vm.web.name
  name      = "config"
  host_path = "/srv/config"
  writable  = false
  automount = true
}
