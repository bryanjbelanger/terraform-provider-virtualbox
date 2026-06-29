//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name virtualbox

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/bryanbelanger/virtualbox",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), New("dev"), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
