package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConditionalBindingsDataSourceModel struct {
	Project  types.String `tfsdk:"project"`
	Role     types.String `tfsdk:"role"`
	Bindings types.List   `tfsdk:"bindings"`
	Id       types.String `tfsdk:"id"`
}

type ConditionalBindingModel struct {
	Role      types.String `tfsdk:"role"`
	Members   types.List   `tfsdk:"members"`
	Condition types.Object `tfsdk:"condition"`
}

var ConditionalBindingAttrTypes = map[string]attr.Type{
	"role":      types.StringType,
	"members":   types.ListType{ElemType: types.StringType},
	"condition": types.ObjectType{AttrTypes: ConditionalBindingConditionAttrTypes},
}

type ConditionalBindingConditionModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Expression  types.String `tfsdk:"expression"`
}

var ConditionalBindingConditionAttrTypes = map[string]attr.Type{
	"title":       types.StringType,
	"description": types.StringType,
	"expression":  types.StringType,
}
