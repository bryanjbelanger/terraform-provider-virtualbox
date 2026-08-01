package virtualbox

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// hostonlyifCreatedRE extracts the auto-assigned interface name from the
// output of `VBoxManage hostonlyif create`, e.g.
// "Interface 'vboxnet0' was successfully created".
var hostonlyifCreatedRE = regexp.MustCompile(`Interface '([^']+)' was successfully created`)

// createHostOnlyInterface creates a legacy host-only interface and returns its
// auto-assigned name (vboxnet0, vboxnet1, ...). The name cannot be chosen.
func (c *Client) createHostOnlyInterface(ctx context.Context) (string, error) {
	output, err := c.RunContext(ctx, "hostonlyif", "create")
	if err != nil {
		return "", fmt.Errorf("failed to create host-only interface: %w", err)
	}

	m := hostonlyifCreatedRE.FindStringSubmatch(output)
	if m == nil {
		return "", fmt.Errorf("could not determine created interface name from VBoxManage output:\n%s", output)
	}
	return m[1], nil
}

// configureHostOnlyInterface assigns the host-side IPv4 address and netmask to
// a host-only interface.
func (c *Client) configureHostOnlyInterface(ctx context.Context, name, ip, netmask string) error {
	args := []string{"hostonlyif", "ipconfig", name, "--ip", ip}
	if netmask != "" {
		args = append(args, "--netmask", netmask)
	}
	if _, err := c.RunContext(ctx, args...); err != nil {
		return fmt.Errorf("failed to configure host-only interface %q: %w", name, err)
	}
	return nil
}

// scrubDHCPServer removes any DHCP server bound to the given internal network
// name, best-effort. On Linux/Windows, VirtualBox pairs a stock DHCP server
// (192.168.56.x) with the default host-only interface, and servers also
// outlive `hostonlyif remove` because they are registered by network name.
// Either way a server that predates the interface it now matches is stale
// configuration, so both create and delete clear it.
func (c *Client) scrubDHCPServer(ctx context.Context, networkName string) {
	_, _ = c.RunContext(ctx, "dhcpserver", "remove", "--network="+networkName)
}

// removeHostOnlyInterface removes a legacy host-only interface.
func (c *Client) removeHostOnlyInterface(ctx context.Context, name string) error {
	if _, err := c.RunContext(ctx, "hostonlyif", "remove", name); err != nil {
		return fmt.Errorf("failed to remove host-only interface %q: %w", name, err)
	}
	return nil
}

// listHostOnlyInterfaces returns all legacy host-only interfaces as Networks
// with the hostonlyif backend.
func (c *Client) listHostOnlyInterfaces(ctx context.Context) ([]Network, error) {
	output, err := c.RunContext(ctx, "list", "hostonlyifs")
	if err != nil {
		return nil, fmt.Errorf("failed to list host-only interfaces: %w", err)
	}
	return parseHostOnlyInterfaceList(output), nil
}

// parseHostOnlyInterfaceList parses the output of "VBoxManage list hostonlyifs".
//
// Each record begins with a "Name:" line and spans several key/value lines,
// for example:
//
//	Name:            vboxnet0
//	GUID:            786f6276-656e-4074-8000-0a0027000000
//	DHCP:            Disabled
//	IPAddress:       192.168.56.1
//	NetworkMask:     255.255.255.0
//	Status:          Up
//	VBoxNetworkName: HostInterfaceNetworking-vboxnet0
//
// The interface's assigned address is surfaced as the Network's LowerIP: on
// this backend the host takes the lower bound of the range, mirroring how the
// hostonlynet backend hands the host the network's lower bound.
func parseHostOnlyInterfaceList(output string) []Network {
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
			flush()
			current = &Network{
				Backend:       BackendHostOnlyIf,
				Name:          value,
				HostInterface: value,
			}
		case "GUID":
			if current != nil {
				current.GUID = value
			}
		case "IPAddress":
			if current != nil {
				current.LowerIP = value
			}
		case "NetworkMask":
			if current != nil {
				current.NetworkMask = value
			}
		case "Status":
			if current != nil {
				current.DHCP = strings.EqualFold(value, "Up")
			}
		case "VBoxNetworkName":
			if current != nil {
				current.VBoxNetworkName = value
			}
		}
	}

	flush()
	return networks
}
