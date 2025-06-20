package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GsudoEscalationsDataSourceModel struct {
	Escalations types.Map    `tfsdk:"escalations"`
	Id          types.String `tfsdk:"id"`
	IamPath     types.String `tfsdk:"iam_path"`
	UserEmail   types.String `tfsdk:"user_email"` // optional filter for user email
}
