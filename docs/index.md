# VirtualBox Provider

The VirtualBox provider is used to manage VirtualBox virtual machines, networks, and shared folders. The provider needs to be configured with the path to the `VBoxManage` executable, which is typically found in the VirtualBox installation directory.

## Example Usage

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
  # Path to VBoxManage (optional, defaults to "VBoxManage" in PATH)
  vboxmanage_path = "VBoxManage"
}
```

## Configuration

| Attribute | Type | Default | Description |
| ----------- | ------ | --------- | ------------- |
| `vboxmanage_path` | string | `"VBoxManage"` | Path to the VBoxManage executable |

## Resources

- [virtualbox_vm](./resources/virtualbox_vm.md)
- [virtualbox_network](./resources/virtualbox_network.md)
- [virtualbox_shared_folder](./resources/virtualbox_shared_folder.md)

## Data Sources

- [virtualbox_vm](./resources/virtualbox_vm.md)
