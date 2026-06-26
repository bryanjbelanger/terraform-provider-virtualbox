package virtualbox

import (
	"fmt"
	"strconv"
	"strings"
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
func (c *Client) CreateVM(params CreateVMParams) (*VM, error) {
	// Create the VM
	args := []string{
		"createvm",
		"--name", params.Name,
		"--ostype", params.OSType,
		"--register",
	}
	_, err := c.Run(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	// Configure memory
	if params.Memory > 0 {
		_, err = c.Run("modifyvm", params.Name, "--memory", strconv.Itoa(params.Memory))
		if err != nil {
			return nil, fmt.Errorf("failed to set memory: %w", err)
		}
	}

	// Configure CPUs
	if params.CPUs > 0 {
		_, err = c.Run("modifyvm", params.Name, "--cpus", strconv.Itoa(params.CPUs))
		if err != nil {
			return nil, fmt.Errorf("failed to set CPUs: %w", err)
		}
	}

	// Configure VRAM
	if params.VRAM > 0 {
		_, err = c.Run("modifyvm", params.Name, "--vram", strconv.Itoa(params.VRAM))
		if err != nil {
			return nil, fmt.Errorf("failed to set VRAM: %w", err)
		}
	}

	return c.ReadVM(params.Name)
}

// ReadVM retrieves information about a VM by name or UUID.
func (c *Client) ReadVM(nameOrUUID string) (*VM, error) {
	output, err := c.Run("showvminfo", nameOrUUID, "--machinereadable")
	if err != nil {
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
func (c *Client) UpdateVM(params UpdateVMParams) (*VM, error) {
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
		_, err := c.Run(args...)
		if err != nil {
			return nil, fmt.Errorf("failed to update VM: %w", err)
		}
	}

	return c.ReadVM(params.Name)
}

// DeleteVM removes a VM and optionally deletes its files.
func (c *Client) DeleteVM(nameOrUUID string, deleteFiles bool) error {
	args := []string{"unregistervm", nameOrUUID}
	if deleteFiles {
		args = append(args, "--delete")
	}

	_, err := c.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	return nil
}

// StartVM powers on a VM.
func (c *Client) StartVM(nameOrUUID string) error {
	_, err := c.Run("startvm", nameOrUUID, "--type", "headless")
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	return nil
}

// StopVM powers off a VM.
func (c *Client) StopVM(nameOrUUID string) error {
	_, err := c.Run("controlvm", nameOrUUID, "poweroff")
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}
	return nil
}

// ListVMs returns all registered VMs.
func (c *Client) ListVMs() ([]VM, error) {
	output, err := c.Run("list", "vms")
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
