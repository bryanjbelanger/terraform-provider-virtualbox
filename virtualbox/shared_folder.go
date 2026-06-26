package virtualbox

import (
	"fmt"
	"strings"
)

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
func (c *Client) CreateSharedFolder(params CreateSharedFolderParams) (*SharedFolder, error) {
	args := []string{
		"sharedfolder", "add", params.VMName,
		"--name", params.Name,
		"--hostpath", params.HostPath,
	}

	if params.Writable {
		args = append(args, "--writable")
	}
	if params.AutoMount {
		args = append(args, "--automount")
	}

	_, err := c.Run(args...)
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
func (c *Client) ReadSharedFolder(vmName, folderName string) (*SharedFolder, error) {
	folders, err := c.ListSharedFolders(vmName)
	if err != nil {
		return nil, err
	}

	for _, f := range folders {
		if f.Name == folderName {
			return &f, nil
		}
	}

	return nil, fmt.Errorf("shared folder '%s' not found on VM '%s'", folderName, vmName)
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
func (c *Client) UpdateSharedFolder(params UpdateSharedFolderParams) (*SharedFolder, error) {
	// VirtualBox doesn't support modifying shared folders in-place.
	// Remove and re-add.
	err := c.DeleteSharedFolder(params.VMName, params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to remove shared folder for update: %w", err)
	}

	return c.CreateSharedFolder(CreateSharedFolderParams{
		VMName:    params.VMName,
		Name:      params.Name,
		HostPath:  params.HostPath,
		Writable:  params.Writable,
		AutoMount: params.AutoMount,
	})
}

// DeleteSharedFolder removes a shared folder from a VM.
func (c *Client) DeleteSharedFolder(vmName, folderName string) error {
	_, err := c.Run("sharedfolder", "remove", vmName, "--name", folderName)
	if err != nil {
		return fmt.Errorf("failed to remove shared folder: %w", err)
	}
	return nil
}

// ListSharedFolders returns all shared folders for a given VM.
func (c *Client) ListSharedFolders(vmName string) ([]SharedFolder, error) {
	output, err := c.Run("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	return parseSharedFolders(output), nil
}

// parseSharedFolders parses shared folder entries from VBoxManage showvminfo output.
func parseSharedFolders(output string) []SharedFolder {
	var folders []SharedFolder
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "SharedFolderName") {
			continue
		}

		// Parse line like: SharedFolderNameMachineMapping1="folder_name"
		// followed by: SharedFolderPathMachineMapping1="/host/path"
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.Trim(parts[1], "\"")
		folders = append(folders, SharedFolder{Name: name})
	}

	return folders
}
