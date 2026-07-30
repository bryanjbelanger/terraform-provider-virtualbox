package virtualbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrSharedFolderNotFound is returned when a requested shared folder is not found on a VM.
var ErrSharedFolderNotFound = errors.New("shared folder not found")

// SharedFolder represents a VirtualBox shared folder.
type SharedFolder struct {
	Name      string
	HostPath  string
	Writable  bool
	AutoMount bool
}

// CreateSharedFolderParams holds parameters for adding a shared folder.
type CreateSharedFolderParams struct {
	VMName    string
	Name      string
	HostPath  string
	Writable  bool
	AutoMount bool
}

// CreateSharedFolder adds a shared folder to a VM.
func (c *Client) CreateSharedFolder(ctx context.Context, params CreateSharedFolderParams) (*SharedFolder, error) {
	args := []string{
		"sharedfolder", "add", params.VMName,
		"--name", params.Name,
		"--hostpath", params.HostPath,
	}

	// VBoxManage shares folders writable by default; there is no --writable flag.
	// Read-only access is opt-in via --readonly.
	if !params.Writable {
		args = append(args, "--readonly")
	}
	if params.AutoMount {
		args = append(args, "--automount")
	}

	_, err := c.RunContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to add shared folder: %w", err)
	}

	return &SharedFolder{
		Name:      params.Name,
		HostPath:  params.HostPath,
		Writable:  params.Writable,
		AutoMount: params.AutoMount,
	}, nil
}

// ReadSharedFolder retrieves information about a shared folder on a VM.
func (c *Client) ReadSharedFolder(ctx context.Context, vmName, folderName string) (*SharedFolder, error) {
	folders, err := c.ListSharedFolders(ctx, vmName)
	if err != nil {
		return nil, err
	}

	for i := range folders {
		if folders[i].Name == folderName {
			return &folders[i], nil
		}
	}

	return nil, fmt.Errorf("%w: %q on VM %q", ErrSharedFolderNotFound, folderName, vmName)
}

// UpdateSharedFolderParams holds parameters for updating a shared folder.
type UpdateSharedFolderParams struct {
	VMName    string
	Name      string
	HostPath  string
	Writable  bool
	AutoMount bool
}

// UpdateSharedFolder modifies a shared folder on a VM.
func (c *Client) UpdateSharedFolder(ctx context.Context, params UpdateSharedFolderParams) (*SharedFolder, error) {
	// VirtualBox doesn't support modifying shared folders in-place.
	// Remove and re-add.
	err := c.DeleteSharedFolder(ctx, params.VMName, params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to remove shared folder for update: %w", err)
	}

	return c.CreateSharedFolder(ctx, CreateSharedFolderParams(params))
}

// DeleteSharedFolder removes a shared folder from a VM.
func (c *Client) DeleteSharedFolder(ctx context.Context, vmName, folderName string) error {
	_, err := c.RunContext(ctx, "sharedfolder", "remove", vmName, "--name", folderName)
	if err != nil {
		return fmt.Errorf("failed to remove shared folder: %w", err)
	}
	return nil
}

// ListSharedFolders returns all shared folders for a given VM.
func (c *Client) ListSharedFolders(ctx context.Context, vmName string) ([]SharedFolder, error) {
	output, err := c.RunContext(ctx, "showvminfo", vmName, "--machinereadable")
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	return parseSharedFolders(output), nil
}

// parseSharedFolders parses shared folder entries from VBoxManage showvminfo output.
//
// The machine-readable output lists each mapping across two keys sharing a numeric
// suffix, e.g.:
//
//	SharedFolderNameMachineMapping1="data"
//	SharedFolderPathMachineMapping1="/host/data"
//
// so name and host path are correlated by that suffix. Writable/automount flags are
// not exposed by showvminfo, so they are left to the caller to preserve.
func parseSharedFolders(output string) []SharedFolder {
	const (
		namePrefix = "SharedFolderNameMachineMapping"
		pathPrefix = "SharedFolderPathMachineMapping"
	)

	names := map[string]string{}
	paths := map[string]string{}
	var order []string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], "\"")

		switch {
		case strings.HasPrefix(key, namePrefix):
			idx := strings.TrimPrefix(key, namePrefix)
			if _, seen := names[idx]; !seen {
				order = append(order, idx)
			}
			names[idx] = value
		case strings.HasPrefix(key, pathPrefix):
			paths[strings.TrimPrefix(key, pathPrefix)] = value
		}
	}

	folders := make([]SharedFolder, 0, len(order))
	for _, idx := range order {
		folders = append(folders, SharedFolder{
			Name:     names[idx],
			HostPath: paths[idx],
		})
	}
	return folders
}
