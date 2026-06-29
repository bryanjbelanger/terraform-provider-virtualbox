package datasources

import (
	"context"
	"fmt"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &vmDataSource{}
var _ datasource.DataSourceWithConfigure = &vmDataSource{}

// NewVMDataSource is a helper function to simplify the provider implementation.
func NewVMDataSource() datasource.DataSource {
	return &vmDataSource{}
}

// vmDataSource is the data source implementation.
type vmDataSource struct {
	client *virtualbox.Client
}

// vmDataSourceModel maps the data source schema data to a Go type.
type vmDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	OSType types.String `tfsdk:"os_type"`
	Memory types.Int64  `tfsdk:"memory"`
	CPUs   types.Int64  `tfsdk:"cpus"`
	VRAM   types.Int64  `tfsdk:"vram"`
	Status types.String `tfsdk:"status"`
	UUID   types.String `tfsdk:"uuid"`
}

// Metadata returns the data source type name.
func (d *vmDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the data source.
func (d *vmDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a VirtualBox virtual machine.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the virtual machine.",
				Required:    true,
			},
			"os_type": schema.StringAttribute{
				Description: "The guest OS type.",
				Computed:    true,
			},
			"memory": schema.Int64Attribute{
				Description: "Amount of RAM in MB.",
				Computed:    true,
			},
			"cpus": schema.Int64Attribute{
				Description: "Number of CPU cores.",
				Computed:    true,
			},
			"vram": schema.Int64Attribute{
				Description: "Video RAM in MB.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the VM.",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "The UUID of the virtual machine.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *vmDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*virtualbox.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *virtualbox.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *vmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vmDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := d.client.ReadVM(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading VM", err.Error())
		return
	}

	state.UUID = types.StringValue(vm.UUID)
	state.OSType = types.StringValue(vm.OSType)
	state.Memory = types.Int64Value(int64(vm.MemoryMB))
	state.CPUs = types.Int64Value(int64(vm.CPUs))
	state.VRAM = types.Int64Value(int64(vm.VRAM))
	state.Status = types.StringValue(vm.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
