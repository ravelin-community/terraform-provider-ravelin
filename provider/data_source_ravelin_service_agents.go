package provider

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/ravelin-community/terraform-provider-ravelin/google"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func dataSourceServiceAgents() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServiceAgentsRead,
		Schema: map[string]*schema.Schema{
			"project": {
				Type:     schema.TypeString,
				Required: true,
			},
			"service_agent_policy": {
				Computed: true,
				Type:     schema.TypeString,
			},
		},
	}
}

func dataSourceServiceAgentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics
	var c google.Config
	project := d.Get("project").(string)

	// Initialising new client
	err := c.NewCloudResourceManagerService(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Fetching project policy data and project number
	policy, err := c.GetProjectIAMPolicy(project)
	if err != nil {
		return diag.FromErr(err)
	}
	projectNumber, err := c.GetProjectIDNumber(project)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceAgents, err := filterPolicy(policy, projectNumber)
	if err != nil {
		return diag.FromErr(err)
	}

	data, _ := json.Marshal(serviceAgents)

	if err := d.Set("service_agent_policy", string(data)); err != nil {
		return diag.FromErr(err)
	}

	// always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
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
		}
	}
	return serviceAgents, nil
}
