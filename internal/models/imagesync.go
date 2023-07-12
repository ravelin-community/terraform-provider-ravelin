package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ImageSyncResourceModel struct {
	Source       types.String `tfsdk:"source"`
	Destination  types.String `tfsdk:"destination"`
	SourceDigest types.String `tfsdk:"source_digest"`
	Id           types.String `tfsdk:"id"`
}
