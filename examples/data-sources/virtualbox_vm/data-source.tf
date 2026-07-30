data "virtualbox_vm" "example" {
  name = "example-vm"
}

output "vm_uuid" {
  value = data.virtualbox_vm.example.uuid
}
