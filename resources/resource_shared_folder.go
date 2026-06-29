package resources

import (
	"context"
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceSharedFolder returns the schema for the virtualbox_shared_folder resource.
func ResourceSharedFolder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSharedFolderCreate,
		ReadContext:   resourceSharedFolderRead,
		UpdateContext: resourceSharedFolderUpdate,
		DeleteContext: resourceSharedFolderDelete,

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

func resourceSharedFolderCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	params := virtualbox.CreateSharedFolderParams{
		VMName:    d.Get("vm_name").(string),
		Name:      d.Get("name").(string),
		HostPath:  d.Get("host_path").(string),
		Writable:  d.Get("writable").(bool),
		AutoMount: d.Get("automount").(bool),
	}

	_, err := client.CreateSharedFolder(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating shared folder: %w", err))
	}

	d.SetId(fmt.Sprintf("%s/%s", params.VMName, params.Name))

	return resourceSharedFolderRead(ctx, d, meta)
}

func resourceSharedFolderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("vm_name").(string)
	folderName := d.Get("name").(string)

	folder, err := client.ReadSharedFolder(ctx, vmName, folderName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading shared folder: %w", err))
	}

	d.Set("name", folder.Name)
	d.Set("host_path", folder.HostPath)
	d.Set("writable", folder.Writable)
	d.Set("automount", folder.AutoMount)

	return nil
}

func resourceSharedFolderUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	params := virtualbox.UpdateSharedFolderParams{
		VMName:    d.Get("vm_name").(string),
		Name:      d.Get("name").(string),
		HostPath:  d.Get("host_path").(string),
		Writable:  d.Get("writable").(bool),
		AutoMount: d.Get("automount").(bool),
	}

	_, err := client.UpdateSharedFolder(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating shared folder: %w", err))
	}

	return resourceSharedFolderRead(ctx, d, meta)
}

func resourceSharedFolderDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("vm_name").(string)
	folderName := d.Get("name").(string)

	err := client.DeleteSharedFolder(ctx, vmName, folderName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting shared folder: %w", err))
	}

	d.SetId("")
	return nil
}
