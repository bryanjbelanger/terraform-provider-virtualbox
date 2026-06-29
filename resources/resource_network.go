package resources

import (
	"context"
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &networkResource{}
var _ resource.ResourceWithConfigure = &networkResource{}
var _ resource.ResourceWithImportState = &networkResource{}

// NewNetworkResource is a helper function to simplify the provider implementation.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

// networkResource is the resource implementation.
type networkResource struct {
	client *virtualbox.Client
}

// networkResourceModel maps the resource schema data to a Go type.
type networkResourceModel struct {
	Name        types.String `tfsdk:"name"`
	NetworkCIDR types.String `tfsdk:"network_cidr"`
	DHCP        types.Bool   `tfsdk:"dhcp"`
	DHCPLowerIP types.String `tfsdk:"dhcp_lower_ip"`
	DHCPUpperIP types.String `tfsdk:"dhcp_upper_ip"`
	GUID        types.String `tfsdk:"guid"`
}

// Metadata returns the resource type name.
func (r *networkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Schema defines the schema for the resource.
func (r *networkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VirtualBox Host-Only network.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the host-only network.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_cidr": schema.StringAttribute{
				Description: "The network CIDR (e.g., '192.168.56.0/24').",
				Optional:    true,
			},
			"dhcp": schema.BoolAttribute{
				Description: "Enable DHCP on the network.",
				Optional:    true,
			},
			"dhcp_lower_ip": schema.StringAttribute{
				Description: "Lower bound of the DHCP IP range.",
				Optional:    true,
			},
			"dhcp_upper_ip": schema.StringAttribute{
				Description: "Upper bound of the DHCP IP range.",
				Optional:    true,
			},
			"guid": schema.StringAttribute{
				Description: "The GUID of the network.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *networkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*virtualbox.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *virtualbox.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dhcp := true
	if !plan.DHCP.IsNull() && !plan.DHCP.IsUnknown() {
		dhcp = plan.DHCP.ValueBool()
	}

	params := virtualbox.CreateNetworkParams{
		Name:        plan.Name.ValueString(),
		NetworkCIDR: plan.NetworkCIDR.ValueString(),
		DHCP:        dhcp,
		LowerIP:     plan.DHCPLowerIP.ValueString(),
		UpperIP:     plan.DHCPUpperIP.ValueString(),
	}

	network, err := r.client.CreateNetwork(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	// Update state
	plan.GUID = types.StringValue(network.GUID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.ReadNetwork(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	state.GUID = types.StringValue(network.GUID)
	state.DHCP = types.BoolValue(network.DHCP)
	state.NetworkCIDR = types.StringValue(network.NetworkCIDR)
	state.DHCPLowerIP = types.StringValue(network.LowerIP)
	state.DHCPUpperIP = types.StringValue(network.UpperIP)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dhcp := true
	if !plan.DHCP.IsNull() && !plan.DHCP.IsUnknown() {
		dhcp = plan.DHCP.ValueBool()
	}

	params := virtualbox.UpdateNetworkParams{
		Name:        plan.Name.ValueString(),
		NetworkCIDR: plan.NetworkCIDR.ValueString(),
		DHCP:        &dhcp,
		LowerIP:     plan.DHCPLowerIP.ValueString(),
		UpperIP:     plan.DHCPUpperIP.ValueString(),
	}

	_, err := r.client.UpdateNetwork(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNetwork(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting network", err.Error())
		return
	}
}

// ImportState imports a resource state.
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
