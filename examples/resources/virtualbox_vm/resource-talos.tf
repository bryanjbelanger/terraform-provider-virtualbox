# A Talos Linux node: boots from a prebuilt Talos disk image (cloned per VM with a
# fresh UUID), attached to a host-only network so the host and peer nodes can
# reach its Talos/Kubernetes API, plus a NAT adapter for pulling images.
resource "virtualbox_network" "talos" {
  name         = "talos-net"
  network_cidr = "192.168.56.0/24"
}

resource "virtualbox_vm" "talos_cp" {
  name    = "talos-cp-1"
  os_type = "Linux_64"
  memory  = 4096
  cpus    = 2

  disk_image = "/images/talos-amd64.vdi"

  network_adapter {
    type         = "hostonlynet"
    network_name = virtualbox_network.talos.name
    nic_type     = "virtio"
  }

  network_adapter {
    type = "nat"
  }
}
