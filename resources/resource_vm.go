package resources

import (
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceVM returns the schema for the virtualbox_vm resource.
func ResourceVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceVMCreate,
		Read:   resourceVMRead,
		Update: resourceVMUpdate,
		Delete: resourceVMDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the virtual machine.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Other",
				Description: "The guest OS type (e.g., 'Ubuntu_64', 'Windows10_64', 'Other').",
			},
			"memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1024,
				Description: "Amount of RAM in MB.",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Number of CPU cores.",
			},
			"vram": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     8,
				Description: "Video RAM in MB.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the VM (e.g., 'running', 'poweroff').",
			},
			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the virtual machine.",
			},
		},
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	params := virtualbox.CreateVMParams{
		Name:   d.Get("name").(string),
		OSType: d.Get("os_type").(string),
		Memory: d.Get("memory").(int),
		CPUs:   d.Get("cpus").(int),
		VRAM:   d.Get("vram").(int),
	}

	vm, err := client.CreateVM(params)
	if err != nil {
		return fmt.Errorf("error creating VM: %w", err)
	}

	d.SetId(vm.Name)

	return resourceVMRead(d, meta)
}

func resourceVMRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	vm, err := client.ReadVM(d.Id())
	if err != nil {
		return fmt.Errorf("error reading VM: %w", err)
	}

	d.Set("name", vm.Name)
	d.Set("os_type", vm.OSType)
	d.Set("memory", vm.MemoryMB)
	d.Set("cpus", vm.CPUs)
	d.Set("vram", vm.VRAM)
	d.Set("status", vm.Status)
	d.Set("uuid", vm.UUID)

	return nil
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	params := virtualbox.UpdateVMParams{
		Name:   d.Id(),
		OSType: d.Get("os_type").(string),
		Memory: d.Get("memory").(int),
		CPUs:   d.Get("cpus").(int),
		VRAM:   d.Get("vram").(int),
	}

	_, err := client.UpdateVM(params)
	if err != nil {
		return fmt.Errorf("error updating VM: %w", err)
	}

	return resourceVMRead(d, meta)
}

func resourceVMDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	err := client.DeleteVM(d.Id(), true)
	if err != nil {
		return fmt.Errorf("error deleting VM: %w", err)
	}

	d.SetId("")
	return nil
}
