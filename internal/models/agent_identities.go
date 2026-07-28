package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type AgentIdentitiesDataSourceModel struct {
	Project             types.String `tfsdk:"project"`
	Role                types.String `tfsdk:"role"`
	AgentIdentityPolicy types.String `tfsdk:"agent_identity_policy"`
	Id                  types.String `tfsdk:"id"`
}
