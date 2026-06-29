package resources

import (
	"context"
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceNetwork returns the schema for the virtualbox_network resource.
func ResourceNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the host-only network.",
			},
			"network_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The network CIDR (e.g., '192.168.56.0/24').",
			},
			"dhcp": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable DHCP on the network.",
			},
			"dhcp_lower_ip": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Lower bound of the DHCP IP range.",
			},
			"dhcp_upper_ip": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Upper bound of the DHCP IP range.",
			},
			"guid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The GUID of the network.",
			},
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	params := virtualbox.CreateNetworkParams{
		Name:        d.Get("name").(string),
		NetworkCIDR: d.Get("network_cidr").(string),
		DHCP:        d.Get("dhcp").(bool),
		LowerIP:     d.Get("dhcp_lower_ip").(string),
		UpperIP:     d.Get("dhcp_upper_ip").(string),
	}

	network, err := client.CreateNetwork(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating network: %w", err))
	}

	d.SetId(network.Name)

	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	network, err := client.ReadNetwork(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading network: %w", err))
	}

	d.Set("name", network.Name)
	d.Set("guid", network.GUID)
	d.Set("dhcp", network.DHCP)
	d.Set("network_cidr", network.NetworkCIDR)
	d.Set("dhcp_lower_ip", network.LowerIP)
	d.Set("dhcp_upper_ip", network.UpperIP)

	return nil
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	dhcp := d.Get("dhcp").(bool)
	params := virtualbox.UpdateNetworkParams{
		Name:        d.Id(),
		NetworkCIDR: d.Get("network_cidr").(string),
		DHCP:        &dhcp,
		LowerIP:     d.Get("dhcp_lower_ip").(string),
		UpperIP:     d.Get("dhcp_upper_ip").(string),
	}

	_, err := client.UpdateNetwork(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating network: %w", err))
	}

	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	err := client.DeleteNetwork(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting network: %w", err))
	}

	d.SetId("")
	return nil
}
