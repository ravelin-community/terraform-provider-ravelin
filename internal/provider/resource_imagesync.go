package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/image"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/planmodifiers"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ImageSyncResource{}
var _ resource.ResourceWithImportState = &ImageSyncResource{}

func NewImageSyncResource() resource.Resource {
	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		panic(fmt.Errorf("failed to create google authenticator, %v", err.Error()))
	}

	return &ImageSyncResource{
		auth: googleAuth,
	}
}

type ImageSyncResource struct {
	auth authn.Authenticator
}

func (r *ImageSyncResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_imagesync"
}

func (r *ImageSyncResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"source": schema.StringAttribute{
				MarkdownDescription: "Repository reference to the source image you wish to mirror",
				Required:            true,
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "Repository reference to the source image that you wish to mirror",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_digest": schema.StringAttribute{
				MarkdownDescription: "Digest of the source image; should always match the digest of the destination image",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImageDigestModifier(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Repository reference for the mirrored image in the destination, referenced by the image digest, rather than the tag.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		MarkdownDescription: "Resource to import and sync images from public container registries into your own" +
			"Google Container Registries (GCR) or Google Artifact Registries (GAR).",
	}
}

func (r *ImageSyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.ImageSyncResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	src := data.Source.ValueString()
	dest := data.Destination.ValueString()

	// getting source image
	srcImg, exists, srcDigest, err := image.GetRemoteImage(src, authn.Anonymous)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get remote image", err.Error())
		return
	case !exists:
		resp.Diagnostics.AddError("source image does not exist", src)
		return
	}

	destRef, err := name.ParseReference(dest, name.WeakValidation)
	if err != nil {
		resp.Diagnostics.AddError("failed to parse destination reference", err.Error())
		return
	}

	if err := remote.Write(destRef, srcImg, remote.WithAuth(r.auth)); err != nil {
		resp.Diagnostics.AddError("failed to write image", err.Error())
		return
	}

	// get the image from registry to verify it was properly written
	destImg, exists, destDigest, err := image.GetRemoteImage(dest, r.auth)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get registry image", err.Error())
		return
	case !exists:
		resp.Diagnostics.AddError("image did not get synched properly", dest)
		return
	case srcDigest != destDigest:
		resp.Diagnostics.AddError("image did not get synched properly", fmt.Sprintf("source and destination digests do not match: %s != %s", srcDigest, destDigest))
	}

	imgID, err := image.ImageID(data.Destination.ValueString(), destImg)
	if err != nil {
		resp.Diagnostics.AddError("failed to get image ID", err.Error())
		return
	}
	data.Id = types.StringValue(imgID)
	data.SourceDigest = types.StringValue(srcDigest)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.ImageSyncResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dest := data.Destination.ValueString()

	destImg, exists, _, err := image.GetRemoteImage(dest, r.auth)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get destination image", err.Error())
		return
	case !exists:
		resp.State.RemoveResource(ctx)
		return
	}

	imgID, err := image.ImageID(data.Destination.ValueString(), destImg)
	if err != nil {
		resp.Diagnostics.AddError("failed to get image ID", err.Error())
		return
	}
	data.Id = types.StringValue(imgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config models.ImageSyncResourceModel
	var state models.ImageSyncResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// updates can only be triggered by 'source' changes that don't change the
	// 'source_digest', suggesting a new registry/tag, but not a new underlying
	// image. No actual update is necessary, only a state update is needed.
	state.Source = config.Source
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ImageSyncResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.ImageSyncResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dest := data.Destination.ValueString()
	destRef, err := name.ParseReference(dest, name.WeakValidation)
	if err != nil {
		resp.Diagnostics.AddError("failed to parse destination reference", err.Error())
		return
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		resp.Diagnostics.AddError("failed to create google authenticator", err.Error())
		return
	}
	authOpt := remote.WithAuth(googleAuth)

	// delete this tag. Perform this regardless of if other tags exist
	if err := remote.Delete(destRef, authOpt); err != nil {
		resp.Diagnostics.AddError("failed to delete image", err.Error())
		return
	}

	// check through all available tags to see if there are any more images
	// referencing these blobs
	tags, err := remote.List(destRef.Context(), authOpt)
	if err != nil {
		if strings.Contains(err.Error(), "METHOD_UNKNOWN") {
			resp.Diagnostics.AddWarning("listing unsupported", "registry does not support listing images, cannot verify if blobs are in use")
			return
		}
		resp.Diagnostics.AddError("failed to list images", err.Error())
		return
	}

	for _, t := range tags {
		imgRef, err := name.ParseReference(destRef.Context().String()+":"+t, name.WeakValidation)
		if err != nil {
			resp.Diagnostics.AddError("failed to parse image reference", err.Error())
			return
		}

		i, err := remote.Image(imgRef, authOpt)
		if err != nil {
			resp.Diagnostics.AddError("failed to get image", err.Error())
			return
		}

		imageID, err := i.Digest()
		if err != nil {
			resp.Diagnostics.AddError("failed to get image digest", err.Error())
			return
		}

		if imageID.String() == image.DigestFromReference(data.Id.ValueString()) {
			// another image is using the same layers as we are, do not delete these
			// layers!
			return
		}
	}

	// No other tag references these layers, we're free to delete
	idRef, err := name.ParseReference(data.Id.ValueString(), name.WeakValidation)
	if err != nil {
		resp.Diagnostics.AddError("failed to parse image reference", err.Error())
		return
	}

	remote.Delete(idRef, authOpt)
}

func (r *ImageSyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	imageParts := strings.Split(req.ID, ",")

	if len(imageParts) != 2 || imageParts[0] == "" || imageParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: <source>,<destination>. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source"), imageParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("destination"), imageParts[1])...)
}
