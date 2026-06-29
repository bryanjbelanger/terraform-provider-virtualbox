package virtualbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// StorageControllerType represents the type of storage controller.
// Values must be lowercase as required by VirtualBox 7.x storagectl --type.
type StorageControllerType string

const (
	ControllerSATA StorageControllerType = "sata"
	ControllerIDE  StorageControllerType = "ide"
	ControllerSAS  StorageControllerType = "sas"
)

// controllerSubtype returns the VBoxManage --controller subtype name for a given StorageControllerType.
// VirtualBox requires a specific chipset/subtype name distinct from the bus type.
func controllerSubtype(ctype StorageControllerType) string {
	switch ctype {
	case ControllerSATA:
		return "IntelAhci"
	case ControllerIDE:
		return "PIIX4"
	case ControllerSAS:
		return "LSILogicSAS"
	default:
		return string(ctype)
	}
}

// AddStorageController adds a storage controller to the VM.
// Uses --add with a lowercase bus type (e.g. "sata", "ide") and a proper chipset
// subtype via --controller (e.g. "IntelAhci" for SATA). VBoxManage requires lowercase
// bus-type values; passing uppercase causes "Invalid --type argument" errors.
func (c *Client) AddStorageController(ctx context.Context, vmName string, name string, ctype StorageControllerType) error {
	args := []string{
		"storagectl", vmName,
		"--name", name,
		"--add", string(ctype),
		"--controller", controllerSubtype(ctype),
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			// Some VirtualBox versions (like 7.x) automatically create default storage
			// controllers upon VM creation. If it's already there, that's fine.
			return nil
		}
		return fmt.Errorf("failed to add %s controller %q: %w", ctype, name, err)
	}
	return nil
}

// CreateDisk creates a VirtualBox virtual disk (VDI).
func (c *Client) CreateDisk(ctx context.Context, diskPath string, sizeMB int) error {
	args := []string{
		"createmedium", "disk",
		"--filename", diskPath,
		"--size", fmt.Sprintf("%d", sizeMB),
		"--format", "VDI",
		"--variant", "Standard",
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to create disk at %s: %w", diskPath, err)
	}
	return nil
}

// AttachDisk attaches a disk image to a storage controller.
func (c *Client) AttachDisk(ctx context.Context, vmName string, controllerName string, diskPath string, port int) error {
	args := []string{
		"storageattach", vmName,
		"--storagectl", controllerName,
		"--port", fmt.Sprintf("%d", port),
		"--device", "0",
		"--type", "hdd",
		"--medium", diskPath,
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to attach disk %s to %s: %w", diskPath, vmName, err)
	}
	return nil
}

// AttachISO attaches an ISO image to a storage controller.
func (c *Client) AttachISO(ctx context.Context, vmName string, controllerName string, isoPath string) error {
	args := []string{
		"storageattach", vmName,
		"--storagectl", controllerName,
		"--port", "0",
		"--device", "0",
		"--type", "dvddrive",
		"--medium", isoPath,
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to attach ISO %s to %s: %w", isoPath, vmName, err)
	}
	return nil
}

// DetachISO removes the ISO from the DVD drive and empties it.
func (c *Client) DetachISO(ctx context.Context, vmName string, controllerName string) error {
	args := []string{
		"storageattach", vmName,
		"--storagectl", controllerName,
		"--port", "0",
		"--device", "0",
		"--type", "dvddrive",
		"--medium", "none",
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to detach ISO from %s: %w", vmName, err)
	}
	return nil
}

// DefaultDiskPath returns a default disk path for a VM in the default VBox folder.
func DefaultDiskPath(vmName string) string {
	return filepath.Join("~", "VirtualBox VMs", vmName, vmName+".vdi")
}

// ConvertToAbsolutePath expands ~ to the user's home directory.
func ConvertToAbsolutePath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		// The client.Run will handle VBoxManage path resolution, but disk paths
		// need to be absolute for VirtualBox. We expand to avoid issues.
		return filepath.Join("C:\\Users\\bryanbelanger", path[1:])
	}
	return path
}
