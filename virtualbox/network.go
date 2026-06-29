package virtualbox

import (
	"context"
	"fmt"
	"strings"
)

// Network represents a VirtualBox host-only network.
type Network struct {
	Name        string
	GUID        string
	DHCP        bool
	NetworkCIDR string
	LowerIP     string
	UpperIP     string
}

// CreateNetworkParams holds parameters for creating a host-only network.
type CreateNetworkParams struct {
	Name        string
	NetworkCIDR string
	DHCP        bool
	LowerIP     string
	UpperIP     string
}

// CreateNetwork creates a new host-only network.
func (c *Client) CreateNetwork(ctx context.Context, params CreateNetworkParams) (*Network, error) {
	// Create host-only network
	_, err := c.RunContext(ctx, "hostonlynet", "add", "--name", params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Configure network CIDR
	if params.NetworkCIDR != "" {
		_, err = c.RunContext(ctx, "hostonlynet", "modify", "--name", params.Name, "--netmask", params.NetworkCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to set network CIDR: %w", err)
		}
	}

	// Configure DHCP
	if params.DHCP {
		dhcpArgs := []string{"hostonlynet", "modify", "--name", params.Name, "--dhcp", "on"}
		if params.LowerIP != "" {
			dhcpArgs = append(dhcpArgs, "--ip", params.LowerIP)
		}
		if params.UpperIP != "" {
			dhcpArgs = append(dhcpArgs, "--netmask", params.UpperIP)
		}
		_, err = c.RunContext(ctx, dhcpArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to enable DHCP: %w", err)
		}
	} else {
		_, err = c.RunContext(ctx, "hostonlynet", "modify", "--name", params.Name, "--dhcp", "off")
		if err != nil {
			return nil, fmt.Errorf("failed to disable DHCP: %w", err)
		}
	}

	return c.ReadNetwork(ctx, params.Name)
}

// ReadNetwork retrieves information about a host-only network.
func (c *Client) ReadNetwork(ctx context.Context, name string) (*Network, error) {
	networks, err := c.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}

	for _, n := range networks {
		if n.Name == name {
			return &n, nil
		}
	}

	return nil, fmt.Errorf("network '%s' not found", name)
}

// UpdateNetworkParams holds parameters for updating a host-only network.
type UpdateNetworkParams struct {
	Name        string
	NetworkCIDR string
	DHCP        *bool
	LowerIP     string
	UpperIP     string
}

// UpdateNetwork modifies an existing host-only network.
func (c *Client) UpdateNetwork(ctx context.Context, params UpdateNetworkParams) (*Network, error) {
	if params.NetworkCIDR != "" {
		_, err := c.RunContext(ctx, "hostonlynet", "modify", "--name", params.Name, "--netmask", params.NetworkCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to update network CIDR: %w", err)
		}
	}

	if params.DHCP != nil {
		dhcpValue := "off"
		if *params.DHCP {
			dhcpValue = "on"
		}
		_, err := c.RunContext(ctx, "hostonlynet", "modify", "--name", params.Name, "--dhcp", dhcpValue)
		if err != nil {
			return nil, fmt.Errorf("failed to update network DHCP: %w", err)
		}
	}

	return c.ReadNetwork(ctx, params.Name)
}

// DeleteNetwork removes a host-only network.
func (c *Client) DeleteNetwork(ctx context.Context, name string) error {
	_, err := c.RunContext(ctx, "hostonlynet", "remove", "--name", name)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}
	return nil
}

// ListNetworks returns all host-only networks.
func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	output, err := c.RunContext(ctx, "list", "hostonlynets")
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	return parseNetworkList(output), nil
}

// parseNetworkList parses the output of "VBoxManage list hostonlynets".
func parseNetworkList(output string) []Network {
	var networks []Network
	var current *Network

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				networks = append(networks, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if current == nil {
			current = &Network{}
		}

		switch key {
		case "Name":
			current.Name = value
		case "GUID":
			current.GUID = value
		case "DHCP":
			current.DHCP = value == "Enabled"
		case "NetworkMask":
			// This is a simplification; real parsing would need more logic
			current.NetworkCIDR = value
		case "LowerIP":
			current.LowerIP = value
		case "UpperIP":
			current.UpperIP = value
		}
	}

	// Handle last entry if no trailing newline
	if current != nil {
		networks = append(networks, *current)
	}

	return networks
}
