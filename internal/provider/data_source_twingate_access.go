package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iam "github.com/ravelin-community/terraform-provider-ravelin/internal/iam"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
)

type TwingateAccessDataSource struct {
	provider *ravelinProvider
}

func (r *TwingateAccessDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_twingate_access"
}

func (r *TwingateAccessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"iam_path": schema.StringAttribute{
				MarkdownDescription: "Path to the root of the IAM directory containing user and group definitions",
				Required:            true,
			},
			"twingate_access": schema.MapAttribute{
				MarkdownDescription: "Map of users to Twingate access. The key is the user email and the value is an object of Twingate access details.",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"enabled": types.BoolType,
						"admin":   types.BoolType,
					},
				},
			},
			"user_email": schema.StringAttribute{
				MarkdownDescription: "Email of the user to retrieve twingate access for. If not specified, all users access is returned.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *TwingateAccessDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*ravelinProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ravelinProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.provider = provider
}

func (d *TwingateAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.TwingateAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	iamPath := data.Iam_path.ValueString()
	if iamPath == "" {
		resp.Diagnostics.AddError(
			"Missing IAM Path",
			"The `iam_path` attribute is required but was not set. Please provide the path to the IAM directory containing user and group definitions.",
		)
		return
	}

	err, allUserAccess := iam.ExtractUserAccess(iamPath)
	if err != nil {
		resp.Diagnostics.AddError("failed to extract user access", err.Error())
		return
	}

	twingateAccess := make(map[string]models.TwingateAccessModel)
	for _, userAccess := range allUserAccess {
		if userAccess.TwingateAccess.Enabled {

			if !data.UserEmail.IsNull() && data.UserEmail.ValueString() != userAccess.Email {
				continue // Skip if user email does not match
			}

			twingateAccess[userAccess.Email] = models.TwingateAccessModel{
				Enabled: userAccess.TwingateAccess.Enabled,
				Admin:   userAccess.TwingateAccess.Admin,
			}
		}
	}

	dataTwingateAccess, diags := types.MapValueFrom(ctx, types.ObjectType{
		AttrTypes: models.TwingateAccessAttrTypes,
	}, twingateAccess)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.TwingateAccess = dataTwingateAccess

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}
