package main

import (
	"context"
	"fmt"

	"github.com/bryanjbelanger/terraform-provider-virtualbox/datasources"
	"github.com/bryanjbelanger/terraform-provider-virtualbox/resources"
	"github.com/bryanjbelanger/terraform-provider-virtualbox/virtualbox"

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
		Description: "Interact with a local Oracle VirtualBox installation to manage " +
			"virtual machines, host-only networks, and shared folders. The provider drives the `VBoxManage` " +
			"command-line tool, so VirtualBox must be installed on the same host that runs Terraform.",
		Attributes: map[string]schema.Attribute{
			"vboxmanage_path": schema.StringAttribute{
				Optional: true,
				Description: "Path to the `VBoxManage` executable used to drive VirtualBox. When omitted, the " +
					"provider looks up `VBoxManage` on the system `PATH`. Set this when VBoxManage is installed in a " +
					"non-standard location, for example `/usr/local/bin/VBoxManage` on macOS or " +
					"`C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe` on Windows.",
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

	// Fail fast with an actionable message if VBoxManage cannot be executed,
	// rather than surfacing a cryptic error on the first resource operation.
	if _, err := client.RunContext(ctx, "--version"); err != nil {
		resp.Diagnostics.AddError(
			"VBoxManage Not Available",
			fmt.Sprintf("Could not execute VBoxManage at %q. Ensure VirtualBox is installed and that "+
				"VBoxManage is on your PATH, or set the provider's vboxmanage_path attribute.\n\nError: %s",
				vboxPath, err),
		)
		return
	}

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
