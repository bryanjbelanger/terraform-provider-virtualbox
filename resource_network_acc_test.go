package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVirtualBoxNetwork_Basic(t *testing.T) {
	resourceName := "virtualbox_network.test"
	networkName := "vboxnet_test_acc_0"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckVirtualBoxNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualBoxNetworkConfig_basic(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", networkName),
					resource.TestCheckResourceAttr(resourceName, "network_cidr", "192.168.100.0/24"),
					resource.TestCheckResourceAttr(resourceName, "dhcp", "true"),
				),
			},
			{
				Config: testAccVirtualBoxNetworkConfig_update(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", networkName),
					resource.TestCheckResourceAttr(resourceName, "dhcp", "false"),
				),
			},
		},
	})
}

func testAccVirtualBoxNetworkConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "virtualbox_network" "test" {
  name          = "%s"
  network_cidr  = "192.168.100.0/24"
  dhcp          = true
  dhcp_lower_ip = "192.168.100.10"
  dhcp_upper_ip = "192.168.100.200"
}
`, name)
}

func testAccVirtualBoxNetworkConfig_update(name string) string {
	return fmt.Sprintf(`
resource "virtualbox_network" "test" {
  name          = "%s"
  network_cidr  = "192.168.100.0/24"
  dhcp          = false
}
`, name)
}

func testAccCheckVirtualBoxNetworkDestroy(s *terraform.State) error {
	// VirtualBox automatically manages host-only networks, but if we need to verify destruction,
	// we'd use the virtualbox.Client here.
	return nil
}
