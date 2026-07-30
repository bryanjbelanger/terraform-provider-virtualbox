package virtualbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrNetworkNotFound is returned when a requested host-only network is not found.
var ErrNetworkNotFound = errors.New("host-only network not found")

// Network represents a VirtualBox host-only network.
type Network struct {
	Name        string
	GUID        string
	DHCP        bool
	NetworkCIDR string
	NetworkMask string
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

// netmaskFromCIDR converts a CIDR (e.g. "192.168.56.0/24") into the dotted-decimal
// netmask (e.g. "255.255.255.0") that VBoxManage's --netmask flag expects. An empty
// CIDR yields an empty mask so callers can omit the flag.
func netmaskFromCIDR(cidr string) (string, error) {
	if cidr == "" {
		return "", nil
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid network_cidr %q: %w", cidr, err)
	}
	return net.IP(ipNet.Mask).String(), nil
}

// DeriveDHCPRange returns a default DHCP lower/upper IP pair spanning the usable
// range of the given IPv4 CIDR (network+1 .. broadcast-1). VBoxManage requires a
// range at network-creation time, so this supplies one when the user omits it.
func DeriveDHCPRange(cidr string) (lower string, upper string, err error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	network := ipNet.IP.To4()
	if network == nil {
		return "", "", fmt.Errorf("only IPv4 CIDRs are supported, got %q", cidr)
	}
	mask := ipNet.Mask

	broadcast := make(net.IP, len(network))
	for i := range network {
		broadcast[i] = network[i] | ^mask[i]
	}

	lowerIP := make(net.IP, len(network))
	copy(lowerIP, network)
	incIP(lowerIP)

	upperIP := make(net.IP, len(broadcast))
	copy(upperIP, broadcast)
	decIP(upperIP)

	return lowerIP.String(), upperIP.String(), nil
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func decIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] != 0 {
			ip[i]--
			break
		}
		ip[i]--
	}
}

// CreateNetwork creates a new host-only network.
//
// NOTE: VirtualBox 7.x "hostonlynet" uses --enable/--disable to toggle the network
// (there is no --dhcp flag), and --lower-ip/--upper-ip for the DHCP range. The `dhcp`
// attribute is mapped to the enabled state of the network.
func (c *Client) CreateNetwork(ctx context.Context, params CreateNetworkParams) (*Network, error) {
	mask, err := netmaskFromCIDR(params.NetworkCIDR)
	if err != nil {
		return nil, err
	}

	args := []string{"hostonlynet", "add", "--name", params.Name}
	if mask != "" {
		args = append(args, "--netmask", mask)
	}
	if params.LowerIP != "" {
		args = append(args, "--lower-ip", params.LowerIP)
	}
	if params.UpperIP != "" {
		args = append(args, "--upper-ip", params.UpperIP)
	}
	if params.DHCP {
		args = append(args, "--enable")
	} else {
		args = append(args, "--disable")
	}

	if _, err := c.RunContext(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return c.ReadNetwork(ctx, params.Name)
}

// ReadNetwork retrieves information about a host-only network. It returns
// ErrNetworkNotFound (wrapped) when no network with the given name exists so
// callers can distinguish "gone" from a genuine failure.
func (c *Client) ReadNetwork(ctx context.Context, name string) (*Network, error) {
	networks, err := c.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}

	for i := range networks {
		if networks[i].Name == name {
			return &networks[i], nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrNetworkNotFound, name)
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
	mask, err := netmaskFromCIDR(params.NetworkCIDR)
	if err != nil {
		return nil, err
	}

	args := []string{"hostonlynet", "modify", "--name", params.Name}
	if mask != "" {
		args = append(args, "--netmask", mask)
	}
	if params.LowerIP != "" {
		args = append(args, "--lower-ip", params.LowerIP)
	}
	if params.UpperIP != "" {
		args = append(args, "--upper-ip", params.UpperIP)
	}
	if params.DHCP != nil {
		if *params.DHCP {
			args = append(args, "--enable")
		} else {
			args = append(args, "--disable")
		}
	}

	// Only issue the command when there is something to change beyond the selector.
	if len(args) > 4 {
		if _, err := c.RunContext(ctx, args...); err != nil {
			return nil, fmt.Errorf("failed to update network: %w", err)
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
//
// Each record begins with a "Name:" line and spans several key/value lines. Note
// that VirtualBox 7.x prints a blank line *within* a record (between GUID and
// State), so records are delimited by the "Name:" key rather than by blank lines.
func parseNetworkList(output string) []Network {
	var networks []Network
	var current *Network

	flush := func() {
		if current != nil {
			networks = append(networks, *current)
			current = nil
		}
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			// Start of a new record.
			flush()
			current = &Network{Name: value}
		case "GUID":
			if current != nil {
				current.GUID = value
			}
		case "State", "Enabled":
			if current != nil {
				current.DHCP = value == "Enabled" || value == "enabled" || value == "yes"
			}
		case "NetworkMask":
			// Kept as the raw mask; network_cidr is preserved from configuration
			// rather than reconstructed here to avoid round-trip drift.
			if current != nil {
				current.NetworkMask = value
			}
		case "LowerIP":
			if current != nil {
				current.LowerIP = value
			}
		case "UpperIP":
			if current != nil {
				current.UpperIP = value
			}
		}
	}

	flush()
	return networks
}
