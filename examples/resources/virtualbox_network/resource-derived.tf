# A minimal host-only network. When the DHCP bounds are omitted they default to
# the first and last usable addresses of network_cidr — here 192.168.99.1 and
# 192.168.99.254.
resource "virtualbox_network" "minimal" {
  name         = "minimal-net"
  network_cidr = "192.168.99.0/24"
}
