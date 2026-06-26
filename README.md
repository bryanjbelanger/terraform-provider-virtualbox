# Terraform Provider VirtualBox

A [Terraform](https://www.terraform.io) / [OpenTofu](https://opentofu.org) provider for managing [VirtualBox](https://www.virtualbox.org) virtual machines.

## Requirements

- [Go](https://golang.org/doc/install) 1.22+
- [VirtualBox](https://www.virtualbox.org/wiki/Downloads) (with `VBoxManage` CLI in PATH)
- [Terraform](https://www.terraform.io/downloads) 1.x or [OpenTofu](https://opentofu.org/downloads) 1.6+

## Building the Provider

```bash
git clone https://github.com/bryanbelanger/terraform-provider-virtualbox
cd terraform-provider-virtualbox
make build
```

## Installing Locally

```bash
make install
```

This installs the provider to `~/.terraform.d/plugins/registry.terraform.io/bryanbelanger/virtualbox/0.1.0/`.

## Using the Provider

### Provider Configuration

```hcl
terraform {
  required_providers {
    virtualbox = {
      source  = "bryanbelanger/virtualbox"
      version = "~> 0.1.0"
    }
  }
}

provider "virtualbox" {
  # Optional: specify a custom path to VBoxManage
  # vboxmanage_path = "/usr/local/bin/VBoxManage"
}
```

### Example Usage

```hcl
# Create a VM
resource "virtualbox_vm" "web_server" {
  name    = "web-server"
  os_type = "Ubuntu_64"
  memory  = 2048
  cpus    = 2
  vram    = 16
}

# Create a host-only network
resource "virtualbox_network" "internal" {
  name         = "internal-net"
  network_cidr = "192.168.56.0/24"
  dhcp         = true
}

# Add a shared folder
resource "virtualbox_shared_folder" "data" {
  vm_name   = virtualbox_vm.web_server.name
  name      = "data"
  host_path = "/host/data"
  writable  = true
  automount = true
}

# Data source to read existing VM info
data "virtualbox_vm" "existing" {
  name = virtualbox_vm.web_server.name
}

output "vm_uuid" {
  value = data.virtualbox_vm.existing.uuid
}
```

## Resources

### `virtualbox_vm`

Manages a VirtualBox virtual machine.

| Attribute | Type | Required | Description |
| ----------- | ------ | ---------- | ------------- |
| `name` | string | yes | VM name |
| `os_type` | string | no | Guest OS type (default: `Other`) |
| `memory` | int | no | RAM in MB (default: `1024`) |
| `cpus` | int | no | Number of CPUs (default: `1`) |
| `vram` | int | no | Video RAM in MB (default: `8`) |
| `status` | string | computed | Current VM status |
| `uuid` | string | computed | VM UUID |

### `virtualbox_network`

Manages a host-only network.

| Attribute | Type | Required | Description |
| ----------- | ------ | ---------- | ------------- |
| `name` | string | yes | Network name |
| `network_cidr` | string | no | CIDR notation (e.g., `192.168.56.0/24`) |
| `dhcp` | bool | no | Enable DHCP (default: `true`) |
| `dhcp_lower_ip` | string | no | DHCP range lower bound |
| `dhcp_upper_ip` | string | no | DHCP range upper bound |
| `guid` | string | computed | Network GUID |

### `virtualbox_shared_folder`

Manages a shared folder on a VM.

| Attribute | Type | Required | Description |
| ----------- | ------ | ---------- | ------------- |
| `vm_name` | string | yes | Target VM name |
| `name` | string | yes | Folder name in the guest |
| `host_path` | string | yes | Path on the host system |
| `writable` | bool | no | Mount as writable (default: `true`) |
| `automount` | bool | no | Auto-mount in guest (default: `false`) |

## Data Sources

### virtualbox_vm

Reads information about an existing VM.

| Attribute | Type | Description |
| ----------- | ------ | ------------- |
| `name` | string (required) | VM name to look up |
| `os_type` | string | Guest OS type |
| `memory` | int | RAM in MB |
| `cpus` | int | Number of CPUs |
| `vram` | int | Video RAM |
| `status` | string | Current VM status |
| `uuid` | string | VM UUID |

## Development

Run tests:

```bash
make test
```

Run linter:

```bash
make lint
```

Format code:

```bash
make fmt
```

## License

MIT
