package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TwingateAccessDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	TwingateAccess types.Map    `tfsdk:"twingate_access"` // Map of users to Twingate access
	Iam_path       types.String `tfsdk:"iam_path"`        // Path to the root of the IAM directory containing user and group definitions
	UserEmail      types.String `tfsdk:"user_email"`      // Email of the user to retrieve Twingate access for
}

type TwingateAccessModel struct {
	Enabled bool `tfsdk:"enabled"` // whether the user has Twingate access
	Admin   bool `tfsdk:"admin"`   // whether the user has Twingate admin access
}

var TwingateAccessAttrTypes = map[string]attr.Type{
	"enabled": types.BoolType,
	"admin":   types.BoolType,
}
