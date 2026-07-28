package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/google"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"

	"google.golang.org/api/cloudresourcemanager/v1"
)

var _ datasource.DataSource = &AgentIdentitiesDataSource{}

type AgentIdentitiesDataSource struct {
	provider *ravelinProvider
}

func (r *AgentIdentitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_identities"
}

func (r *AgentIdentitiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "Name of the GCP project to fetch agent identities for. If not specified, the provider-level project will be used.",
				Optional:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role to filter agent identity bindings by. If not specified, all roles are returned.",
				Optional:            true,
			},
			"agent_identity_policy": schema.StringAttribute{
				MarkdownDescription: "IAM policy JSON encoded as a string of all agent identity bindings",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		MarkdownDescription: "Get all agent identities contained in a project level IAM policy.\n\n" +
			"Use this data source to get all the GCP agent identities (principal sets) " +
			"that have role bindings in your project level IAM policy. These are identities " +
			"automatically created when deploying agents through the AI Platform in GCP. " +
			"These bindings can then be added to a `google_project_iam_policy` resource.",
	}
}

func (d *AgentIdentitiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AgentIdentitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.AgentIdentitiesDataSourceModel

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
	projectNumber, err := c.GetProjectIDNumber(project)
	if err != nil {
		resp.Diagnostics.AddError("error fetching project number", err.Error())
		return
	}

	role := data.Role.ValueString()

	agentIdentities, err := filterAgentIdentities(policy, projectNumber, role)
	if err != nil {
		resp.Diagnostics.AddError("error filtering policy", err.Error())
		return
	}

	agentIdentityPolicy, err := json.Marshal(agentIdentities)
	if err != nil {
		resp.Diagnostics.AddError("error encoding agent identities to json", err.Error())
		return
	}

	data.AgentIdentityPolicy = types.StringValue(string(agentIdentityPolicy))
	data.Id = types.StringValue(strconv.FormatInt(time.Now().Unix(), 10))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterAgentIdentities extracts agent identity principal sets from the IAM policy.
// These are identities created when deploying agents through the AI Platform in GCP.
// Example format: principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777
func filterAgentIdentities(policy *cloudresourcemanager.Policy, projectNumber, role string) (map[string][]string, error) {
	// Match agent identity principal sets containing the project number
	// Format: principalSet://agents.global.org-<org_number>.system.id.goog/attribute.platformContainer/aiplatform/projects/<project_number>
	pattern := fmt.Sprintf(`^principalSet://agents\.global\.org-\d+\.system\.id\.goog/attribute\.platformContainer/aiplatform/projects/%s$`, projectNumber)
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	agentIdentities := map[string][]string{}

	for _, b := range policy.Bindings {
		// Filter by role if specified
		if role != "" && b.Role != role {
			continue
		}

		for _, member := range b.Members {
			// Ignore conditional bindings
			if b.Condition != nil {
				continue
			}

			// Check if member matches the agent identity pattern
			if r.MatchString(member) {
				agentIdentities[b.Role] = append(agentIdentities[b.Role], member)
			}
		}
	}

	return agentIdentities, nil
}
