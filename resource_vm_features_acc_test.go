package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVirtualBoxVM_NetworkAdapters verifies a VM can be given a host-only and
// a NAT adapter, and that the configuration round-trips through Read (the
// implicit post-apply plan check fails on any drift).
func TestAccVirtualBoxVM_NetworkAdapters(t *testing.T) {
	resourceName := "virtualbox_vm.test"
	vmName := "vbox_test_acc_nic"
	netName := "vboxnet_acc_nic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVMNetworkAdaptersConfig(netName, vmName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "network_adapter.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "network_adapter.0.type", "hostonlynet"),
					resource.TestCheckResourceAttr(resourceName, "network_adapter.0.network_name", netName),
					resource.TestCheckResourceAttr(resourceName, "network_adapter.0.nic_type", "virtio"),
					resource.TestCheckResourceAttrSet(resourceName, "network_adapter.0.mac_address"),
					resource.TestCheckResourceAttr(resourceName, "network_adapter.1.type", "nat"),
				),
			},
		},
	})
}

func testAccVMNetworkAdaptersConfig(netName, vmName string) string {
	return fmt.Sprintf(`
resource "virtualbox_network" "test" {
  name         = "%s"
  network_cidr = "192.168.88.0/24"
}

resource "virtualbox_vm" "test" {
  name         = "%s"
  os_type      = "Linux_64"
  memory       = 128
  disk_size_mb = 0

  network_adapter {
    type         = "hostonlynet"
    network_name = virtualbox_network.test.name
    nic_type     = "virtio"
  }

  network_adapter {
    type = "nat"
  }
}
`, netName, vmName)
}

// TestAccVirtualBoxVM_DiskImage verifies that an existing disk image is cloned
// to a per-VM disk and attached. A throwaway base image is created directly with
// VBoxManage before the test and removed afterward.
func TestAccVirtualBoxVM_DiskImage(t *testing.T) {
	resourceName := "virtualbox_vm.test"
	vmName := "vbox_test_acc_diskimg"
	base := filepath.Join(os.TempDir(), "tf_acc_base.vdi")

	makeBase := func() {
		_ = exec.Command("VBoxManage", "closemedium", "disk", base, "--delete").Run()
		_ = os.Remove(base)
		if out, err := exec.Command("VBoxManage", "createmedium", "disk",
			"--filename", base, "--size", "64", "--format", "VDI").CombinedOutput(); err != nil {
			t.Fatalf("failed to create base image: %v\n%s", err, out)
		}
	}
	t.Cleanup(func() {
		_ = exec.Command("VBoxManage", "closemedium", "disk", base, "--delete").Run()
		_ = os.Remove(base)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 makeBase,
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "virtualbox_vm" "test" {
  name       = "%s"
  os_type    = "Linux_64"
  memory     = 128
  disk_image = "%s"
}
`, vmName, base),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vmName),
					resource.TestCheckResourceAttr(resourceName, "disk_image", base),
					resource.TestCheckResourceAttrSet(resourceName, "disk_path"),
				),
			},
		},
	})
}
