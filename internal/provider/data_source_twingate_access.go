package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
	iam "github.com/ravelin-community/terraform-provider-ravelin/internal/ravelinaccess"
)

type TwingateAccessDataSource struct{}

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

func (d *TwingateAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.TwingateAccessDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	iamPath := data.IamPath.ValueString()
	if iamPath == "" {
		resp.Diagnostics.AddError(
			"missing IAM path",
			"the `iam_path` attribute is required but was not set, please provide the path to the IAM directory containing user and group definitions.",
		)
		return
	}

	userFiles, err := iam.GetUserFiles(iamPath)
	if err != nil {
		resp.Diagnostics.AddError("failed to retrieve user files", err.Error())
		return
	}

	allUserAccess := make([]iam.RavelinAccess, 0, len(userFiles))

	for _, userFile := range userFiles {
		userAccess, err := iam.ExtractRavelinAccess(fmt.Sprintf("%s/users/%s", iamPath, userFile))
		if err != nil {
			resp.Diagnostics.AddError("failed to extract user access", err.Error())
			return
		}

		err = userAccess.InheritTwingateAccess()
		if err != nil {
			resp.Diagnostics.AddError("failed to inherit twingate access", err.Error())
			return
		}

		// if after inheritance the user access is not set, set it to false
		if userAccess.Twingate.Enabled == nil {
			*userAccess.Twingate.Enabled = false
		}
		if userAccess.Twingate.Admin == nil {
			*userAccess.Twingate.Admin = false
		}

		allUserAccess = append(allUserAccess, userAccess)
	}

	twingateAccess := make(map[string]models.TwingateAccessModel)
	for _, userAccess := range allUserAccess {
		if *userAccess.Twingate.Enabled {

			if !data.UserEmail.IsNull() && data.UserEmail.ValueString() != userAccess.Email {
				continue // Skip if user email does not match
			}

			twingateAccess[userAccess.Email] = models.TwingateAccessModel{
				Enabled: *userAccess.Twingate.Enabled,
				Admin:   *userAccess.Twingate.Admin,
			}
		}
	}

	dataTwingateAccess, diags := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: models.TwingateAccessAttrTypes}, twingateAccess)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.TwingateAccess = dataTwingateAccess
	data.Id = types.StringValue(strconv.FormatInt(time.Now().Unix(), 10))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
