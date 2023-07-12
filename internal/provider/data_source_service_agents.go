package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/google"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"

	"google.golang.org/api/cloudresourcemanager/v1"
)

var _ datasource.DataSource = &ServiceAgentsDataSource{}

type ServiceAgentsDataSource struct{}

func (r *ServiceAgentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_agents"
}

func (r *ServiceAgentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "Name of the GCP project to fetch service agents for",
				Optional:            true,
			},
			"service_agent_policy": schema.StringAttribute{
				MarkdownDescription: "IAM policy JSON encoded as a string of all service agent bindings",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		MarkdownDescription: "Get all service agents contained in a project level IAM policy.\n\n" +
			"Use this data source to get all the google managed service accounts (service agents)" +
			"that have role bindings in your project level IAM policy. These bindings can then be" +
			"added to a `google_project_iam_policy` resource.",
	}
}

func (d *ServiceAgentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.ServiceAgentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	project := data.Project.ValueString()

	var c google.Config
	err := c.NewCloudResourceManagerService(ctx)
	if err != nil {
		resp.Diagnostics.AddError("error creating cloud resource manager client", err.Error())
		return
	}

	// Fetching project policy data and project number
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

	serviceAgents, err := filterPolicy(policy, projectNumber)
	if err != nil {
		resp.Diagnostics.AddError("error filtering policy", err.Error())
		return
	}

	serviceAgentPolicy, err := json.Marshal(serviceAgents)
	if err != nil {
		resp.Diagnostics.AddError("error encoding service agents to json", err.Error())
		return
	}

	data.ServiceAgentPolicy = types.StringValue(string(serviceAgentPolicy))
	data.Id = types.StringValue(strconv.FormatInt(time.Now().Unix(), 10))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// extract all google managed service account (ie service agents) from the IAM policy
func filterPolicy(policy *cloudresourcemanager.Policy, ProjectNumber string) (map[string][]string, error) {

	// the domain of service agents follow this regex (example: gcp-sa-websecurityscanner)
	// source: https://cloud.google.com/iam/docs/service-agents
	r, err := regexp.Compile("^gcp-sa-[a-z-]+$")
	if err != nil {
		return nil, err
	}

	// extra domains that don't follow the gcp-sa regex above
	// haven't found an elegant way to keep this up to date with google
	serviceAgentDomains := []string{
		"gae-api-prod",
		"gcp-gae-service",
		"cloudcomposer-accounts",
		"dlp-api",
		"dataflow-service-producer-prod",
		"cloud-filer",
		"cloud-memcache-sa",
		"cloud-redis",
		"sourcerepo-service-accounts",
		"compute-system",
		"container-analysis",
		"endpoints-portal",
		"firebase-rules",
		"dataproc-accounts",
		"gcf-admin-robot",
		"cloud-ml",
		"serverless-robot-prod",
		"containerregistry",
		"genomics-api",
		"container-engine-robot",
		"remotebuildexecution",
		"service-consumer-management",
		"service-networking",
		"cloud-tpu",
		"cloudservices",
		"repo",
	}

	// for some extra domains we want to ignore a combination of service agent and role
	serviceAgentRoles := map[string]string{
		"cloudbuild": "roles/cloudbuild.builds.builder",
	}

	// map that will contain all the service agents
	serviceAgents := map[string][]string{}

	for _, b := range policy.Bindings {
		for _, member := range b.Members {
			// only look at service accounts
			if !strings.HasPrefix(member, "serviceAccount:") {
				continue
			}

			// only look at service accounts with project number in suffix
			if !strings.Contains(strings.Split(member, "@")[0], ProjectNumber) {
				continue
			}

			// fetching domain of the service account
			domain := strings.Split(strings.Split(member, "@")[1], ".")[0]

			// either domain matches regex
			if r.MatchString(domain) {
				serviceAgents[b.Role] = append(serviceAgents[b.Role], member)
				continue
			}

			// or domain is in serviceAgentDomains
			for _, v := range serviceAgentDomains {
				if v == domain {
					serviceAgents[b.Role] = append(serviceAgents[b.Role], member)
					continue
				}
			}

			for k, v := range serviceAgentRoles {
				if k == domain && b.Role == v {
					serviceAgents[b.Role] = append(serviceAgents[b.Role], member)
					continue
				}
			}
		}
	}
	return serviceAgents, nil
}
