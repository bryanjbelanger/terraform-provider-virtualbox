package virtualbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrVMNotFound is returned when a requested virtual machine is not found.
	ErrVMNotFound = errors.New("virtual machine not found")
)

// VM represents a VirtualBox virtual machine.
type VM struct {
	Name         string
	UUID         string
	OSType       string
	MemoryMB     int
	CPUs         int
	Status       string
	VRAM         int
	Acceleration string
}

// CreateVMParams holds parameters for creating a new VM.
type CreateVMParams struct {
	Name   string
	OSType string
	Memory int
	CPUs   int
	VRAM   int
}

// CreateVM creates a new VirtualBox VM.
func (c *Client) CreateVM(ctx context.Context, params CreateVMParams) (*VM, error) {
	// Create the VM
	args := []string{
		"createvm",
		"--name", params.Name,
		"--ostype", params.OSType,
		"--register",
	}
	_, err := c.RunContext(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "VBOX_E_OBJECT_IN_USE") {
			// A ghost directory or corrupted registry entry exists from a reverted snapshot.
			// 0. Forcefully kill the VM if it happens to be running in the background!
			c.RunContext(ctx, "controlvm", params.Name, "poweroff")

			// 1. Aggressively purge it from VirtualBox.xml (ignore errors if it's not registered)
			c.RunContext(ctx, "unregistervm", params.Name, "--delete")

			// 2. Nuke the physical folder to be absolutely sure
			home, _ := os.UserHomeDir()
			ghostDir := filepath.Join(home, "VirtualBox VMs", params.Name)
			os.RemoveAll(ghostDir)

			// 3. Retry creation
			_, err = c.RunContext(ctx, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to create VM even after aggressive registry purge: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}
	}

	// Helper function for resilient modifyvm execution
	modifyWithRetry := func(args ...string) error {
		var lastErr error
		for i := 0; i < 5; i++ {
			_, err := c.RunContext(ctx, args...)
			if err == nil {
				return nil
			}
			if strings.Contains(err.Error(), "VBOX_E_INVALID_OBJECT_STATE") || strings.Contains(err.Error(), "already locked") {
				lastErr = err
				time.Sleep(1 * time.Second)
			} else {
				return err
			}
		}
		return fmt.Errorf("failed after 5 retries due to lock contention: %w", lastErr)
	}

	// Configure memory
	if params.Memory > 0 {
		if err := modifyWithRetry("modifyvm", params.Name, "--memory", strconv.Itoa(params.Memory)); err != nil {
			return nil, fmt.Errorf("failed to set memory: %w", err)
		}
	}

	// Configure CPUs
	if params.CPUs > 0 {
		if err := modifyWithRetry("modifyvm", params.Name, "--cpus", strconv.Itoa(params.CPUs)); err != nil {
			return nil, fmt.Errorf("failed to set CPUs: %w", err)
		}
	}

	// Configure VRAM
	if params.VRAM > 0 {
		if err := modifyWithRetry("modifyvm", params.Name, "--vram", strconv.Itoa(params.VRAM)); err != nil {
			return nil, fmt.Errorf("failed to set VRAM: %w", err)
		}
	}

	return c.ReadVM(ctx, params.Name)
}

// ReadVM retrieves information about a VM by name or UUID.
func (c *Client) ReadVM(ctx context.Context, nameOrUUID string) (*VM, error) {
	output, err := c.RunContext(ctx, "showvminfo", nameOrUUID, "--machinereadable")
	if err != nil {
		if strings.Contains(err.Error(), "Could not find a registered machine") ||
			strings.Contains(err.Error(), "Could not find a registered virtual machine") {
			return nil, fmt.Errorf("%w: %v", ErrVMNotFound, err)
		}
		return nil, fmt.Errorf("failed to read VM: %w", err)
	}

	return parseVMInfo(output), nil
}

// UpdateVMParams holds parameters for updating an existing VM.
type UpdateVMParams struct {
	Name   string
	OSType string
	Memory int
	CPUs   int
	VRAM   int
}

// UpdateVM modifies an existing VM's configuration.
func (c *Client) UpdateVM(ctx context.Context, params UpdateVMParams) (*VM, error) {
	args := []string{"modifyvm", params.Name}

	if params.Memory > 0 {
		args = append(args, "--memory", strconv.Itoa(params.Memory))
	}
	if params.CPUs > 0 {
		args = append(args, "--cpus", strconv.Itoa(params.CPUs))
	}
	if params.VRAM > 0 {
		args = append(args, "--vram", strconv.Itoa(params.VRAM))
	}

	// Only run modifyvm if there are changes
	if len(args) > 2 {
		_, err := c.RunContext(ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to update VM: %w", err)
		}
	}

	return c.ReadVM(ctx, params.Name)
}

// DeleteVM removes a VM and optionally deletes its files.
func (c *Client) DeleteVM(ctx context.Context, nameOrUUID string, deleteFiles bool) error {
	args := []string{"unregistervm", nameOrUUID}
	if deleteFiles {
		args = append(args, "--delete")
	}

	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	return nil
}

// StartVM powers on a VM.
func (c *Client) StartVM(ctx context.Context, nameOrUUID string) error {
	_, err := c.RunContext(ctx, "startvm", nameOrUUID, "--type", "headless")
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	return nil
}

// StopVM powers off a VM.
func (c *Client) StopVM(ctx context.Context, nameOrUUID string) error {
	_, err := c.RunContext(ctx, "controlvm", nameOrUUID, "poweroff")
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}
	return nil
}

// ListVMs returns all registered VMs.
func (c *Client) ListVMs(ctx context.Context) ([]VM, error) {
	output, err := c.RunContext(ctx, "list", "vms")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var vms []VM
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 1 {
			name := strings.Trim(parts[0], "\"")
			vms = append(vms, VM{Name: name})
		}
	}
	return vms, nil
}

// parseVMInfo parses the machine-readable output of showvminfo.
func parseVMInfo(output string) *VM {
	vm := &VM{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		switch key {
		case "name":
			vm.Name = value
		case "UUID":
			vm.UUID = value
		case "ostype":
			vm.OSType = value
		case "memory":
			if mem, err := strconv.Atoi(value); err == nil {
				vm.MemoryMB = mem
			}
		case "cpus":
			if cpus, err := strconv.Atoi(value); err == nil {
				vm.CPUs = cpus
			}
		case "VMState":
			vm.Status = value
		case "vram":
			if vram, err := strconv.Atoi(value); err == nil {
				vm.VRAM = vram
			}
		}
	}
	return vm
}
