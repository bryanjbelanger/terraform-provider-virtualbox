package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderHasResources(t *testing.T) {
	provider := Provider()

	expectedResources := []string{
		"virtualbox_vm",
		"virtualbox_network",
		"virtualbox_shared_folder",
	}

	for _, name := range expectedResources {
		if _, ok := provider.ResourcesMap[name]; !ok {
			t.Errorf("Expected resource %s not found in provider", name)
		}
	}
}

func TestProviderHasDataSources(t *testing.T) {
	provider := Provider()

	expectedDataSources := []string{
		"virtualbox_vm",
	}

	for _, name := range expectedDataSources {
		if _, ok := provider.DataSourcesMap[name]; !ok {
			t.Errorf("Expected data source %s not found in provider", name)
		}
	}
}

func TestProviderConfigure(t *testing.T) {
	resource := schema.TestResourceDataRaw(t, Provider().Schema, map[string]interface{}{
		"vboxmanage_path": "VBoxManage",
	})

	result, err := providerConfigure(resource)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if result == nil {
		t.Fatal("expected client to be returned")
	}
}
