package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/google"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
	"google.golang.org/api/cloudresourcemanager/v1"
)

var _ datasource.DataSource = &ConditionalBindingsDataSource{}

type ConditionalBindingsDataSource struct {
	provider *ravelinProvider
}

func (r *ConditionalBindingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_conditional_bindings"
}

func (r *ConditionalBindingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "Name of the GCP project to fetch conditional bindings for. If not specified, the provider-level project will be used.",
				Optional:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role to filter conditional bindings by",
				Optional:            true,
			},
			"bindings": schema.ListNestedAttribute{
				MarkdownDescription: "Conditional bindings",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Role of the binding",
						},
						"members": schema.ListAttribute{
							Computed:            true,
							MarkdownDescription: "Members of the binding",
							ElementType:         types.StringType,
						},
						"condition": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"title": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Title of the condition",
								},
								"description": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Description of the condition",
								},
								"expression": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Expression of the condition",
								},
							},
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		MarkdownDescription: "Get all conditional bindings contained in a project level IAM policy.\n\n" +
			"Use this data source to get all the conditional bindings in your project level IAM policy, " +
			"these conditional bindings can often be managed by third party tooling outside of Terraform. " +
			"These bindings can then be added to a `google_project_iam_policy` resource.",
	}
}

func (d *ConditionalBindingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConditionalBindingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.ConditionalBindingsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	project := data.Project.ValueString()
	if project == "" && d.provider != nil {
		project = d.provider.project
	}

	if project == "" {
		resp.Diagnostics.AddError(
			"Missing Project Configuration",
			"The project attribute is required when not specified in the provider configuration.",
		)
		return
	}

	role := data.Role.ValueString()

	var c google.Config
	err := c.NewCloudResourceManagerService(ctx)
	if err != nil {
		resp.Diagnostics.AddError("error creating cloud resource manager client", err.Error())
		return
	}

	policy, err := c.GetProjectIAMPolicy(project)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("error fetching project policy %s", project), err.Error())
		return
	}

	conditionalBindings, err := filterPolicyForConditionalBindings(policy, role)
	if err != nil {
		resp.Diagnostics.AddError("error filtering policy", err.Error())
		return
	}

	data, diags := newConditionalBindingList(ctx, conditionalBindings)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func newConditionalBinding(ctx context.Context, in *cloudresourcemanager.Binding) (models.ConditionalBindingModel, diag.Diagnostics) {
	var data models.ConditionalBindingModel
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	data.Role = types.StringValue(in.Role)

	condition := models.ConditionalBindingConditionModel{
		Title:       types.StringValue(in.Condition.Title),
		Description: types.StringValue(in.Condition.Description),
		Expression:  types.StringValue(in.Condition.Expression),
	}

	data.Condition, newDiags = types.ObjectValueFrom(ctx, models.ConditionalBindingConditionAttrTypes, condition)
	diags.Append(newDiags...)

	data.Members, newDiags = types.ListValueFrom(ctx, types.StringType, in.Members)
	diags.Append(newDiags...)

	return data, diags
}

func newConditionalBindingList(ctx context.Context, in []*cloudresourcemanager.Binding) (models.ConditionalBindingsDataSourceModel, diag.Diagnostics) {
	var data models.ConditionalBindingsDataSourceModel
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	conditionalBindings := make([]models.ConditionalBindingModel, len(in))

	for i, item := range in {
		conditionalBinding, newDiags := newConditionalBinding(ctx, item)
		diags.Append(newDiags...)
		conditionalBindings[i] = conditionalBinding
	}

	data.Id = types.StringValue(strconv.FormatInt(time.Now().Unix(), 10))

	data.Bindings, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: models.ConditionalBindingAttrTypes}, conditionalBindings)
	diags.Append(newDiags...)

	return data, diags
}

// extract all conditional bindings from the IAM policy
func filterPolicyForConditionalBindings(policy *cloudresourcemanager.Policy, role string) ([]*cloudresourcemanager.Binding, error) {
	conditionalBindings := []*cloudresourcemanager.Binding{}
	for _, b := range policy.Bindings {
		if role != "" && role != b.Role {
			continue
		}

		if b.Condition == nil {
			continue
		}

		conditionalBindings = append(conditionalBindings, b)
	}
	return conditionalBindings, nil
}
