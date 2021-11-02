package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema:        map[string]*schema.Schema{},
		ConfigureFunc: nil,
		DataSourcesMap: map[string]*schema.Resource{
			"ravelin_service_agents": dataSourceServiceAgents(),
		},
	}
}
