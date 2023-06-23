package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

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
	return &ImageSyncResource{}
}

type ImageSyncResource struct{}

type ImageSyncResourceModel struct {
	Source       types.String `tfsdk:"source"`
	Destination  types.String `tfsdk:"destination"`
	SourceDigest types.String `tfsdk:"source_digest"`
	Id           types.String `tfsdk:"id"`
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
					stringplanmodifier.RequiresReplace(),
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
	var data ImageSyncResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		resp.Diagnostics.AddError("failed to create google authenticator", err.Error())
		return
	}

	src := data.Source.ValueString()
	dest := data.Destination.ValueString()

	// getting source image
	srcImg, exists, srcDigest, err := getRemoteImage(src, authn.Anonymous)
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

	if err := remote.Write(destRef, srcImg, remote.WithAuth(googleAuth)); err != nil {
		resp.Diagnostics.AddError("failed to write image", err.Error())
		return
	}

	// get the image from registry to verify it was properly written
	destImg, exists, destDigest, err := getRemoteImage(dest, googleAuth)
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

	imgID, err := imageID(data.Destination.ValueString(), destImg)
	if err != nil {
		resp.Diagnostics.AddError("failed to get image ID", err.Error())
		return
	}
	data.Id = types.StringValue(imgID)
	data.SourceDigest = types.StringValue(srcDigest)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ImageSyncResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		resp.Diagnostics.AddError("failed to create google authenticator", err.Error())
		return
	}

	src := data.Source.ValueString()
	dest := data.Destination.ValueString()

	_, exists, srcDigest, err := getRemoteImage(src, authn.Anonymous)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get source image", err.Error())
		return
	case !exists:
		resp.Diagnostics.AddError("source image does not exist", fmt.Sprintf("unable to locate source image at '%s'", src))
		return
	}

	destImg, exists, destDigest, err := getRemoteImage(dest, googleAuth)
	switch {
	case err != nil:
		resp.Diagnostics.AddError("failed to get destination image", err.Error())
		return
	case !exists:
		resp.State.RemoveResource(ctx)
		return
	}

	imgID, err := imageID(data.Destination.ValueString(), destImg)
	if err != nil {
		resp.Diagnostics.AddError("failed to get image ID", err.Error())
		return
	}
	data.Id = types.StringValue(imgID)

	if srcDigest != destDigest {
		data.SourceDigest = types.StringValue(destDigest)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ImageSyncResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ImageSyncResourceModel

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

		if imageID.String() == digestFromReference(data.Id.ValueString()) {
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

// getRemoteImage returns a remote image if it exists along with a string
// representation of it's digest
func getRemoteImage(url string, auth authn.Authenticator) (v1.Image, bool, string, error) {
	urlRef, err := name.ParseReference(url, name.WeakValidation)
	if err != nil {
		return empty.Image, false, "", err
	}

	i, err := remote.Image(urlRef, remote.WithAuth(auth))
	if err != nil {
		if tErr, ok := (err).(*transport.Error); ok && tErr.StatusCode == 404 {
			return empty.Image, false, "", nil
		}
		return empty.Image, false, "", err
	}

	imgDigest, err := i.Digest()
	if err != nil {
		return empty.Image, true, "", err
	}

	return i, true, imgDigest.String(), nil
}

// imageID is the fully qualified URL to the image, with any tags replaced with
// the sha256 digest instead
func imageID(url string, img v1.Image) (string, error) {
	if hasSHA, _ := regexp.MatchString("(.+)(@sha256:)([a-f0-9]{64})", url); hasSHA {
		return url, nil
	}

	// Trim any tags from the url
	trimTo := strings.LastIndex(url, ":")
	if trimTo != -1 && trimTo < len(url) {
		url = url[:trimTo]
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return url + "@" + digest.String(), nil
}

// digestFromReference strips all content preceding the digest for the given,
// fully qualified, reference. If no digest is present, the resulting string
// will be empty
func digestFromReference(ref string) string {
	at := strings.LastIndex(ref, "@")
	if at == -1 {
		return ""
	}

	if at+1 >= len(ref) {
		return ""
	}

	return ref[at+1:]
}
