package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/bryanjbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	Name            types.String `tfsdk:"name"`
	NetworkCIDR     types.String `tfsdk:"network_cidr"`
	DHCP            types.Bool   `tfsdk:"dhcp"`
	DHCPLowerIP     types.String `tfsdk:"dhcp_lower_ip"`
	DHCPUpperIP     types.String `tfsdk:"dhcp_upper_ip"`
	GUID            types.String `tfsdk:"guid"`
	HostInterface   types.String `tfsdk:"host_interface"`
	AdapterType     types.String `tfsdk:"adapter_type"`
	AdapterNetwork  types.String `tfsdk:"adapter_network"`
	DHCPNetworkName types.String `tfsdk:"dhcp_network_name"`
}

// setComputedFrom copies the backend-derived computed attributes from a read
// or created network into the model.
func (m *networkResourceModel) setComputedFrom(network *virtualbox.Network) {
	m.GUID = types.StringValue(network.GUID)
	m.HostInterface = types.StringValue(network.HostInterface)
	m.AdapterType = types.StringValue(network.AdapterType())
	m.AdapterNetwork = types.StringValue(network.AdapterNetwork())
	m.DHCPNetworkName = types.StringValue(network.DHCPNetworkName())
}

// Metadata returns the resource type name.
func (r *networkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Schema defines the schema for the resource.
func (r *networkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Oracle VirtualBox host-only network — an isolated " +
			"network segment that connects the host to its guest virtual machines without exposing them to the " +
			"wider network. VirtualBox has two mutually exclusive host-only mechanisms and this resource uses " +
			"whichever the host OS supports: host-only *networks* (`VBoxManage hostonlynet`) on macOS and " +
			"Solaris, and legacy host-only *interfaces* (`VBoxManage hostonlyif`, `vboxnet0`-style) on Linux " +
			"and Windows, where the `hostonlynet` subcommand does not exist. Attach VMs portably via the " +
			"computed `adapter_type` and `adapter_network` attributes.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the host-only network. Must be unique within the VirtualBox installation. " +
					"Changing this forces a new network to be created. On the Linux/Windows interface backend " +
					"VirtualBox auto-assigns the actual interface name (`vboxnet0`, ...) — see `host_interface` — " +
					"and this name only identifies the resource in state.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_cidr": schema.StringAttribute{
				Description: "Network range in CIDR notation, for example `192.168.56.0/24`. VirtualBox requires " +
					"this to derive the network mask applied to the host-only adapter.",
				Required: true,
				Validators: []validator.String{
					validCIDR(),
				},
			},
			"dhcp": schema.BoolAttribute{
				Description: "Whether the host-only network is enabled. On macOS/Solaris VirtualBox maps the " +
					"network's enabled state to DHCP availability on the segment. On the Linux/Windows interface " +
					"backend DHCP is served by a separate `VBoxManage dhcpserver` bound to `dhcp_network_name`, " +
					"so this attribute has no effect there. Defaults to `true`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"dhcp_lower_ip": schema.StringAttribute{
				Description: "Lower bound of the DHCP address pool. The host itself takes this address on every " +
					"backend: hostonlynet hands the host the network's lower bound, and the interface backend " +
					"assigns it to the host-only interface explicitly. When omitted, defaults to the first usable " +
					"address in `network_cidr` (for `192.168.56.0/24`, that is `192.168.56.1`). Must be a valid " +
					"IPv4 address.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validIP(),
				},
			},
			"dhcp_upper_ip": schema.StringAttribute{
				Description: "Upper bound of the DHCP address pool. When omitted, defaults to the last usable " +
					"address in `network_cidr` (for `192.168.56.0/24`, that is `192.168.56.254`). Must be a valid " +
					"IPv4 address. On the Linux/Windows interface backend this is recorded but only takes effect " +
					"through a `VBoxManage dhcpserver` bound to `dhcp_network_name`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validIP(),
				},
			},
			"guid": schema.StringAttribute{
				Description: "Globally unique identifier assigned to the network by VirtualBox.",
				Computed:    true,
			},
			"host_interface": schema.StringAttribute{
				Description: "Name of the backing host-only interface (`vboxnet0`, ...) on the Linux/Windows " +
					"interface backend. Empty on macOS/Solaris, where the network is not interface-backed.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"adapter_type": schema.StringAttribute{
				Description: "Attachment type VMs must use to join this network — feed it to " +
					"`network_adapter.type`. `hostonlynet` on macOS/Solaris, `hostonly` on Linux/Windows.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"adapter_network": schema.StringAttribute{
				Description: "Network identifier VMs must use to join this network — feed it to " +
					"`network_adapter.network_name`. The network's `name` on macOS/Solaris, the auto-assigned " +
					"interface name on Linux/Windows.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dhcp_network_name": schema.StringAttribute{
				Description: "Internal network name a `VBoxManage dhcpserver --network` must bind to in order " +
					"to serve this segment: `hostonly-<name>` on macOS/Solaris, " +
					"`HostInterfaceNetworking-<interface>` on Linux/Windows.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	lower, upper, err := r.dhcpRange(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid DHCP range", err.Error())
		return
	}

	params := virtualbox.CreateNetworkParams{
		Name:        plan.Name.ValueString(),
		NetworkCIDR: plan.NetworkCIDR.ValueString(),
		DHCP:        plan.DHCP.ValueBool(),
		LowerIP:     lower,
		UpperIP:     upper,
	}

	network, err := r.client.CreateNetwork(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	plan.setComputedFrom(network)
	plan.DHCP = types.BoolValue(network.DHCP)
	plan.DHCPLowerIP = types.StringValue(network.LowerIP)
	plan.DHCPUpperIP = types.StringValue(network.UpperIP)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// dhcpRange returns the DHCP lower/upper IPs for a plan, deriving sensible
// defaults from network_cidr when the user did not specify them (VBoxManage
// requires a range at network-creation time).
func (r *networkResource) dhcpRange(plan networkResourceModel) (lower string, upper string, err error) {
	lower = plan.DHCPLowerIP.ValueString()
	upper = plan.DHCPUpperIP.ValueString()
	if lower != "" && upper != "" {
		return lower, upper, nil
	}

	dl, du, err := virtualbox.DeriveDHCPRange(plan.NetworkCIDR.ValueString())
	if err != nil {
		return "", "", err
	}
	if lower == "" {
		lower = dl
	}
	if upper == "" {
		upper = du
	}
	return lower, upper, nil
}

// Read refreshes the Terraform state with the latest data.
func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.ReadNetwork(ctx, state.Name.ValueString(), state.HostInterface.ValueString())
	if err != nil {
		if errors.Is(err, virtualbox.ErrNetworkNotFound) {
			// The network was removed outside of Terraform; drop it from state so
			// the next plan recreates it.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	state.setComputedFrom(network)
	if network.Backend == virtualbox.BackendHostOnlyNet {
		// On the interface backend `dhcp` has no VirtualBox-side representation
		// (a separate dhcpserver provides it), so the state value stands.
		state.DHCP = types.BoolValue(network.DHCP)
	}
	if network.LowerIP != "" {
		state.DHCPLowerIP = types.StringValue(network.LowerIP)
	}
	if network.UpperIP != "" {
		state.DHCPUpperIP = types.StringValue(network.UpperIP)
	}
	// network_cidr is preserved from state: VBoxManage reports a dotted netmask
	// rather than a CIDR, so reconstructing it here could cause spurious diffs.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lower, upper, err := r.dhcpRange(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid DHCP range", err.Error())
		return
	}

	dhcp := plan.DHCP.ValueBool()
	params := virtualbox.UpdateNetworkParams{
		Name:          plan.Name.ValueString(),
		NetworkCIDR:   plan.NetworkCIDR.ValueString(),
		DHCP:          &dhcp,
		LowerIP:       lower,
		UpperIP:       upper,
		HostInterface: plan.HostInterface.ValueString(),
	}

	network, err := r.client.UpdateNetwork(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	plan.setComputedFrom(network)
	plan.DHCP = types.BoolValue(network.DHCP)
	plan.DHCPLowerIP = types.StringValue(network.LowerIP)
	plan.DHCPUpperIP = types.StringValue(network.UpperIP)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNetwork(ctx, state.Name.ValueString(), state.HostInterface.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting network", err.Error())
		return
	}
}

// ImportState imports a resource state.
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
