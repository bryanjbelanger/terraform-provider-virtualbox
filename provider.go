package main

import (
	"context"

	"github.com/bryanbelanger/terraform-provider-virtualbox/datasources"
	"github.com/bryanbelanger/terraform-provider-virtualbox/resources"
	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &virtualboxProvider{}

// virtualboxProvider defines the provider implementation.
type virtualboxProvider struct {
	version string
}

// virtualboxProviderModel maps provider schema data to a Go type.
type virtualboxProviderModel struct {
	VBoxManagePath types.String `tfsdk:"vboxmanage_path"`
}

// New returns a new virtualbox provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &virtualboxProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *virtualboxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "virtualbox"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *virtualboxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "VirtualBox Terraform provider.",
		Attributes: map[string]schema.Attribute{
			"vboxmanage_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to the VBoxManage executable. Defaults to 'VBoxManage' found in PATH.",
			},
		},
	}
}

// Configure prepares a VirtualBox API client for data sources and resources.
func (p *virtualboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data virtualboxProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vboxPath := "VBoxManage"
	if !data.VBoxManagePath.IsNull() && !data.VBoxManagePath.IsUnknown() {
		vboxPath = data.VBoxManagePath.ValueString()
	}

	client := virtualbox.NewClient(vboxPath)
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *virtualboxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewVMResource,
		resources.NewNetworkResource,
		resources.NewSharedFolderResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *virtualboxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewVMDataSource,
	}
}
