package virtualbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
)

// ErrNetworkNotFound is returned when a requested host-only network is not found.
var ErrNetworkNotFound = errors.New("host-only network not found")

// Host-only networking backends. VirtualBox has two mutually exclusive
// mechanisms, selected by host OS:
//
//   - BackendHostOnlyNet: "host-only networks" (`VBoxManage hostonlynet`),
//     backed by vmnet. Only exists on macOS and Solaris hosts.
//   - BackendHostOnlyIf: legacy "host-only interfaces" (`VBoxManage
//     hostonlyif`, vboxnet0-style). The only mechanism on Linux and Windows
//     hosts, where the hostonlynet subcommand is not even recognized.
const (
	BackendHostOnlyNet = "hostonlynet"
	BackendHostOnlyIf  = "hostonlyif"
)

// defaultNetworkBackend returns the host-only backend the current host OS
// supports.
func defaultNetworkBackend() string {
	switch runtime.GOOS {
	case "darwin", "solaris", "illumos":
		return BackendHostOnlyNet
	default:
		return BackendHostOnlyIf
	}
}

// Network represents a VirtualBox host-only network, regardless of which
// backend provides it.
type Network struct {
	// Backend is BackendHostOnlyNet or BackendHostOnlyIf.
	Backend string
	Name    string
	GUID    string
	DHCP    bool
	// HostInterface is the auto-assigned interface name (e.g. "vboxnet0") on
	// the hostonlyif backend; empty on hostonlynet.
	HostInterface string
	// VBoxNetworkName is the internal network name as reported by VirtualBox
	// (hostonlyif backend only, e.g. "HostInterfaceNetworking-vboxnet0").
	VBoxNetworkName string
	NetworkCIDR     string
	NetworkMask     string
	LowerIP         string
	UpperIP         string
}

// AdapterType returns the network_adapter attachment type VMs must use to join
// this network: "hostonlynet" or "hostonly" depending on the backend.
func (n *Network) AdapterType() string {
	if n.Backend == BackendHostOnlyIf {
		return "hostonly"
	}
	return "hostonlynet"
}

// AdapterNetwork returns the network_adapter network_name VMs must use to join
// this network: the network's name on hostonlynet, the interface name on
// hostonlyif.
func (n *Network) AdapterNetwork() string {
	if n.Backend == BackendHostOnlyIf {
		return n.HostInterface
	}
	return n.Name
}

// DHCPNetworkName returns the internal network name a `VBoxManage dhcpserver`
// must bind to (--network) to serve this segment: "hostonly-<name>" on
// hostonlynet, "HostInterfaceNetworking-<interface>" on hostonlyif.
func (n *Network) DHCPNetworkName() string {
	if n.Backend == BackendHostOnlyIf {
		if n.VBoxNetworkName != "" {
			return n.VBoxNetworkName
		}
		return "HostInterfaceNetworking-" + n.HostInterface
	}
	return "hostonly-" + n.Name
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

// CreateNetwork creates a new host-only network using whichever backend the
// host OS supports.
//
// NOTE: VirtualBox 7.x "hostonlynet" uses --enable/--disable to toggle the network
// (there is no --dhcp flag), and --lower-ip/--upper-ip for the DHCP range. The `dhcp`
// attribute is mapped to the enabled state of the network.
func (c *Client) CreateNetwork(ctx context.Context, params CreateNetworkParams) (*Network, error) {
	mask, err := netmaskFromCIDR(params.NetworkCIDR)
	if err != nil {
		return nil, err
	}

	if c.networkBackend == BackendHostOnlyIf {
		return c.createHostOnlyIfNetwork(ctx, params, mask)
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

	return c.ReadNetwork(ctx, params.Name, "")
}

// createHostOnlyIfNetwork creates a legacy host-only interface and assigns it
// the range's lower IP — on this backend the host takes the lower bound,
// mirroring how hostonlynet hands the host the network's lower bound. The DHCP
// range itself lives on a separate `VBoxManage dhcpserver` (see
// Network.DHCPNetworkName), so params.DHCP and params.UpperIP are echoed back
// rather than applied.
func (c *Client) createHostOnlyIfNetwork(ctx context.Context, params CreateNetworkParams, mask string) (*Network, error) {
	iface, err := c.createHostOnlyInterface(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// A DHCP server bound to this interface's network name necessarily
	// predates the interface (VirtualBox's stock vboxnet0 server, or one
	// orphaned by an earlier removal), so its configuration is stale.
	c.scrubDHCPServer(ctx, "HostInterfaceNetworking-"+iface)

	if params.LowerIP != "" {
		if err := c.configureHostOnlyInterface(ctx, iface, params.LowerIP, mask); err != nil {
			// Don't leak a half-configured interface VirtualBox auto-named for
			// us; nothing references it yet.
			_ = c.removeHostOnlyInterface(ctx, iface)
			return nil, fmt.Errorf("failed to create network: %w", err)
		}
	}

	network, err := c.ReadNetwork(ctx, params.Name, iface)
	if err != nil {
		return nil, err
	}
	network.Name = params.Name
	network.NetworkCIDR = params.NetworkCIDR
	network.DHCP = params.DHCP
	network.UpperIP = params.UpperIP
	return network, nil
}

// ReadNetwork retrieves information about a host-only network. It returns
// ErrNetworkNotFound (wrapped) when no matching network exists so callers can
// distinguish "gone" from a genuine failure.
//
// On the hostonlyif backend the lookup key is the interface name
// (hostInterface, falling back to name so that imports by interface name
// work); on hostonlynet it is the network name and hostInterface is ignored.
func (c *Client) ReadNetwork(ctx context.Context, name, hostInterface string) (*Network, error) {
	if c.networkBackend == BackendHostOnlyIf {
		key := hostInterface
		if key == "" {
			key = name
		}
		interfaces, err := c.listHostOnlyInterfaces(ctx)
		if err != nil {
			return nil, err
		}
		for i := range interfaces {
			if interfaces[i].HostInterface == key {
				return &interfaces[i], nil
			}
		}
		return nil, fmt.Errorf("%w: %q", ErrNetworkNotFound, key)
	}

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
	// HostInterface identifies the backing interface on the hostonlyif
	// backend; ignored on hostonlynet.
	HostInterface string
}

// UpdateNetwork modifies an existing host-only network.
func (c *Client) UpdateNetwork(ctx context.Context, params UpdateNetworkParams) (*Network, error) {
	mask, err := netmaskFromCIDR(params.NetworkCIDR)
	if err != nil {
		return nil, err
	}

	if c.networkBackend == BackendHostOnlyIf {
		return c.updateHostOnlyIfNetwork(ctx, params, mask)
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

	return c.ReadNetwork(ctx, params.Name, "")
}

// updateHostOnlyIfNetwork re-applies the host address and netmask to the
// backing interface. DHCP and the upper bound have no interface-level
// representation (see createHostOnlyIfNetwork), so they are echoed back.
func (c *Client) updateHostOnlyIfNetwork(ctx context.Context, params UpdateNetworkParams, mask string) (*Network, error) {
	if params.HostInterface == "" {
		return nil, fmt.Errorf("failed to update network %q: no backing host interface recorded", params.Name)
	}

	if params.LowerIP != "" {
		if err := c.configureHostOnlyInterface(ctx, params.HostInterface, params.LowerIP, mask); err != nil {
			return nil, fmt.Errorf("failed to update network: %w", err)
		}
	}

	network, err := c.ReadNetwork(ctx, params.Name, params.HostInterface)
	if err != nil {
		return nil, err
	}
	network.Name = params.Name
	network.NetworkCIDR = params.NetworkCIDR
	if params.DHCP != nil {
		network.DHCP = *params.DHCP
	}
	network.UpperIP = params.UpperIP
	return network, nil
}

// DeleteNetwork removes a host-only network. On the hostonlyif backend the
// backing interface (hostInterface, falling back to name) is removed instead.
func (c *Client) DeleteNetwork(ctx context.Context, name, hostInterface string) error {
	if c.networkBackend == BackendHostOnlyIf {
		iface := hostInterface
		if iface == "" {
			iface = name
		}
		// DHCP servers are registered by network name and would outlive the
		// interface, poisoning a future interface that reuses the name.
		c.scrubDHCPServer(ctx, "HostInterfaceNetworking-"+iface)
		if err := c.removeHostOnlyInterface(ctx, iface); err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}
		return nil
	}

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
			current = &Network{Backend: BackendHostOnlyNet, Name: value}
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
