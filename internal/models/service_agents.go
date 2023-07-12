package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type ServiceAgentsDataSourceModel struct {
	Project            types.String `tfsdk:"project"`
	ServiceAgentPolicy types.String `tfsdk:"service_agent_policy"`
	Id                 types.String `tfsdk:"id"`
}
