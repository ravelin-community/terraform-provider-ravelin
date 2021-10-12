package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	provider "github.com/unravelin/terraform-provider-ravelin/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: provider.Provider})
}
