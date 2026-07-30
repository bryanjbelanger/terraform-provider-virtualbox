//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate --provider-name virtualbox

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set at build time via -ldflags "-X main.version=...". It defaults to
// "dev" for local builds.
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/bryanjbelanger/virtualbox",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
