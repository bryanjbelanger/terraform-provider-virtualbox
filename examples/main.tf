terraform {
  required_providers {
    virtualbox = {
      source  = "bryanbelanger/virtualbox"
      version = "~> 0.1.0"
    }
  }
}

provider "virtualbox" {
  # Path to VBoxManage (defaults to "VBoxManage" found in PATH)
  # vboxmanage_path = "/usr/local/bin/VBoxManage"
}

# Create a host-only network
resource "virtualbox_network" "main" {
  name         = "main-network"
  network_cidr = "192.168.56.0/24"
  dhcp         = true
}

# Create a virtual machine
resource "virtualbox_vm" "example" {
  name    = "example-vm"
  os_type = "Ubuntu_64"
  memory  = 2048
  cpus    = 2
  vram    = 16
}

# Add a shared folder
resource "virtualbox_shared_folder" "example" {
  vm_name   = virtualbox_vm.example.name
  name      = "shared"
  host_path = "/tmp/shared"
  writable  = true
  automount = true
}

# Read back the VM info
data "virtualbox_vm" "example" {
  name = virtualbox_vm.example.name
}

output "vm_details" {
  value = {
    name   = data.virtualbox_vm.example.name
    uuid   = data.virtualbox_vm.example.uuid
    status = data.virtualbox_vm.example.status
    memory = data.virtualbox_vm.example.memory
    cpus   = data.virtualbox_vm.example.cpus
  }
}

output "network_name" {
  value = virtualbox_network.main.name
}

output "shared_folder_name" {
  value = virtualbox_shared_folder.example.name
}