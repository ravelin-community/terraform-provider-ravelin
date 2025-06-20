package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
	iam "github.com/ravelin-community/terraform-provider-ravelin/internal/ravelinaccess"
)

type GsudoEscalationsDataSource struct {
	provider *ravelinProvider
}

func (r *GsudoEscalationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gsudo_escalations"
}

func (r *GsudoEscalationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"iam_path": schema.StringAttribute{
				MarkdownDescription: "Path to the root of the IAM directory containing user and group definitions",
				Required:            true,
			},
			"escalations": schema.MapAttribute{
				MarkdownDescription: "Map of projects to escalation roles for each user. The key is the user email and the value is a map of project names to escalation roles.",
				Computed:            true,
				ElementType: types.MapType{
					ElemType: types.ListType{
						ElemType: types.StringType,
					},
				},
			},
			"user_email": schema.StringAttribute{
				MarkdownDescription: "Email of the user to filter escalations for. If not specified, all users' escalations will be returned.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *GsudoEscalationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.GsudoEscalationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	iamPath := data.IamPath.ValueString()
	if iamPath == "" {
		resp.Diagnostics.AddError(
			"missing IAM path",
			"the `iam_path` attribute is required but was not set, please provide the path to the IAM directory containing user and group definitions.",
		)
		return
	}

	err, allUserAccess := iam.ExtractUserAccess(iamPath)
	if err != nil {
		resp.Diagnostics.AddError("failed to extract user access", err.Error())
		return
	}

	allUsersEscalations := make(map[string]basetypes.MapValue, len(allUserAccess))

	for _, userAccess := range allUserAccess {
		userEscalationsMap := make(map[string]basetypes.ListValue, len(userAccess.Gsudo.Escalations))

		for p, r := range userAccess.Gsudo.Escalations {
			escalationRoles := make([]types.String, len(r))
			for i, role := range r {
				escalationRoles[i] = types.StringValue(role)
			}
			listVal, diags := types.ListValueFrom(ctx, types.StringType, escalationRoles)
			resp.Diagnostics.Append(diags...)
			userEscalationsMap[p] = listVal
		}

		mapVal, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, userEscalationsMap)
		resp.Diagnostics.Append(diags...)
		allUsersEscalations[userAccess.Email] = mapVal
	}

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"error processing escalations",
			"an error occurred while processing the user escalations, please check the IAM path and ensure it contains valid user and group definitions.",
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(strconv.FormatInt(time.Now().Unix(), 10)))

	if userEmail := data.UserEmail.ValueString(); userEmail != "" {
		userEscalations := make(map[string]basetypes.MapValue, 1)

		if _, found := allUsersEscalations[userEmail]; found {
			userEscalations[userEmail] = allUsersEscalations[userEmail]
		} else {
			resp.Diagnostics.AddWarning(
				"user email not found",
				fmt.Sprintf("the specified user email '%s' does not have any escalations defined, returning empty escalations.", userEmail),
			)
		}

		resp.State.SetAttribute(ctx, path.Root("escalations"), userEscalations)
		return

	}

	resp.State.SetAttribute(ctx, path.Root("escalations"), allUsersEscalations)
}
