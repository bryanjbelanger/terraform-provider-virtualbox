package main

import (

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProviderFactories are used to instantiate a provider during acceptance testing.
var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"virtualbox": providerserver.NewProtocol6WithError(New("test")()),
}


