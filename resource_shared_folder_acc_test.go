package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVirtualBoxSharedFolder_Basic(t *testing.T) {
	resourceName := "virtualbox_shared_folder.test"
	vmName := "vbox_test_acc_sf"
	folderName := "shared_data"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualBoxSharedFolderConfig(vmName, folderName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "vm_name", vmName),
					resource.TestCheckResourceAttr(resourceName, "name", folderName),
					resource.TestCheckResourceAttr(resourceName, "host_path", "/tmp"),
					resource.TestCheckResourceAttr(resourceName, "writable", "true"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        vmName + "/" + folderName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				// writable/automount are not reported by showvminfo, so they cannot
				// be recovered on a fresh import and are excluded from verification.
				ImportStateVerifyIgnore: []string{"writable", "automount"},
			},
		},
	})
}

func testAccVirtualBoxSharedFolderConfig(vmName, folderName string) string {
	return fmt.Sprintf(`
resource "virtualbox_vm" "test" {
  name         = "%s"
  memory       = 128
  cpus         = 1
  os_type      = "Other"
  disk_size_mb = 0
}

resource "virtualbox_shared_folder" "test" {
  vm_name   = virtualbox_vm.test.name
  name      = "%s"
  host_path = "/tmp"
  writable  = true
}
`, vmName, folderName)
}
