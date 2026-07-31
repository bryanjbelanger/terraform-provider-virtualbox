package virtualbox

import "testing"

func TestNetmaskFromCIDR(t *testing.T) {
	cases := []struct {
		name    string
		cidr    string
		want    string
		wantErr bool
	}{
		{name: "slash24", cidr: "192.168.56.0/24", want: "255.255.255.0"},
		{name: "slash16", cidr: "10.0.0.0/16", want: "255.255.0.0"},
		{name: "empty", cidr: "", want: ""},
		{name: "invalid", cidr: "not-a-cidr", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := netmaskFromCIDR(tc.cidr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tc.cidr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("netmaskFromCIDR(%q) = %q, want %q", tc.cidr, got, tc.want)
			}
		})
	}
}

func TestParseSharedFolders(t *testing.T) {
	output := `name="demo"
SharedFolderNameMachineMapping1="data"
SharedFolderPathMachineMapping1="/host/data"
SharedFolderNameMachineMapping2="code"
SharedFolderPathMachineMapping2="/host/code"
`
	folders := parseSharedFolders(output)
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d: %+v", len(folders), folders)
	}
	// Name and host path must be correlated by their shared numeric suffix.
	if folders[0].Name != "data" || folders[0].HostPath != "/host/data" {
		t.Errorf("folder[0] = %+v, want {data /host/data}", folders[0])
	}
	if folders[1].Name != "code" || folders[1].HostPath != "/host/code" {
		t.Errorf("folder[1] = %+v, want {code /host/code}", folders[1])
	}
}

func TestParseVMInfo(t *testing.T) {
	output := `name="web"
UUID="1234-5678"
ostype="Ubuntu_64"
memory=2048
cpus=2
vram=16
VMState="running"
`
	vm := parseVMInfo(output)
	if vm.Name != "web" || vm.UUID != "1234-5678" || vm.OSType != "Ubuntu_64" {
		t.Errorf("unexpected identity fields: %+v", vm)
	}
	if vm.MemoryMB != 2048 || vm.CPUs != 2 || vm.VRAM != 16 {
		t.Errorf("unexpected hardware fields: %+v", vm)
	}
	if vm.Status != "running" {
		t.Errorf("Status = %q, want running", vm.Status)
	}
}

func TestParseNetworkAdapters(t *testing.T) {
	// Machine-readable showvminfo keys differ per attachment flavour: a
	// hostonlynet network reports nic="hostonlynetwork" with a
	// hostonly-network<N> name, while a legacy host-only interface (the only
	// flavour on Windows hosts) reports nic="hostonly" with hostonlyadapter<N>.
	output := `nic1="hostonlynetwork"
hostonly-network1="talos-net"
nictype1="virtio"
macaddress1="080027AAAAAA"
cableconnected1="on"
nic2="hostonly"
hostonlyadapter2="VirtualBox Host-Only Ethernet Adapter"
nictype2="virtio"
macaddress2="080027BBBBBB"
cableconnected2="on"
nic3="nat"
nictype3="82540EM"
macaddress3="080027CCCCCC"
cableconnected3="on"
nic4="none"
`
	adapters := parseNetworkAdapters(output)
	if len(adapters) != 3 {
		t.Fatalf("expected 3 adapters, got %d: %+v", len(adapters), adapters)
	}

	if adapters[0].Type != "hostonlynet" || adapters[0].NetworkName != "talos-net" {
		t.Errorf("adapter[0] = %+v, want hostonlynet/talos-net", adapters[0])
	}
	if adapters[1].Type != "hostonly" || adapters[1].NetworkName != "VirtualBox Host-Only Ethernet Adapter" {
		t.Errorf("adapter[1] = %+v, want hostonly/VirtualBox Host-Only Ethernet Adapter", adapters[1])
	}
	if adapters[2].Type != "nat" || adapters[2].NetworkName != "" {
		t.Errorf("adapter[2] = %+v, want nat with no network name", adapters[2])
	}
	if adapters[1].MACAddress != "080027BBBBBB" || !adapters[1].CableConnected {
		t.Errorf("adapter[1] MAC/cable = %+v", adapters[1])
	}
}

func TestParseNetworkList(t *testing.T) {
	// Real VBoxManage 7.x output: note the blank line *within* each record
	// (between GUID and State), and two records back to back.
	output := `Name:            net-a
GUID:            aaaa-bbbb

State:           Enabled
NetworkMask:     255.255.255.0
LowerIP:         192.168.56.100
UpperIP:         192.168.56.200
VBoxNetworkName: hostonly-net-a


Name:            net-b
GUID:            cccc-dddd

State:           Disabled
NetworkMask:     255.255.0.0
LowerIP:         10.0.0.10
UpperIP:         10.0.255.254
VBoxNetworkName: hostonly-net-b

`
	networks := parseNetworkList(output)
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d: %+v", len(networks), networks)
	}

	a := networks[0]
	if a.Name != "net-a" || a.GUID != "aaaa-bbbb" || !a.DHCP || a.NetworkMask != "255.255.255.0" ||
		a.LowerIP != "192.168.56.100" || a.UpperIP != "192.168.56.200" {
		t.Errorf("net-a parsed incorrectly: %+v", a)
	}

	b := networks[1]
	if b.Name != "net-b" || b.GUID != "cccc-dddd" || b.DHCP || b.LowerIP != "10.0.0.10" {
		t.Errorf("net-b parsed incorrectly: %+v", b)
	}
}

func TestDeriveDHCPRange(t *testing.T) {
	cases := []struct {
		cidr, lower, upper string
	}{
		{"192.168.56.0/24", "192.168.56.1", "192.168.56.254"},
		{"10.0.0.0/16", "10.0.0.1", "10.0.255.254"},
	}
	for _, tc := range cases {
		lower, upper, err := DeriveDHCPRange(tc.cidr)
		if err != nil {
			t.Fatalf("DeriveDHCPRange(%q) error: %v", tc.cidr, err)
		}
		if lower != tc.lower || upper != tc.upper {
			t.Errorf("DeriveDHCPRange(%q) = %s..%s, want %s..%s", tc.cidr, lower, upper, tc.lower, tc.upper)
		}
	}
}
