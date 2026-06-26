package resources

import (
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceSharedFolder returns the schema for the virtualbox_shared_folder resource.
func ResourceSharedFolder() *schema.Resource {
	return &schema.Resource{
		Create: resourceSharedFolderCreate,
		Read:   resourceSharedFolderRead,
		Update: resourceSharedFolderUpdate,
		Delete: resourceSharedFolderDelete,

		Schema: map[string]*schema.Schema{
			"vm_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the virtual machine.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the shared folder.",
			},
			"host_path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path on the host system to share.",
			},
			"writable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the shared folder is writable.",
			},
			"automount": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the shared folder is automounted in the guest.",
			},
		},
	}
}

func resourceSharedFolderCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	params := virtualbox.CreateSharedFolderParams{
		VMName:    d.Get("vm_name").(string),
		Name:      d.Get("name").(string),
		HostPath:  d.Get("host_path").(string),
		Writable:  d.Get("writable").(bool),
		AutoMount: d.Get("automount").(bool),
	}

	_, err := client.CreateSharedFolder(params)
	if err != nil {
		return fmt.Errorf("error creating shared folder: %w", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", params.VMName, params.Name))

	return resourceSharedFolderRead(d, meta)
}

func resourceSharedFolderRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("vm_name").(string)
	folderName := d.Get("name").(string)

	folder, err := client.ReadSharedFolder(vmName, folderName)
	if err != nil {
		return fmt.Errorf("error reading shared folder: %w", err)
	}

	d.Set("name", folder.Name)
	d.Set("host_path", folder.HostPath)
	d.Set("writable", folder.Writable)
	d.Set("automount", folder.AutoMount)

	return nil
}

func resourceSharedFolderUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	params := virtualbox.UpdateSharedFolderParams{
		VMName:    d.Get("vm_name").(string),
		Name:      d.Get("name").(string),
		HostPath:  d.Get("host_path").(string),
		Writable:  d.Get("writable").(bool),
		AutoMount: d.Get("automount").(bool),
	}

	_, err := client.UpdateSharedFolder(params)
	if err != nil {
		return fmt.Errorf("error updating shared folder: %w", err)
	}

	return resourceSharedFolderRead(d, meta)
}

func resourceSharedFolderDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("vm_name").(string)
	folderName := d.Get("name").(string)

	err := client.DeleteSharedFolder(vmName, folderName)
	if err != nil {
		return fmt.Errorf("error deleting shared folder: %w", err)
	}

	d.SetId("")
	return nil
}
