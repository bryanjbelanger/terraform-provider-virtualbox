package main

import (
	"context"

	"github.com/bryanbelanger/terraform-provider-virtualbox/datasources"
	"github.com/bryanbelanger/terraform-provider-virtualbox/resources"
	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns the VirtualBox Terraform provider schema.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"vboxmanage_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "VBoxManage",
				Description: "Path to the VBoxManage executable. Defaults to 'VBoxManage' found in PATH.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"virtualbox_vm":            resources.ResourceVM(),
			"virtualbox_network":       resources.ResourceNetwork(),
			"virtualbox_shared_folder": resources.ResourceSharedFolder(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"virtualbox_vm": datasources.DataSourceVM(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	vboxmanagePath := d.Get("vboxmanage_path").(string)
	return virtualbox.NewClient(vboxmanagePath), nil
}
