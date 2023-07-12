package planmodifiers

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/image"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
)

// ImageDigestModifier returns an attribute plan modifier that checks the digest
// of the source image and adds it to the plan. If the source digest doesn't
// match with what we have in the state, it will trigger a replacement.
func ImageDigestModifier() planmodifier.String {
	return ImageDigest{}
}

type ImageDigest struct{}

func (r ImageDigest) Description(ctx context.Context) string {
	return "If the value of the source image digest changes, Terraform will destroy and recreate the resource."
}

func (r ImageDigest) MarkdownDescription(ctx context.Context) string {
	return r.Description(ctx)
}

func (r ImageDigest) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// check we have a config value to work with or if we're not destroying the resource
	if req.ConfigValue.IsUnknown() || req.Plan.Raw.IsNull() {
		return
	}

	var data models.ImageSyncResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	source := data.Source.ValueString()

	// let's get the source image digest
	_, exists, srcDigest, err := image.GetRemoteImage(data.Source.ValueString(), authn.Anonymous)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get remote image", err.Error())
		return
	case !exists:
		resp.Diagnostics.AddError("source image does not exist", source)
		return
	}

	resp.PlanValue = basetypes.NewStringValue(srcDigest)

	if resp.PlanValue.Equal(req.StateValue) {
		// if the plan and the state are in agreement, this attribute
		// isn't changing, don't require replace
		return
	}

	resp.RequiresReplace = true
}
