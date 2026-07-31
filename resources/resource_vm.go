package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bryanjbelanger/terraform-provider-virtualbox/virtualbox"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &vmResource{}
var _ resource.ResourceWithConfigure = &vmResource{}
var _ resource.ResourceWithImportState = &vmResource{}
var _ resource.ResourceWithValidateConfig = &vmResource{}

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
	Name              types.String          `tfsdk:"name"`
	OSType            types.String          `tfsdk:"os_type"`
	Memory            types.Int64           `tfsdk:"memory"`
	CPUs              types.Int64           `tfsdk:"cpus"`
	VRAM              types.Int64           `tfsdk:"vram"`
	Status            types.String          `tfsdk:"status"`
	UUID              types.String          `tfsdk:"uuid"`
	ISOPath           types.String          `tfsdk:"iso_path"`
	ISOController     types.String          `tfsdk:"iso_controller"`
	DiskPath          types.String          `tfsdk:"disk_path"`
	DiskSizeMB        types.Int64           `tfsdk:"disk_size_mb"`
	DiskImage         types.String          `tfsdk:"disk_image"`
	StorageController types.String          `tfsdk:"storage_controller"`
	StartOnCreate     types.Bool            `tfsdk:"start_on_create"`
	NetworkAdapters   []networkAdapterModel `tfsdk:"network_adapter"`
}

// networkAdapterModel maps a single network_adapter block to a Go type.
type networkAdapterModel struct {
	Type           types.String `tfsdk:"type"`
	NetworkName    types.String `tfsdk:"network_name"`
	NICType        types.String `tfsdk:"nic_type"`
	CableConnected types.Bool   `tfsdk:"cable_connected"`
	MACAddress     types.String `tfsdk:"mac_address"`
}

// Metadata returns the resource type name.
func (r *vmResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the resource.
func (r *vmResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Oracle VirtualBox virtual machine, including its CPU, " +
			"memory, and video-memory allocation, an optional virtual disk, and an optional installation ISO. " +
			"The virtual machine is created with `VBoxManage createvm` and configured with `VBoxManage modifyvm`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the virtual machine. Must be unique within the VirtualBox installation. " +
					"Changing this forces a new virtual machine to be created.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_type": schema.StringAttribute{
				Description: "VirtualBox guest OS type identifier used to apply sensible hardware defaults, for " +
					"example `Ubuntu_64`, `Windows10_64`, `RedHat_64`, or `Other`. Run `VBoxManage list ostypes` to " +
					"see every supported value. Note that VirtualBox may report a normalized form of this value (for " +
					"example `Other` is reported as `Other/Unknown`); the configured value is preserved to avoid a " +
					"perpetual diff. Defaults to `Other`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Other"),
			},
			"memory": schema.Int64Attribute{
				Description: "Amount of RAM, in megabytes, allocated to the virtual machine. Must be at least `4`. " +
					"Defaults to `1024`.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1024),
				Validators: []validator.Int64{
					int64validator.AtLeast(4),
				},
			},
			"cpus": schema.Int64Attribute{
				Description: "Number of virtual CPU cores assigned to the virtual machine. Must be at least `1`. " +
					"Defaults to `1`.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"vram": schema.Int64Attribute{
				Description: "Amount of video memory, in megabytes, allocated to the virtual graphics adapter. " +
					"Must be between `1` and `256`. Defaults to `8`.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(8),
				Validators: []validator.Int64{
					int64validator.Between(1, 256),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current power state of the virtual machine as reported by VirtualBox, for example " +
					"`running` or `poweroff`.",
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Description: "Universally unique identifier assigned to the virtual machine by VirtualBox.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"iso_path": schema.StringAttribute{
				Description: "Absolute path to an ISO image to attach as a DVD drive, typically an operating-system " +
					"installer. When set, the boot order is configured to boot from DVD first, then disk. Changing " +
					"this forces a new virtual machine to be created.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"iso_controller": schema.StringAttribute{
				Description: "Name of the storage controller the ISO is attached to. An IDE controller with this " +
					"name is created if it does not already exist. Changing this forces a new virtual machine to be " +
					"created. Defaults to `IDE`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("IDE"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disk_path": schema.StringAttribute{
				Description: "Absolute path to the virtual disk image (VDI) file. When omitted, a disk is created " +
					"at `~/VirtualBox VMs/<name>/<name>.vdi`. Changing this forces a new virtual machine to be created.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disk_size_mb": schema.Int64Attribute{
				Description: "Size of the virtual disk, in megabytes. Set to `0` to skip disk creation entirely, " +
					"which is useful when booting from an ISO only or attaching an existing disk out of band. " +
					"Changing this forces a new virtual machine to be created. Defaults to `0`.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"disk_image": schema.StringAttribute{
				Description: "Absolute path to an existing disk image (for example a prebuilt Talos VDI/VMDK) to " +
					"use as the boot disk. The image is cloned to a per-VM disk with a fresh UUID and attached, so " +
					"the source image is never modified and the same image can back many VMs. Mutually exclusive " +
					"with `disk_size_mb`. Changing this forces a new virtual machine to be created.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"storage_controller": schema.StringAttribute{
				Description: "Name of the SATA storage controller created for the virtual disk. Changing this " +
					"forces a new virtual machine to be created. Defaults to `SATA`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("SATA"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"start_on_create": schema.BoolAttribute{
				Description: "Whether to power on the virtual machine in headless mode immediately after it is " +
					"created. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"network_adapter": schema.ListNestedBlock{
				Description: "Ordered list of network adapters attached to the virtual machine, mapping to " +
					"VirtualBox adapters `nic1`, `nic2`, and so on. When at least one block is present the provider " +
					"manages all eight adapters (unlisted ones are disabled); when omitted, the VM keeps " +
					"VirtualBox's default NAT adapter.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Attachment type: `nat` (outbound only), `hostonlynet` (host-only " +
								"network), `bridged`, `intnet` (internal network), `natnetwork`, or `null` " +
								"(present but disconnected).",
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("nat", "natnetwork", "hostonlynet", "bridged", "intnet", "null"),
							},
						},
						"network_name": schema.StringAttribute{
							Description: "Backing network, interpreted per `type`: the host-only network name for " +
								"`hostonlynet`, the host interface for `bridged`, the internal network name for " +
								"`intnet`, or the NAT network name for `natnetwork`. Ignored for `nat` and `null`.",
							Optional: true,
						},
						"nic_type": schema.StringAttribute{
							Description: "Emulated adapter hardware, for example `virtio` (recommended for Linux " +
								"guests such as Talos), `82540EM`, or `Am79C973`.",
							Optional: true,
							Computed: true,
						},
						"cable_connected": schema.BoolAttribute{
							Description: "Whether the virtual network cable is connected. Defaults to `true`.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"mac_address": schema.StringAttribute{
							Description: "MAC address for the adapter as 12 uppercase hex digits without separators " +
								"(for example `080027AB12CD`), matching how VirtualBox reports it. When omitted, " +
								"VirtualBox generates one. Setting it enables stable, pre-seeded MACs for e.g. Talos " +
								"`deviceSelector.hardwareAddr` matching or DHCP reservations.",
							Optional: true,
							Computed: true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[0-9A-F]{12}$`),
									"must be 12 uppercase hexadecimal digits with no separators (e.g. 080027AB12CD)",
								),
							},
						},
					},
				},
			},
		},
	}
}

// ValidateConfig enforces cross-attribute rules the schema cannot express alone.
func (r *vmResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg vmResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !cfg.DiskImage.IsNull() && !cfg.DiskImage.IsUnknown() && cfg.DiskImage.ValueString() != "" &&
		!cfg.DiskSizeMB.IsNull() && !cfg.DiskSizeMB.IsUnknown() && cfg.DiskSizeMB.ValueInt64() > 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("disk_image"),
			"Conflicting Disk Configuration",
			"disk_image and disk_size_mb are mutually exclusive: disk_image clones an existing image, while "+
				"disk_size_mb creates a new blank disk. Set only one.",
		)
	}

	for i, a := range cfg.NetworkAdapters {
		if a.Type.IsUnknown() || a.NetworkName.IsUnknown() {
			// network_name is often a reference resolved after plan; can't validate yet.
			continue
		}
		t := a.Type.ValueString()
		needsName := t == "hostonlynet" || t == "bridged" || t == "intnet" || t == "natnetwork"
		if needsName && (a.NetworkName.IsNull() || a.NetworkName.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(
				path.Root("network_adapter").AtListIndex(i).AtName("network_name"),
				"Missing network_name",
				fmt.Sprintf("network_adapter type %q requires network_name to be set.", t),
			)
		}
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

	// Resolve disk_path to a concrete value up front. It is a Computed attribute,
	// so it must be known after apply even when no disk is created (disk_size_mb=0).
	diskPath := plan.DiskPath.ValueString()
	if diskPath == "" {
		diskPath = diskPathForVM(vmName)
	}
	plan.DiskPath = types.StringValue(diskPath)

	diskImage := plan.DiskImage.ValueString()
	diskSize := int(plan.DiskSizeMB.ValueInt64())
	if diskImage != "" || diskSize > 0 {
		diskDir := filepath.Dir(diskPath)
		if err := os.MkdirAll(diskDir, 0755); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error creating disk directory %s", diskDir), err.Error())
			return
		}
		if diskImage != "" {
			// Clone the golden image to a per-VM disk with a fresh UUID; the source
			// image is left untouched so it can back many VMs.
			if err := r.client.CloneDisk(ctx, diskImage, diskPath, "VDI"); err != nil {
				resp.Diagnostics.AddError("Error cloning disk image", err.Error())
				return
			}
		} else {
			if err := r.client.CreateDisk(ctx, diskPath, diskSize); err != nil {
				resp.Diagnostics.AddError("Error creating disk", err.Error())
				return
			}
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
		_, _ = r.client.RunContext(ctx, "modifyvm", vmName, "--boot1", "dvd", "--boot2", "disk")
	} else {
		_, _ = r.client.RunContext(ctx, "modifyvm", vmName, "--boot1", "disk")
	}

	if len(plan.NetworkAdapters) > 0 {
		if err := r.client.ConfigureNetworkAdapters(ctx, vmName, adaptersFromModel(plan.NetworkAdapters)); err != nil {
			resp.Diagnostics.AddError("Error configuring network adapters", err.Error())
			return
		}
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

	// Refresh computed adapter fields (mac_address, nic_type, cable_connected)
	// from the running configuration so they are known after apply.
	if len(plan.NetworkAdapters) > 0 {
		if refreshed, err := r.client.ReadVM(ctx, vmName); err == nil {
			plan.NetworkAdapters = adaptersToModel(refreshed.Adapters)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// adaptersFromModel converts schema block models into client adapter structs.
func adaptersFromModel(models []networkAdapterModel) []virtualbox.NetworkAdapter {
	out := make([]virtualbox.NetworkAdapter, 0, len(models))
	for _, m := range models {
		out = append(out, virtualbox.NetworkAdapter{
			Type:           m.Type.ValueString(),
			NetworkName:    m.NetworkName.ValueString(),
			NICType:        m.NICType.ValueString(),
			MACAddress:     m.MACAddress.ValueString(),
			CableConnected: m.CableConnected.ValueBool(),
		})
	}
	return out
}

// adaptersToModel converts client adapter structs into schema block models,
// mapping an empty network name to null to avoid spurious diffs.
func adaptersToModel(adapters []virtualbox.NetworkAdapter) []networkAdapterModel {
	out := make([]networkAdapterModel, 0, len(adapters))
	for _, a := range adapters {
		networkName := types.StringNull()
		if a.NetworkName != "" {
			networkName = types.StringValue(a.NetworkName)
		}
		out = append(out, networkAdapterModel{
			Type:           types.StringValue(a.Type),
			NetworkName:    networkName,
			NICType:        types.StringValue(a.NICType),
			CableConnected: types.BoolValue(a.CableConnected),
			MACAddress:     types.StringValue(a.MACAddress),
		})
	}
	return out
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
	state.Memory = types.Int64Value(int64(vm.MemoryMB))
	state.CPUs = types.Int64Value(int64(vm.CPUs))
	state.VRAM = types.Int64Value(int64(vm.VRAM))
	state.Status = types.StringValue(vm.Status)
	// os_type is preserved from state rather than refreshed: VirtualBox normalises
	// the value (e.g. "Other" is reported as "Other/Unknown" by showvminfo), so
	// writing it back would cause a permanent diff against the configured value.

	// Refresh network adapters only when they are managed (already in state), so an
	// unmanaged VM doesn't show a diff for VirtualBox's default NAT adapter.
	if len(state.NetworkAdapters) > 0 {
		state.NetworkAdapters = adaptersToModel(vm.Adapters)
	}

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

	if len(plan.NetworkAdapters) > 0 {
		if err := r.client.ConfigureNetworkAdapters(ctx, vmName, adaptersFromModel(plan.NetworkAdapters)); err != nil {
			resp.Diagnostics.AddError("Error configuring network adapters", err.Error())
			return
		}
	}

	if wasRunning {
		err = r.client.StartVM(ctx, vmName)
		if err != nil {
			resp.Diagnostics.AddError("Error restarting VM after hardware update", err.Error())
			return
		}
	}

	// Update status and adapters from Read.
	vmUpdated, _ := r.client.ReadVM(ctx, vmName)
	if vmUpdated != nil {
		plan.Status = types.StringValue(vmUpdated.Status)
		if len(plan.NetworkAdapters) > 0 {
			plan.NetworkAdapters = adaptersToModel(vmUpdated.Adapters)
		}
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
		_ = r.client.StopVM(ctx, vmName)
	}

	err = r.client.DeleteVM(ctx, vmName, true)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting VM", err.Error())
		return
	}

	// unregistervm --delete already removes disks that live inside the VM's own
	// folder. Only clean up a custom disk stored outside that folder, and surface
	// (rather than swallow) any failure to do so.
	if diskPath != "" {
		defaultDir := filepath.Dir(diskPathForVM(vmName))
		if !strings.HasPrefix(filepath.Clean(diskPath), filepath.Clean(defaultDir)) {
			if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
				resp.Diagnostics.AddWarning(
					"Could not remove custom disk",
					fmt.Sprintf("VM %q was deleted but its disk %q could not be removed: %s", vmName, diskPath, err),
				)
			}
		}
	}
}

// ImportState imports a resource state.
func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
