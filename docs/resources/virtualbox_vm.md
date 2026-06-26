# Resource: virtualbox_vm

Manages a VirtualBox virtual machine.

## Example Usage

```hcl
resource "virtualbox_vm" "web" {
  name    = "web-server"
  os_type = "Ubuntu_64"
  memory  = 2048
  cpus    = 2
  vram    = 16
}
```

## Schema

### Required

- **name** (string) - The name of the virtual machine.

### Optional

- **os_type** (string) - The guest OS type. Default: `"Other"`.
  Common values: `"Ubuntu_64"`, `"Ubuntu"`, `"Debian_64"`, `"Windows10_64"`, `"Windows11_64"`, `"Fedora_64"`, `"RedHat_64"`, `"MacOS"`, `"Other_64"`.
- **memory** (int) - Amount of RAM in MB. Default: `1024`.
- **cpus** (int) - Number of CPU cores. Default: `1`.
- **vram** (int) - Video RAM in MB. Default: `8`.

### Read-Only

- **status** (string) - Current VM status (e.g., `"running"`, `"poweroff"`, `"saved"`).
- **uuid** (string) - The UUID of the virtual machine.

## Import

VMs can be imported using the VM name:

```bash
terraform import virtualbox_vm.web web-server
```

## VirtualBox CLI Reference

The following VBoxManage commands are used by this resource:

- `createvm --name <name> --ostype <os_type> --register`
- `modifyvm <name> --memory <mb> --cpus <n> --vram <mb>`
- `showvminfo <name> --machinereadable`
- `unregistervm <name> --delete`
- `startvm <name> --type headless`
- `controlvm <name> poweroff`
