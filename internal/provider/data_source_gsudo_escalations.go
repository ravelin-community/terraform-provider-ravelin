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

type GsudoEscalationsDataSource struct{}

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
			"access_policies": schema.MapAttribute{
				MarkdownDescription: "Indicates if the user has access to switch access context policies from enforce to dry-run mode.",
				Computed:            true,
				ElementType:         types.BoolType,
			},
			"user_email": schema.StringAttribute{
				MarkdownDescription: "Email of the user to filter escalations for. If not specified, all users' escalations will be returned.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		MarkdownDescription: "Get all configured gsudo escalations for ravelin internal users.",
	}
}

func (d *GsudoEscalationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var rData models.GsudoEscalationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &rData)...)
	iamPath := rData.IamPath.ValueString()
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

		err = userAccess.InheritGsudoAccess()
		if err != nil {
			resp.Diagnostics.AddError("failed to inherit gsudo access", err.Error())
			return
		}

		allUserAccess = append(allUserAccess, userAccess)
	}

	allEscalations := convertEscalationsToMap(ctx, allUserAccess, resp)
	accessPolicies := convertAccessPoliciesToMap(allUserAccess)

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"error processing escalations",
			"an error occurred while processing the user escalations, please check the IAM path and ensure it contains valid user and group definitions.",
		)
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(strconv.FormatInt(time.Now().Unix(), 10)))

	if emailFilter := rData.UserEmail.ValueString(); emailFilter != "" {
		userEscalations := make(map[string]basetypes.MapValue, 1)
		userAccessPolicies := make(map[string]types.Bool, 1)

		if accessPolicy, found := accessPolicies[emailFilter]; found {
			userAccessPolicies[emailFilter] = accessPolicy
			if escalation, foundEscalations := allEscalations[emailFilter]; foundEscalations {
				userEscalations[emailFilter] = escalation
			}
		} else {
			resp.Diagnostics.AddWarning(
				"user email not found",
				fmt.Sprintf("the specified user email '%s' was not found in the IAM users, returning empty results.", emailFilter),
			)
		}

		accessPoliciesVal, diags := types.MapValueFrom(ctx, types.BoolType, userAccessPolicies)
		resp.Diagnostics.Append(diags...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_policies"), accessPoliciesVal)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("escalations"), userEscalations)...)
		return
	}

	accessPoliciesVal, diags := types.MapValueFrom(ctx, types.BoolType, accessPolicies)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_policies"), accessPoliciesVal)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("escalations"), allEscalations)...)
}

// convertEscalationsToMap converts the escalations from the RavelinAccess
// struct to the native terraform types.
func convertEscalationsToMap(ctx context.Context, userAccess []iam.RavelinAccess, resp *datasource.ReadResponse) map[string]basetypes.MapValue {
	allEscalations := make(map[string]basetypes.MapValue, len(userAccess))

	for _, access := range userAccess {
		if len(access.Gsudo.Escalations) == 0 {
			continue
		}

		escalationMap := make(map[string]basetypes.ListValue, len(access.Gsudo.Escalations))

		for project, roles := range access.Gsudo.Escalations {
			escalationRoles := make([]types.String, len(roles))
			for i, role := range roles {
				escalationRoles[i] = types.StringValue(role)
			}

			listVal, diags := types.ListValueFrom(ctx, types.StringType, escalationRoles)
			resp.Diagnostics.Append(diags...)
			escalationMap[project] = listVal
		}

		mapVal, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, escalationMap)
		resp.Diagnostics.Append(diags...)
		allEscalations[access.Email] = mapVal
	}

	return allEscalations
}

func convertAccessPoliciesToMap(userAccess []iam.RavelinAccess) map[string]types.Bool {
	accessPolicies := make(map[string]types.Bool, len(userAccess))
	for _, access := range userAccess {
		accessPolicies[access.Email] = boolPtrToValue(access.Gsudo.AccessPolicies)
	}
	return accessPolicies
}

func boolPtrToValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*value)
}
