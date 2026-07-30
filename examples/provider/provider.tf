terraform {
  required_providers {
    virtualbox = {
      source  = "bryanbelanger/virtualbox"
      version = "~> 0.1.0"
    }
  }
}

provider "virtualbox" {
  # Optional: path to VBoxManage (defaults to "VBoxManage" found in PATH).
  # vboxmanage_path = "/usr/local/bin/VBoxManage"
}
