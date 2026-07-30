resource "virtualbox_shared_folder" "example" {
  vm_name   = virtualbox_vm.example.name
  name      = "shared"
  host_path = "/tmp/shared"
  writable  = true
  automount = true
}
