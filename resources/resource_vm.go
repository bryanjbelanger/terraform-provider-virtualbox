package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &vmResource{}
var _ resource.ResourceWithConfigure = &vmResource{}
var _ resource.ResourceWithImportState = &vmResource{}

// NewVMResource is a helper function to simplify the provider implementation.
func NewVMResource() resource.Resource {
	return &vmResource{}
}

// vmResource is the resource implementation.
type vmResource struct {
	client *virtualbox.Client
}

// vmResourceModel maps the resource schema data to a Go type.
type vmResourceModel struct {
	Name              types.String `tfsdk:"name"`
	OSType            types.String `tfsdk:"os_type"`
	Memory            types.Int64  `tfsdk:"memory"`
	CPUs              types.Int64  `tfsdk:"cpus"`
	VRAM              types.Int64  `tfsdk:"vram"`
	Status            types.String `tfsdk:"status"`
	UUID              types.String `tfsdk:"uuid"`
	ISOPath           types.String `tfsdk:"iso_path"`
	ISOController     types.String `tfsdk:"iso_controller"`
	DiskPath          types.String `tfsdk:"disk_path"`
	DiskSizeMB        types.Int64  `tfsdk:"disk_size_mb"`
	StorageController types.String `tfsdk:"storage_controller"`
	StartOnCreate     types.Bool   `tfsdk:"start_on_create"`
}

// Metadata returns the resource type name.
func (r *vmResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the resource.
func (r *vmResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VirtualBox virtual machine.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the virtual machine.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_type": schema.StringAttribute{
				Description: "The guest OS type (e.g., 'Ubuntu_64', 'Windows10_64', 'Other').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Other"),
			},
			"memory": schema.Int64Attribute{
				Description: "Amount of RAM in MB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1024),
			},
			"cpus": schema.Int64Attribute{
				Description: "Number of CPU cores.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"vram": schema.Int64Attribute{
				Description: "Video RAM in MB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(8),
			},
			"status": schema.StringAttribute{
				Description: "The current status of the VM (e.g., 'running', 'poweroff').",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "The UUID of the virtual machine.",
				Computed:    true,
			},
			"iso_path": schema.StringAttribute{
				Description: "Path to an ISO image to attach as a DVD drive for installation.",
				Optional:    true,
			},
			"iso_controller": schema.StringAttribute{
				Description: "Storage controller name to attach the ISO to. Default: 'IDE'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("IDE"),
			},
			"disk_path": schema.StringAttribute{
				Description: "Path to the virtual disk (VDI). Defaults to ~/VirtualBox VMs/<name>/<name>.vdi.",
				Optional:    true,
				Computed:    true,
			},
			"disk_size_mb": schema.Int64Attribute{
				Description: "Size of the virtual disk in MB. Set to 0 to skip disk creation.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"storage_controller": schema.StringAttribute{
				Description: "Name for the SATA storage controller. Default: 'SATA'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SATA"),
			},
			"start_on_create": schema.BoolAttribute{
				Description: "Start the VM after creation.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *vmResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// diskPathForVM returns the default disk path for a VM.
func diskPathForVM(vmName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "VirtualBox VMs", vmName, vmName+".vdi")
}

// Create creates the resource and sets the initial Terraform state.
func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName := plan.Name.ValueString()

	params := virtualbox.CreateVMParams{
		Name:   vmName,
		OSType: plan.OSType.ValueString(),
		Memory: int(plan.Memory.ValueInt64()),
		CPUs:   int(plan.CPUs.ValueInt64()),
		VRAM:   int(plan.VRAM.ValueInt64()),
	}

	vm, err := r.client.CreateVM(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating VM", err.Error())
		return
	}

	ctrlName := plan.StorageController.ValueString()
	if err := r.client.AddStorageController(ctx, vmName, ctrlName, virtualbox.ControllerSATA); err != nil {
		resp.Diagnostics.AddError("Error adding SATA controller", err.Error())
		return
	}

	isoCtrlName := plan.ISOController.ValueString()
	if isoCtrlName != "" && isoCtrlName != ctrlName {
		if err := r.client.AddStorageController(ctx, vmName, isoCtrlName, virtualbox.ControllerIDE); err != nil {
			resp.Diagnostics.AddError("Error adding IDE controller", err.Error())
			return
		}
	}

	diskSize := int(plan.DiskSizeMB.ValueInt64())
	if diskSize > 0 {
		diskPath := plan.DiskPath.ValueString()
		if diskPath == "" {
			diskPath = diskPathForVM(vmName)
			plan.DiskPath = types.StringValue(diskPath)
		}
		diskDir := filepath.Dir(diskPath)
		if err := os.MkdirAll(diskDir, 0755); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error creating disk directory %s", diskDir), err.Error())
			return
		}
		if err := r.client.CreateDisk(ctx, diskPath, diskSize); err != nil {
			resp.Diagnostics.AddError("Error creating disk", err.Error())
			return
		}
		if err := r.client.AttachDisk(ctx, vmName, ctrlName, diskPath, 0); err != nil {
			resp.Diagnostics.AddError("Error attaching disk", err.Error())
			return
		}
	}

	isoPath := plan.ISOPath.ValueString()
	if isoPath != "" {
		attachCtrlName := isoCtrlName
		if attachCtrlName == "" {
			attachCtrlName = ctrlName
		}
		if err := r.client.AttachISO(ctx, vmName, attachCtrlName, isoPath); err != nil {
			resp.Diagnostics.AddError("Error attaching ISO", err.Error())
			return
		}
	}

	if isoPath != "" {
		r.client.RunContext(ctx, "modifyvm", vmName, "--boot1", "dvd", "--boot2", "disk")
	} else {
		r.client.RunContext(ctx, "modifyvm", vmName, "--boot1", "disk")
	}

	startOnCreate := plan.StartOnCreate.ValueBool()
	if startOnCreate {
		if err := r.client.StartVM(ctx, vmName); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error starting VM %s", vmName), err.Error())
			return
		}
	}

	plan.UUID = types.StringValue(vm.UUID)
	plan.Status = types.StringValue(vm.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.ReadVM(ctx, state.Name.ValueString())
	if err != nil {
		if errors.Is(err, virtualbox.ErrVMNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName := state.Name.ValueString()

	vm, err := r.client.ReadVM(ctx, vmName)
	if err != nil {
		resp.Diagnostics.AddError("Error reading VM before update", err.Error())
		return
	}

	wasRunning := (vm.Status == "running")
	if wasRunning {
		err = r.client.StopVM(ctx, vmName)
		if err != nil {
			resp.Diagnostics.AddError("Error stopping VM for hardware update", err.Error())
			return
		}
	}

	params := virtualbox.UpdateVMParams{
		Name:   vmName,
		OSType: plan.OSType.ValueString(),
		Memory: int(plan.Memory.ValueInt64()),
		CPUs:   int(plan.CPUs.ValueInt64()),
		VRAM:   int(plan.VRAM.ValueInt64()),
	}

	_, err = r.client.UpdateVM(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating VM", err.Error())
		return
	}

	if wasRunning {
		err = r.client.StartVM(ctx, vmName)
		if err != nil {
			resp.Diagnostics.AddError("Error restarting VM after hardware update", err.Error())
			return
		}
	}

	// Update status from Read
	vmUpdated, _ := r.client.ReadVM(ctx, vmName)
	if vmUpdated != nil {
		plan.Status = types.StringValue(vmUpdated.Status)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmName := state.Name.ValueString()
	diskPath := state.DiskPath.ValueString()

	vm, err := r.client.ReadVM(ctx, vmName)
	if err == nil && vm.Status == "running" {
		r.client.StopVM(ctx, vmName)
	}

	err = r.client.DeleteVM(ctx, vmName, true)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting VM", err.Error())
		return
	}

	if diskPath != "" && !strings.Contains(diskPath, vmName) {
		os.Remove(diskPath)
	}
}

// ImportState imports a resource state.
func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
