package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanbelanger/terraform-provider-virtualbox/virtualbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceVM returns the schema for the virtualbox_vm resource.
func ResourceVM() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVMCreate,
		ReadContext:   resourceVMRead,
		UpdateContext: resourceVMUpdate,
		DeleteContext: resourceVMDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the virtual machine.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Other",
				Description: "The guest OS type (e.g., 'Ubuntu_64', 'Windows10_64', 'Other').",
			},
			"memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1024,
				Description: "Amount of RAM in MB.",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Number of CPU cores.",
			},
			"vram": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     8,
				Description: "Video RAM in MB.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the VM (e.g., 'running', 'poweroff').",
			},
			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the virtual machine.",
			},
			// --- New fields for ISO boot and disk ---
			"iso_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to an ISO image to attach as a DVD drive for installation.",
			},
			"iso_controller": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "IDE",
				Description: "Storage controller name to attach the ISO to. Default: 'IDE'.",
			},
			"disk_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Path to the virtual disk (VDI). Defaults to ~/VirtualBox VMs/<name>/<name>.vdi.",
			},
			"disk_size_mb": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Size of the virtual disk in MB. Set to 0 to skip disk creation.",
			},
			"storage_controller": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "SATA",
				Description: "Name for the SATA storage controller. Default: 'SATA'.",
			},
			"start_on_create": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Start the VM after creation.",
			},
		},
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceVMCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vmName := d.Get("name").(string)

	// Step 1: Create the VM
	params := virtualbox.CreateVMParams{
		Name:   vmName,
		OSType: d.Get("os_type").(string),
		Memory: d.Get("memory").(int),
		CPUs:   d.Get("cpus").(int),
		VRAM:   d.Get("vram").(int),
	}

	vm, err := client.CreateVM(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating VM: %w", err))
	}

	d.SetId(vm.Name)

	// Step 2: Add storage controllers
	ctrlName := d.Get("storage_controller").(string)
	if err := client.AddStorageController(ctx, vmName, ctrlName, virtualbox.ControllerSATA); err != nil {
		return diag.FromErr(fmt.Errorf("error adding SATA controller: %w", err))
	}

	// We also need an IDE controller for the ISO (DVD drives need IDE or SATA)
	isoCtrlName := d.Get("iso_controller").(string)
	if isoCtrlName != "" && isoCtrlName != ctrlName {
		if err := client.AddStorageController(ctx, vmName, isoCtrlName, virtualbox.ControllerIDE); err != nil {
			return diag.FromErr(fmt.Errorf("error adding IDE controller: %w", err))
		}
	}

	// Step 3: Create and attach disk
	diskSize := d.Get("disk_size_mb").(int)
	if diskSize > 0 {
		diskPath := d.Get("disk_path").(string)
		if diskPath == "" {
			diskPath = diskPathForVM(vmName)
			d.Set("disk_path", diskPath)
		}
		// Ensure directory exists
		diskDir := filepath.Dir(diskPath)
		if err := os.MkdirAll(diskDir, 0755); err != nil {
			return diag.FromErr(fmt.Errorf("error creating disk directory %s: %w", diskDir, err))
		}
		if err := client.CreateDisk(ctx, diskPath, diskSize); err != nil {
			return diag.FromErr(fmt.Errorf("error creating disk: %w", err))
		}
		if err := client.AttachDisk(ctx, vmName, ctrlName, diskPath, 0); err != nil {
			return diag.FromErr(fmt.Errorf("error attaching disk: %w", err))
		}
	}

	// Step 4: Attach ISO if specified
	isoPath := d.Get("iso_path").(string)
	if isoPath != "" {
		attachCtrlName := isoCtrlName
		if attachCtrlName == "" {
			attachCtrlName = ctrlName
		}
		if err := client.AttachISO(ctx, vmName, attachCtrlName, isoPath); err != nil {
			return diag.FromErr(fmt.Errorf("error attaching ISO: %w", err))
		}
	}

	// Step 5: Set boot order to DVD first, then disk
	if isoPath != "" {
		client.RunContext(ctx, "modifyvm", vmName, "--boot1", "dvd", "--boot2", "disk")
	} else {
		client.RunContext(ctx, "modifyvm", vmName, "--boot1", "disk")
	}

	// Step 6: Start VM if requested
	startOnCreate := d.Get("start_on_create").(bool)
	if startOnCreate {
		if err := client.StartVM(ctx, vmName); err != nil {
			return diag.FromErr(fmt.Errorf("error starting VM %s: %w", vmName, err))
		}
	}

	return resourceVMRead(ctx, d, meta)
}

func resourceVMRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vm, err := client.ReadVM(ctx, d.Id())
	if err != nil {
		// If VM is not found, remove from state
		if errors.Is(err, virtualbox.ErrVMNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error reading VM: %w", err))
	}

	d.Set("name", vm.Name)
	d.Set("os_type", vm.OSType)
	d.Set("memory", vm.MemoryMB)
	d.Set("cpus", vm.CPUs)
	d.Set("vram", vm.VRAM)
	d.Set("status", vm.Status)
	d.Set("uuid", vm.UUID)

	return nil
}

func resourceVMUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vmName := d.Id()

	// 1. Check current status
	vm, err := client.ReadVM(ctx, vmName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading VM before update: %w", err))
	}

	wasRunning := (vm.Status == "running")

	// 2. Power off if it is running to allow hardware modifications
	if wasRunning {
		err = client.StopVM(ctx, vmName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error stopping VM for hardware update: %w", err))
		}
	}

	// 3. Apply modifications
	params := virtualbox.UpdateVMParams{
		Name:   vmName,
		OSType: d.Get("os_type").(string),
		Memory: d.Get("memory").(int),
		CPUs:   d.Get("cpus").(int),
		VRAM:   d.Get("vram").(int),
	}

	_, err = client.UpdateVM(ctx, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating VM: %w", err))
	}

	// 4. Restore power state
	if wasRunning {
		err = client.StartVM(ctx, vmName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error restarting VM after hardware update: %w", err))
		}
	}

	return resourceVMRead(ctx, d, meta)
}

func resourceVMDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*virtualbox.Client)

	vmName := d.Id()

	// Read disk path from state before deleting
	diskPath, _ := d.Get("disk_path").(string)

	// Power off if running
	vm, err := client.ReadVM(ctx, vmName)
	if err == nil && vm.Status == "running" {
		client.StopVM(ctx, vmName)
		// Wait a moment for VM to stop
	}

	// Delete the VM and its files
	err = client.DeleteVM(ctx, vmName, true)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting VM: %w", err))
	}

	// Remove the disk file if it's not inside the VM's directory (already deleted by --delete)
	// If disk was stored elsewhere, we should clean it up
	if diskPath != "" && !strings.Contains(diskPath, vmName) {
		os.Remove(diskPath)
	}

	d.SetId("")
	return nil
}

// diskPathForVM returns the default disk path for a VM.
func diskPathForVM(vmName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "VirtualBox VMs", vmName, vmName+".vdi")
}
