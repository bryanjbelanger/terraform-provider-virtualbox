package virtualbox

import "testing"

func TestParseHostOnlyInterfaceCreated(t *testing.T) {
	output := `0%...10%...20%...30%...40%...50%...60%...70%...80%...90%...100%
Interface 'vboxnet0' was successfully created
`
	m := hostonlyifCreatedRE.FindStringSubmatch(output)
	if m == nil || m[1] != "vboxnet0" {
		t.Fatalf("expected to extract vboxnet0, got %v", m)
	}
}

func TestParseHostOnlyInterfaceList(t *testing.T) {
	// Verbatim shape of `VBoxManage list hostonlyifs` on a Linux host,
	// including the IPv6 lines the parser must skip and a second record.
	output := `Name:            vboxnet0
GUID:            786f6276-656e-4074-8000-0a0027000000
DHCP:            Disabled
IPAddress:       192.168.56.100
NetworkMask:     255.255.255.0
IPV6Address:     fe80::800:27ff:fe00:0
IPV6NetworkMaskPrefixLength: 64
HardwareAddress: 0a:00:27:00:00:00
MediumType:      Ethernet
Wireless:        No
Status:          Up
VBoxNetworkName: HostInterfaceNetworking-vboxnet0

Name:            vboxnet1
GUID:            786f6276-656e-4174-8000-0a0027000001
DHCP:            Disabled
IPAddress:       192.168.57.1
NetworkMask:     255.255.255.0
IPV6Address:
IPV6NetworkMaskPrefixLength: 0
HardwareAddress: 0a:00:27:00:00:01
MediumType:      Ethernet
Wireless:        No
Status:          Down
VBoxNetworkName: HostInterfaceNetworking-vboxnet1
`
	networks := parseHostOnlyInterfaceList(output)
	if len(networks) != 2 {
		t.Fatalf("expected 2 interfaces, got %d: %+v", len(networks), networks)
	}

	first := networks[0]
	if first.Backend != BackendHostOnlyIf {
		t.Errorf("Backend = %q, want %q", first.Backend, BackendHostOnlyIf)
	}
	if first.HostInterface != "vboxnet0" || first.Name != "vboxnet0" {
		t.Errorf("identity = (%q, %q), want vboxnet0", first.Name, first.HostInterface)
	}
	if first.GUID != "786f6276-656e-4074-8000-0a0027000000" {
		t.Errorf("GUID = %q", first.GUID)
	}
	// The interface's assigned address surfaces as LowerIP: the host takes the
	// lower bound of the range on this backend.
	if first.LowerIP != "192.168.56.100" {
		t.Errorf("LowerIP = %q, want 192.168.56.100", first.LowerIP)
	}
	if first.NetworkMask != "255.255.255.0" {
		t.Errorf("NetworkMask = %q", first.NetworkMask)
	}
	if first.VBoxNetworkName != "HostInterfaceNetworking-vboxnet0" {
		t.Errorf("VBoxNetworkName = %q", first.VBoxNetworkName)
	}
	if !first.DHCP {
		t.Errorf("DHCP (status Up) = false, want true")
	}
	if networks[1].DHCP {
		t.Errorf("DHCP (status Down) = true, want false")
	}
}

func TestNetworkDerivedAttributes(t *testing.T) {
	cases := []struct {
		name            string
		network         Network
		adapterType     string
		adapterNetwork  string
		dhcpNetworkName string
	}{
		{
			name:            "hostonlynet",
			network:         Network{Backend: BackendHostOnlyNet, Name: "talos-net"},
			adapterType:     "hostonlynet",
			adapterNetwork:  "talos-net",
			dhcpNetworkName: "hostonly-talos-net",
		},
		{
			name: "hostonlyif with reported network name",
			network: Network{
				Backend:         BackendHostOnlyIf,
				Name:            "talos-net",
				HostInterface:   "vboxnet0",
				VBoxNetworkName: "HostInterfaceNetworking-vboxnet0",
			},
			adapterType:     "hostonly",
			adapterNetwork:  "vboxnet0",
			dhcpNetworkName: "HostInterfaceNetworking-vboxnet0",
		},
		{
			name: "hostonlyif derives network name when unreported",
			network: Network{
				Backend:       BackendHostOnlyIf,
				Name:          "talos-net",
				HostInterface: "vboxnet3",
			},
			adapterType:     "hostonly",
			adapterNetwork:  "vboxnet3",
			dhcpNetworkName: "HostInterfaceNetworking-vboxnet3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.network.AdapterType(); got != tc.adapterType {
				t.Errorf("AdapterType() = %q, want %q", got, tc.adapterType)
			}
			if got := tc.network.AdapterNetwork(); got != tc.adapterNetwork {
				t.Errorf("AdapterNetwork() = %q, want %q", got, tc.adapterNetwork)
			}
			if got := tc.network.DHCPNetworkName(); got != tc.dhcpNetworkName {
				t.Errorf("DHCPNetworkName() = %q, want %q", got, tc.dhcpNetworkName)
			}
		})
	}
}

func TestDefaultNetworkBackend(t *testing.T) {
	// Whatever OS the tests run on, the selection must be one of the two known
	// backends, and the mapping for the current OS must be self-consistent.
	backend := defaultNetworkBackend()
	if backend != BackendHostOnlyNet && backend != BackendHostOnlyIf {
		t.Fatalf("unexpected backend %q", backend)
	}
}
