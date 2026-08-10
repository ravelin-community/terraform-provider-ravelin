package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/google"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
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
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Organization ID to fetch agent identities for. If not specified, the provider-level organization_id will be used.",
				Optional:            true,
			},
			"agent_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the agent to fetch the unique identity for. If not specified, the generic agent identity pool for the project will be returned.",
				Optional:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "GCP location of the agent (e.g. us-central1). Required when agent_name is specified. If not specified, the provider-level location will be used.",
				Optional:            true,
			},
			"identity": schema.StringAttribute{
				MarkdownDescription: "The agent identity principal",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		MarkdownDescription: "Get the GCP agent identity for a project or a specific agent.\n\n" +
			"Use this data source to get the GCP agent identity (principal or principal set) " +
			"for use in IAM bindings. When agent_name is specified, returns the unique identity " +
			"for that specific agent. Otherwise returns the generic identity pool for all agents " +
			"in the project.",
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

	organizationId := data.OrganizationId.ValueString()
	if organizationId == "" && d.provider != nil {
		organizationId = d.provider.organizationId
	}

	if organizationId == "" {
		resp.Diagnostics.AddError(
			"Missing Organization ID",
			"The organization_id attribute is required when not specified in the provider configuration.",
		)
		return
	}

	agentName := data.AgentName.ValueString()

	var c google.Config

	if err := c.NewCloudResourceManagerService(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to create Cloud Resource Manager client", err.Error())
		return
	}

	projectNumber, err := c.GetProjectIDNumber(project)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get project number", err.Error())
		return
	}

	if agentName == "" {
		identity := fmt.Sprintf("principalSet://agents.global.org-%s.system.id.goog/attribute.platformContainer/aiplatform/projects/%s", organizationId, projectNumber)
		data.Identity = types.StringValue(identity)
	} else {
		location := data.Location.ValueString()
		if location == "" && d.provider != nil {
			location = d.provider.location
		}

		if location == "" {
			resp.Diagnostics.AddError(
				"Missing Location",
				"The location attribute is required when agent_name is specified, or set at the provider level.",
			)
			return
		}

		if err := c.NewAiplatformService(ctx, location); err != nil {
			resp.Diagnostics.AddError("Failed to create AI Platform client", err.Error())
			return
		}

		engine, err := c.GetReasoningEngineByDisplayName(ctx, project, location, agentName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to find reasoning engine", err.Error())
			return
		}

		// engine.Name is projects/{project}/locations/{location}/reasoningEngines/{engine_id}
		parts := strings.Split(engine.Name, "/")
		engineID := parts[len(parts)-1]

		identity := fmt.Sprintf(
			"principal://agents.global.org-%s.system.id.goog/resources/aiplatform/projects/%s/locations/%s/reasoningEngines/%s",
			organizationId, projectNumber, location, engineID,
		)
		data.Identity = types.StringValue(identity)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
