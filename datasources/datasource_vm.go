package datasources

import (
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DataSourceVM returns the schema for the virtualbox_vm data source.
func DataSourceVM() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVMRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the virtual machine.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The guest OS type.",
			},
			"memory": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Amount of RAM in MB.",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of CPU cores.",
			},
			"vram": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Video RAM in MB.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the VM.",
			},
			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the virtual machine.",
			},
		},
	}
}

func dataSourceVMRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("name").(string)
	vm, err := client.ReadVM(vmName)
	if err != nil {
		return fmt.Errorf("error reading VM data source: %w", err)
	}

	d.SetId(vm.UUID)
	d.Set("name", vm.Name)
	d.Set("os_type", vm.OSType)
	d.Set("memory", vm.MemoryMB)
	d.Set("cpus", vm.CPUs)
	d.Set("vram", vm.VRAM)
	d.Set("status", vm.Status)
	d.Set("uuid", vm.UUID)

	return nil
}
