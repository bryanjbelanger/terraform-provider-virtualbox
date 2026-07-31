package virtualbox

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// maxNICs is the number of network adapters a VirtualBox VM supports.
const maxNICs = 8

// NetworkAdapter describes a single VM network adapter (nic<N>).
type NetworkAdapter struct {
	// Type is the attachment type: nat, natnetwork, hostonlynet, bridged, intnet, or null.
	Type string
	// NetworkName is the backing network/interface, interpreted per Type:
	// hostonlynet -> host-only network name, bridged -> host interface,
	// intnet -> internal network name, natnetwork -> NAT network name.
	// Ignored for nat and null.
	NetworkName string
	// NICType is the emulated hardware model (e.g. "virtio", "82540EM").
	NICType string
	// MACAddress is the adapter's MAC as reported by VirtualBox (12 hex digits).
	MACAddress string
	// CableConnected reports whether the virtual cable is plugged in.
	CableConnected bool
}

// ConfigureNetworkAdapters applies adapters to nic1..nicN and disables any
// remaining adapters (nic(N+1)..nic8) so the VM's networking exactly matches the
// declared configuration. Callers should only invoke it when at least one adapter
// is declared, so that VMs without managed adapters keep VirtualBox's defaults.
func (c *Client) ConfigureNetworkAdapters(ctx context.Context, vmName string, adapters []NetworkAdapter) error {
	for i := 0; i < maxNICs; i++ {
		n := strconv.Itoa(i + 1)

		if i >= len(adapters) {
			if _, err := c.RunContext(ctx, "modifyvm", vmName, "--nic"+n, "none"); err != nil {
				return fmt.Errorf("failed to disable nic%s: %w", n, err)
			}
			continue
		}

		a := adapters[i]
		args := []string{"modifyvm", vmName, "--nic" + n, a.Type}
		switch a.Type {
		case "hostonlynet":
			if a.NetworkName != "" {
				args = append(args, "--host-only-net"+n, a.NetworkName)
			}
		case "bridged":
			if a.NetworkName != "" {
				args = append(args, "--bridgeadapter"+n, a.NetworkName)
			}
		case "intnet":
			if a.NetworkName != "" {
				args = append(args, "--intnet"+n, a.NetworkName)
			}
		case "natnetwork":
			if a.NetworkName != "" {
				args = append(args, "--nat-network"+n, a.NetworkName)
			}
		}
		if a.NICType != "" {
			args = append(args, "--nictype"+n, a.NICType)
		}
		if a.MACAddress != "" {
			// VirtualBox stores MACs uppercase without separators; normalise the
			// input so Read never sees a case-only diff.
			args = append(args, "--macaddress"+n, strings.ToUpper(a.MACAddress))
		}
		cable := "on"
		if !a.CableConnected {
			cable = "off"
		}
		args = append(args, "--cableconnected"+n, cable)

		if _, err := c.RunContext(ctx, args...); err != nil {
			return fmt.Errorf("failed to configure nic%s: %w", n, err)
		}
	}
	return nil
}

// nicIndex returns the 1-based adapter index for a machine-readable key of the
// form "<prefix><N>" (e.g. "nic2"), or ok=false if the key does not match.
func nicIndex(key, prefix string) (int, bool) {
	if !strings.HasPrefix(key, prefix) {
		return 0, false
	}
	i, err := strconv.Atoi(key[len(prefix):])
	if err != nil {
		return 0, false
	}
	return i, true
}

// normalizeAdapterType maps VirtualBox's machine-readable attachment values to
// the flag/config form. Notably showvminfo reports a host-only network as
// "hostonlynetwork" while the modifyvm flag is "hostonlynet".
func normalizeAdapterType(t string) string {
	if t == "hostonlynetwork" {
		return "hostonlynet"
	}
	return t
}

// parseNetworkAdapters extracts the configured adapters (in nic1..nic8 order,
// skipping disabled ones) from "showvminfo --machinereadable" output.
func parseNetworkAdapters(output string) []NetworkAdapter {
	types := map[int]string{}
	names := map[int]string{}
	hwTypes := map[int]string{}
	macs := map[int]string{}
	cables := map[int]bool{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.Trim(parts[1], "\"")

		switch {
		case matchIdx(key, "nictype", hwTypes, val):
		case matchIdx(key, "nic", types, val):
		case matchIdx(key, "hostonly-network", names, val):
		case matchIdx(key, "bridgeadapter", names, val):
		case matchIdx(key, "intnet", names, val):
		case matchIdx(key, "nat-network", names, val):
		case matchIdx(key, "macaddress", macs, val):
		case key != "" && strings.HasPrefix(key, "cableconnected"):
			if i, ok := nicIndex(key, "cableconnected"); ok {
				cables[i] = val == "on"
			}
		}
	}

	var adapters []NetworkAdapter
	for i := 1; i <= maxNICs; i++ {
		t, ok := types[i]
		if !ok || t == "none" || t == "null" {
			continue
		}
		adapters = append(adapters, NetworkAdapter{
			Type:           normalizeAdapterType(t),
			NetworkName:    names[i],
			NICType:        hwTypes[i],
			MACAddress:     macs[i],
			CableConnected: cables[i],
		})
	}
	return adapters
}

// matchIdx stores val under the adapter index parsed from key when key is
// exactly "<prefix><N>", returning whether it matched.
func matchIdx(key, prefix string, dst map[int]string, val string) bool {
	i, ok := nicIndex(key, prefix)
	if !ok {
		return false
	}
	dst[i] = val
	return true
}
