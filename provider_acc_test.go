package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"virtualbox": func() (*schema.Provider, error) {
		return Provider(), nil
	},
}
