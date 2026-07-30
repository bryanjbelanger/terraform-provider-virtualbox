resource "virtualbox_network" "example" {
  name          = "example-network"
  network_cidr  = "192.168.56.0/24"
  dhcp          = true
  dhcp_lower_ip = "192.168.56.100"
  dhcp_upper_ip = "192.168.56.200"
}
