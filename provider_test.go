package main

import (
	"testing"
)

func TestProvider(t *testing.T) {
	// The terraform-plugin-framework does not have the same InternalValidate method
	// as sdk/v2 on the provider itself. 
	// Basic structure is handled by the framework internals.
	p := New("test")()
	if p == nil {
		t.Fatal("expected provider to be returned")
	}
}
