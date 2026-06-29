package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceVirtualBoxVM_Basic(t *testing.T) {
	resourceName := "virtualbox_vm.test"
	dataSourceName := "data.virtualbox_vm.test"
	vmName := "vbox_test_acc_ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualBoxVMConfig(vmName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "memory", resourceName, "memory"),
					resource.TestCheckResourceAttrPair(dataSourceName, "cpus", resourceName, "cpus"),
				),
			},
		},
	})
}

func testAccDataSourceVirtualBoxVMConfig(name string) string {
	return fmt.Sprintf(`
resource "virtualbox_vm" "test" {
  name     = "%s"
  memory   = 128
  cpus     = 1
  os_type  = "Other"
  disk_size_mb = 0
}

data "virtualbox_vm" "test" {
  name = virtualbox_vm.test.name
}
`, name)
}
