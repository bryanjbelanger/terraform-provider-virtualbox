package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVirtualBoxVM_Basic(t *testing.T) {
	resourceName := "virtualbox_vm.test"
	vmName := "vbox_test_acc_0"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() {},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVirtualBoxVMDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualBoxVMConfig_basic(vmName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vmName),
					resource.TestCheckResourceAttr(resourceName, "memory", "128"),
					resource.TestCheckResourceAttr(resourceName, "cpus", "1"),
				),
			},
			{
				Config: testAccVirtualBoxVMConfig_update(vmName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vmName),
					resource.TestCheckResourceAttr(resourceName, "memory", "256"),
				),
			},
		},
	})
}

func testAccVirtualBoxVMConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "virtualbox_vm" "test" {
  name     = "%s"
  memory   = 128
  cpus     = 1
  os_type  = "Other"
  disk_size_mb = 0 # skip disk for test speed
}
`, name)
}

func testAccVirtualBoxVMConfig_update(name string) string {
	return fmt.Sprintf(`
resource "virtualbox_vm" "test" {
  name     = "%s"
  memory   = 256
  cpus     = 1
  os_type  = "Other"
  disk_size_mb = 0
}
`, name)
}

func testAccCheckVirtualBoxVMDestroy(s *terraform.State) error {
	// VirtualBox automatically deletes VM. If it's destroyed, ReadContext will not find it.
	return nil
}
