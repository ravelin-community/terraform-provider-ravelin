package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type AgentIdentitiesDataSourceModel struct {
	OrganizationId types.String `tfsdk:"organization_id"`
	Project        types.String `tfsdk:"project"`
	AgentName      types.String `tfsdk:"agent_name"`
	Location       types.String `tfsdk:"location"`
	Identity       types.String `tfsdk:"identity"`
	Id             types.String `tfsdk:"id"`
}
